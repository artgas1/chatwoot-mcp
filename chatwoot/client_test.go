package chatwoot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// newTestClient returns a Client pointed at a test server that records every
// request and answers with the given JSON body.
func newTestClient(t *testing.T, responseBody string) (*Client, *[]recordedRequest) {
	t.Helper()
	var requests []recordedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("api_access_token") != "token" {
			t.Errorf("missing api_access_token header")
		}
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body)})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(srv.Close)
	return &Client{baseURL: srv.URL, apiToken: "token", accountID: 1, httpClient: srv.Client()}, &requests
}

func TestAssignConversationSendsOnlyProvidedKeys(t *testing.T) {
	ctx := context.Background()
	team, agent := 5, 7

	c, reqs := newTestClient(t, `{}`)
	if err := c.AssignConversation(ctx, 1643, AssignConversationRequest{TeamID: &team}); err != nil {
		t.Fatal(err)
	}
	if len(*reqs) != 1 || (*reqs)[0].Body != `{"team_id":5}` {
		t.Errorf("team-only assignment must not include assignee_id: %+v", *reqs)
	}

	c, reqs = newTestClient(t, `{}`)
	if err := c.AssignConversation(ctx, 1643, AssignConversationRequest{AssigneeID: &agent, TeamID: &team}); err != nil {
		t.Fatal(err)
	}
	if len(*reqs) != 2 || (*reqs)[0].Body != `{"assignee_id":7}` || (*reqs)[1].Body != `{"team_id":5}` {
		t.Errorf("agent+team must be two requests: %+v", *reqs)
	}

	c, reqs = newTestClient(t, `{}`)
	if err := c.AssignConversation(ctx, 1643, AssignConversationRequest{}); err != nil {
		t.Fatal(err)
	}
	if len(*reqs) != 1 || (*reqs)[0].Body != `{"assignee_id":null}` {
		t.Errorf("no ids must unassign the agent: %+v", *reqs)
	}
}

func TestTogglePriorityNoneSendsNull(t *testing.T) {
	c, reqs := newTestClient(t, ``)
	if err := c.TogglePriority(context.Background(), 1, "none"); err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Body != `{"priority":null}` {
		t.Errorf("body = %s", (*reqs)[0].Body)
	}
	c, reqs = newTestClient(t, ``)
	if err := c.TogglePriority(context.Background(), 1, "urgent"); err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Body != `{"priority":"urgent"}` {
		t.Errorf("body = %s", (*reqs)[0].Body)
	}
}

func TestUpdateConversationCustomAttributesUsesDedicatedEndpoint(t *testing.T) {
	c, reqs := newTestClient(t, `{"custom_attributes":{"slack_ts":"1"}}`)
	attrs, err := c.UpdateConversationCustomAttributes(context.Background(), 1643, map[string]any{"slack_ts": "1"})
	if err != nil {
		t.Fatal(err)
	}
	r := (*reqs)[0]
	if r.Method != http.MethodPost || r.Path != "/api/v1/accounts/1/conversations/1643/custom_attributes" {
		t.Errorf("unexpected request: %+v", r)
	}
	if r.Body != `{"custom_attributes":{"slack_ts":"1"}}` {
		t.Errorf("body = %s", r.Body)
	}
	if attrs["slack_ts"] != "1" {
		t.Errorf("attrs = %v", attrs)
	}
}

func TestGetMessagesPagination(t *testing.T) {
	c, reqs := newTestClient(t, `{"meta":{},"payload":[]}`)
	if _, err := c.GetMessages(context.Background(), 1643, 100, 0); err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Query != "before=100" {
		t.Errorf("query = %q", (*reqs)[0].Query)
	}
	c, reqs = newTestClient(t, `{"meta":{},"payload":[]}`)
	if _, err := c.GetMessages(context.Background(), 1643, 0, 0); err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Query != "" {
		t.Errorf("query = %q", (*reqs)[0].Query)
	}
}

func TestMergeContactsParsesBareContact(t *testing.T) {
	c, _ := newTestClient(t, `{"id":42,"name":"Alice","email":"a@example.test"}`)
	contact, err := c.MergeContacts(context.Background(), MergeContactsRequest{BaseContactID: 42, MergeeContactID: 43})
	if err != nil {
		t.Fatal(err)
	}
	if contact.ID != 42 || contact.Name != "Alice" {
		t.Errorf("contact = %+v", contact)
	}
}

func TestWebhooksWrapRequestAndUnwrapResponse(t *testing.T) {
	c, reqs := newTestClient(t, `{"payload":{"webhook":{"id":9,"url":"https://h.test","subscriptions":["message_created"]}}}`)
	w, err := c.CreateWebhook(context.Background(), CreateWebhookRequest{URL: "https://h.test", Subscriptions: []string{"message_created"}})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte((*reqs)[0].Body), &body)
	if _, ok := body["webhook"]; !ok {
		t.Errorf("request must be wrapped in webhook: %s", (*reqs)[0].Body)
	}
	if w.ID != 9 {
		t.Errorf("webhook = %+v", w)
	}

	c, _ = newTestClient(t, `{"payload":{"webhooks":[{"id":1,"url":"https://a"},{"id":2,"url":"https://b"}]}}`)
	list, err := c.ListWebhooks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("list = %+v", list)
	}
}

func TestHelpCenterUsesPortalSlugAndWrapsParams(t *testing.T) {
	c, reqs := newTestClient(t, `{"payload":{"id":3,"title":"Hi","status":"draft"}}`)
	a, err := c.CreateArticle(context.Background(), "my-portal", CreateArticleRequest{Title: "Hi", Content: "body"})
	if err != nil {
		t.Fatal(err)
	}
	r := (*reqs)[0]
	if r.Path != "/api/v1/accounts/1/portals/my-portal/articles" {
		t.Errorf("path = %s", r.Path)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(r.Body), &body)
	if _, ok := body["article"]; !ok {
		t.Errorf("request must be wrapped in article: %s", r.Body)
	}
	if a.ID != 3 || a.Title != "Hi" {
		t.Errorf("article = %+v", a)
	}

	c, reqs = newTestClient(t, `{"payload":[]}`)
	if _, err := c.ListCategories(context.Background(), "my-portal"); err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Path != "/api/v1/accounts/1/portals/my-portal/categories" {
		t.Errorf("path = %s", (*reqs)[0].Path)
	}

	c, reqs = newTestClient(t, `{"id":1,"name":"P","slug":"new-slug"}`)
	p, err := c.UpdatePortal(context.Background(), "my-portal", UpdatePortalRequest{Slug: strPtr("new-slug")})
	if err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Path != "/api/v1/accounts/1/portals/my-portal" || p.Slug != "new-slug" {
		t.Errorf("request %+v portal %+v", (*reqs)[0], p)
	}
}

func TestGetChannelSummaryReturnsMap(t *testing.T) {
	c, reqs := newTestClient(t, `{"Channel::Email":{"open":1,"resolved":2,"pending":0,"snoozed":0,"total":3}}`)
	m, err := c.GetChannelSummary(context.Background(), 100, 200)
	if err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Path != "/api/v2/accounts/1/summary_reports/channel" || m["Channel::Email"].Total != 3 {
		t.Errorf("request %+v result %+v", (*reqs)[0], m)
	}
}

func TestDoReportsHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Resource could not be found"}`))
	}))
	defer srv.Close()
	c := &Client{baseURL: srv.URL, apiToken: "token", accountID: 1, httpClient: srv.Client()}
	if _, err := c.GetConversation(context.Background(), 1); err == nil {
		t.Fatal("expected error for 404")
	}
}

func strPtr(s string) *string { return &s }
