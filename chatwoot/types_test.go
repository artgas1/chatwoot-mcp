package chatwoot

import (
	"encoding/json"
	"testing"
)

// Fixtures below mirror real Chatwoot v4.12.1 responses.

func TestReportSummaryParsesNumericAverages(t *testing.T) {
	body := `{"conversations_count":343,"incoming_messages_count":465,"outgoing_messages_count":415,
	"avg_first_response_time":737033.0477707007,"avg_resolution_time":null,"resolutions_count":731,
	"reply_time":531967.0967741936,"previous":{"conversations_count":364,"incoming_messages_count":476,
	"outgoing_messages_count":317,"avg_first_response_time":367057.184,"avg_resolution_time":555443.6877470355,
	"resolutions_count":253,"reply_time":228810.9846153846}}`
	var s ReportSummary
	if err := json.Unmarshal([]byte(body), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.AvgFirstResponseTime == nil || *s.AvgFirstResponseTime < 737033 {
		t.Errorf("avg_first_response_time = %v", s.AvgFirstResponseTime)
	}
	if s.AvgResolutionTime != nil {
		t.Errorf("null avg_resolution_time should stay nil, got %v", *s.AvgResolutionTime)
	}
	if s.AvgReplyTime == nil {
		t.Errorf("reply_time should be parsed into AvgReplyTime")
	}
	if s.Previous == nil || s.Previous.ConversationsCount != 364 || s.Previous.AvgReplyTime == nil {
		t.Errorf("previous period not parsed: %+v", s.Previous)
	}
}

func TestSummaryReportEntryParsesNumericAverages(t *testing.T) {
	body := `[{"id":3,"conversations_count":114,"resolved_conversations_count":241,
	"avg_resolution_time":1777414.5643153526,"avg_first_response_time":null,"avg_reply_time":286776.06976744183}]`
	var entries []SummaryReportEntry
	if err := json.Unmarshal([]byte(body), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != 3 || entries[0].AvgResolutionTime == nil || entries[0].AvgFirstResponseTime != nil {
		t.Errorf("unexpected entries: %+v", entries)
	}
}

func TestChannelSummaryIsMapKeyedByChannel(t *testing.T) {
	body := `{"Channel::Email":{"open":59,"resolved":276,"pending":8,"snoozed":0,"total":343}}`
	var channels map[string]ChannelSummary
	if err := json.Unmarshal([]byte(body), &channels); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := channels["Channel::Email"]; got.Total != 343 || got.Resolved != 276 {
		t.Errorf("unexpected channel stats: %+v", got)
	}
}

func TestWebhookListResponseShape(t *testing.T) {
	body := `{"payload":{"webhooks":[{"id":2,"name":"","url":"https://example.test/hook","account_id":1,
	"subscriptions":["conversation_created","conversation_status_changed"],"secret":"x"}]}}`
	var resp WebhookListResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Payload.Webhooks) != 1 || resp.Payload.Webhooks[0].URL != "https://example.test/hook" {
		t.Errorf("unexpected webhooks: %+v", resp.Payload.Webhooks)
	}
}

func TestAttributeModelAcceptsStringAndInt(t *testing.T) {
	cases := map[string]AttributeModel{
		`"contact_attribute"`:      AttributeModelContact,
		`"conversation_attribute"`: AttributeModelConversation,
		`1`:                        AttributeModelContact,
		`0`:                        AttributeModelConversation,
		`null`:                     "",
	}
	for raw, want := range cases {
		var m AttributeModel
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Errorf("%s: %v", raw, err)
			continue
		}
		if m != want {
			t.Errorf("%s: got %q want %q", raw, m, want)
		}
	}
	if !AttributeModelContact.IsContact() || AttributeModelConversation.IsContact() {
		t.Errorf("IsContact misclassifies models")
	}
}

func TestCustomFilterUsesFilterTypeKey(t *testing.T) {
	var f CustomFilter
	if err := json.Unmarshal([]byte(`{"id":1,"name":"x","filter_type":"conversation","query":{"payload":[]}}`), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if f.FilterType != "conversation" {
		t.Errorf("filter_type = %q", f.FilterType)
	}
}

func TestMessageParsesAttachments(t *testing.T) {
	body := `{"id":10,"content":null,"message_type":0,"created_at":1783872552,"private":false,
	"sender":{"id":981,"name":"avellia","type":"contact"},
	"attachments":[{"id":5,"message_id":10,"file_type":"image","extension":"png","data_url":"http://x/img.png","thumb_url":"http://x/t.png","file_size":1234}]}`
	var m Message
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m.Attachments) != 1 || m.Attachments[0].FileType != "image" || m.Attachments[0].DataURL == "" {
		t.Errorf("attachments not parsed: %+v", m.Attachments)
	}
}

func TestConversationParsesTeamAndFlexTimes(t *testing.T) {
	body := `{"id":1643,"status":"pending","inbox_id":6,"messages":[],"unread_count":0,
	"meta":{"sender":{"id":981,"name":"avellia"},"assignee":{"id":1,"name":"ben"},"team":{"id":1,"name":"develop"}},
	"labels":["bug"],"priority":null,"custom_attributes":{"slack_ts":"1783872552.229419"},
	"snoozed_until":null,"created_at":1783872550,"updated_at":1787894264.123,"timestamp":1787894264,
	"first_reply_created_at":0,"waiting_since":0,"agent_last_seen_at":1787894264}`
	var c Conversation
	if err := json.Unmarshal([]byte(body), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Meta.Team == nil || c.Meta.Team.Name != "develop" {
		t.Errorf("team not parsed: %+v", c.Meta.Team)
	}
	if !c.CreatedAt.Valid || !c.UpdatedAt.Valid || c.FirstReplyCreatedAt.Valid {
		t.Errorf("flex times: created=%v updated=%v firstReply=%v", c.CreatedAt.Valid, c.UpdatedAt.Valid, c.FirstReplyCreatedAt.Valid)
	}
}

func TestPaginationMetaAcceptsStringPage(t *testing.T) {
	var m PaginationMeta
	if err := json.Unmarshal([]byte(`{"count":42,"current_page":"3"}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Count != 42 || m.CurrentPage != 3 {
		t.Errorf("unexpected meta: %+v", m)
	}
}
