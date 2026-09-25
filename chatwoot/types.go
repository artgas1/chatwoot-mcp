package chatwoot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// FlexTime handles Chatwoot's inconsistent time formats (RFC3339, Unix epoch int/float, null).
type FlexTime struct {
	time.Time
	Valid bool
}

func (ft *FlexTime) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == `""` || s == "0" {
		ft.Valid = false
		return nil
	}
	// Try RFC3339 string
	if len(s) > 2 && s[0] == '"' {
		var t time.Time
		if err := json.Unmarshal(data, &t); err == nil {
			ft.Time = t
			ft.Valid = true
			return nil
		}
	}
	// Try numeric (unix epoch)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		ft.Time = time.Unix(int64(f), 0)
		ft.Valid = true
		return nil
	}
	return fmt.Errorf("FlexTime: cannot parse %s", s)
}

func (ft FlexTime) MarshalJSON() ([]byte, error) {
	if !ft.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ft.Time)
}

func (ft FlexTime) String() string {
	if !ft.Valid {
		return ""
	}
	return ft.Time.Format(time.RFC3339)
}

// ---------------------------------------------------------------------------
// Conversation
// ---------------------------------------------------------------------------

// Conversation represents a Chatwoot conversation.
type Conversation struct {
	ID                   int              `json:"id"`
	UUID                 string           `json:"uuid"`
	AccountID            int              `json:"account_id"`
	InboxID              int              `json:"inbox_id"`
	Status               string           `json:"status"`
	Messages             []Message        `json:"messages"`
	UnreadCount          int              `json:"unread_count"`
	AssigneeID           *int             `json:"assignee_id"`
	Meta                 ConversationMeta `json:"meta"`
	Labels               []string         `json:"labels"`
	Priority             *string          `json:"priority"`
	CustomAttributes     map[string]any   `json:"custom_attributes"`
	AdditionalAttributes json.RawMessage  `json:"additional_attributes"`
	Muted                bool             `json:"muted"`
	CanReply             bool             `json:"can_reply"`
	SnoozedUntil         *FlexTime        `json:"snoozed_until"`
	SLAPolicyID          *int             `json:"sla_policy_id"`
	AgentLastSeenAt      FlexTime         `json:"agent_last_seen_at"`
	AssigneeLastSeenAt   FlexTime         `json:"assignee_last_seen_at"`
	ContactLastSeenAt    FlexTime         `json:"contact_last_seen_at"`
	LastActivityAt       FlexTime         `json:"last_activity_at"`
	CreatedAt            FlexTime         `json:"created_at"`
	UpdatedAt            FlexTime         `json:"updated_at"`
	Timestamp            FlexTime         `json:"timestamp"`
	FirstReplyCreatedAt  *FlexTime        `json:"first_reply_created_at"`
	WaitingSince         *FlexTime        `json:"waiting_since"`
}

// ConversationMeta holds sender, assignee and team info.
type ConversationMeta struct {
	Sender   MetaContact `json:"sender"`
	Assignee *MetaAgent  `json:"assignee"`
	Team     *MetaTeam   `json:"team"`
}

// MetaTeam is a lightweight team reference.
type MetaTeam struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MetaContact is a lightweight contact reference.
type MetaContact struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MetaAgent is a lightweight agent reference.
type MetaAgent struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ConversationListResponse is the response from list conversations.
// ListConversations wraps in {"data": {...}}, FilterConversations returns flat.
type ConversationListResponse struct {
	Data DataPayload `json:"data"`
}

// DataPayload wraps the conversation list with metadata.
type DataPayload struct {
	Meta    PaginationMeta `json:"meta"`
	Payload []Conversation `json:"payload"`
}

// ConversationFilterResponse is the flat response from filter conversations.
type ConversationFilterResponse struct {
	Meta    PaginationMeta `json:"meta"`
	Payload []Conversation `json:"payload"`
}

// FlexInt handles JSON values that can be int or string.
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		*fi = 0
		return nil
	}
	// Remove quotes if string
	if len(s) > 1 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("FlexInt: cannot parse %s", string(data))
	}
	*fi = FlexInt(n)
	return nil
}

// PaginationMeta holds pagination info.
type PaginationMeta struct {
	AllCount    FlexInt `json:"all_count"`
	Count       FlexInt `json:"count"`
	Page        FlexInt `json:"page"`
	CurrentPage FlexInt `json:"current_page"`
	TotalPage   FlexInt `json:"total_pages"`
}

// CreateConversationRequest is the payload for creating a new conversation.
type CreateConversationRequest struct {
	SourceID   *string                     `json:"source_id,omitempty"`
	InboxID    int                         `json:"inbox_id"`
	ContactID  int                         `json:"contact_id"`
	Message    *ConversationInitialMessage `json:"message,omitempty"`
	Status     *string                     `json:"status,omitempty"`
	AssigneeID *int                        `json:"assignee_id,omitempty"`
	TeamID     *int                        `json:"team_id,omitempty"`
}

// ConversationInitialMessage is the initial message when creating a conversation.
type ConversationInitialMessage struct {
	Content string `json:"content"`
}

// ConversationFilterRequest is the payload for filtering conversations.
type ConversationFilterRequest struct {
	Payload []ConversationFilterPayload `json:"payload"`
	Page    *int                        `json:"-"`
}

// ConversationFilterPayload describes a single filter condition.
type ConversationFilterPayload struct {
	AttributeKey   string  `json:"attribute_key"`
	FilterOperator string  `json:"filter_operator"`
	Values         []any   `json:"values"`
	QueryOperator  *string `json:"query_operator,omitempty"`
}

// ConversationMetaResponse is the response containing conversation counts per status.
type ConversationMetaResponse struct {
	Meta ConversationStatusCounts `json:"meta"`
}

// ConversationStatusCounts holds the count of conversations per status.
type ConversationStatusCounts struct {
	MineCount       int `json:"mine_count"`
	AssignedCount   int `json:"assigned_count"`
	UnassignedCount int `json:"unassigned_count"`
	AllCount        int `json:"all_count"`
}

// UpdateConversationRequest is the payload for PATCH /conversations/{id}.
// Chatwoot only permits `priority` on this endpoint; custom attributes must go
// through UpdateConversationCustomAttributes.
type UpdateConversationRequest struct {
	Priority *string `json:"priority,omitempty"`
}

// ConversationCustomAttributesRequest is the payload for POST /conversations/{id}/custom_attributes.
type ConversationCustomAttributesRequest struct {
	CustomAttributes map[string]any `json:"custom_attributes"`
}

// ConversationCustomAttributesResponse is the response from the custom_attributes endpoint.
type ConversationCustomAttributesResponse struct {
	CustomAttributes map[string]any `json:"custom_attributes"`
}

// TogglePriorityRequest is the payload for setting conversation priority.
// A nil Priority clears the priority (Chatwoot rejects the literal string "none").
type TogglePriorityRequest struct {
	Priority *string `json:"priority"`
}

// ToggleStatusRequest is the payload for toggling conversation status.
type ToggleStatusRequest struct {
	Status       string `json:"status"`
	SnoozedUntil *int64 `json:"snoozed_until,omitempty"`
}

// AssignConversationRequest describes an assignment change.
// Chatwoot handles only one of assignee_id / team_id per request (assignee wins
// when both keys are present, even if assignee_id is null), so the client sends
// separate requests and only includes the keys that were provided.
type AssignConversationRequest struct {
	AssigneeID *int `json:"assignee_id,omitempty"`
	TeamID     *int `json:"team_id,omitempty"`
}

// ConversationLabelsRequest is the payload for updating conversation labels.
type ConversationLabelsRequest struct {
	Labels []string `json:"labels"`
}

// ---------------------------------------------------------------------------
// Message
// ---------------------------------------------------------------------------

// Message represents a Chatwoot message.
type Message struct {
	ID                int             `json:"id"`
	AccountID         int             `json:"account_id"`
	InboxID           int             `json:"inbox_id"`
	ConversationID    int             `json:"conversation_id"`
	Content           *string         `json:"content"`
	MessageType       int             `json:"message_type"`
	ContentType       string          `json:"content_type"`
	ContentAttributes json.RawMessage `json:"content_attributes"`
	Status            string          `json:"status"`
	Private           bool            `json:"private"`
	Sender            *Sender         `json:"sender"`
	SenderType        *string         `json:"sender_type"`
	SenderID          *int            `json:"sender_id"`
	SourceID          *string         `json:"source_id"`
	Attachments       []Attachment    `json:"attachments"`
	CreatedAt         int64           `json:"created_at"`
}

// Attachment represents a file attached to a message.
type Attachment struct {
	ID            int    `json:"id"`
	MessageID     int    `json:"message_id"`
	FileType      string `json:"file_type"`
	Extension     string `json:"extension"`
	DataURL       string `json:"data_url"`
	ThumbURL      string `json:"thumb_url"`
	FileSize      int64  `json:"file_size"`
	FallbackTitle string `json:"fallback_title"`
}

// Sender represents who sent a message.
type Sender struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// MessageListResponse is the response from list messages.
type MessageListResponse struct {
	Payload []Message `json:"payload"`
}

// SendMessageRequest is the payload for sending a message.
type SendMessageRequest struct {
	Content           string          `json:"content"`
	MessageType       string          `json:"message_type"`
	Private           bool            `json:"private,omitempty"`
	ContentType       string          `json:"content_type,omitempty"`
	ContentAttributes json.RawMessage `json:"content_attributes,omitempty"`
	TemplateParams    json.RawMessage `json:"template_params,omitempty"`
}

// DeleteMessageResponse is the response after deleting a message.
type DeleteMessageResponse struct {
	Success bool `json:"success"`
}

// ---------------------------------------------------------------------------
// Contact
// ---------------------------------------------------------------------------

// Contact represents a Chatwoot contact.
type Contact struct {
	ID                   int             `json:"id"`
	Name                 string          `json:"name"`
	Email                *string         `json:"email"`
	PhoneNumber          *string         `json:"phone_number"`
	Identifier           *string         `json:"identifier"`
	Thumbnail            string          `json:"thumbnail"`
	AvailabilityStatus   string          `json:"availability_status"`
	Blocked              bool            `json:"blocked"`
	AdditionalAttributes json.RawMessage `json:"additional_attributes"`
	CreatedAt            FlexTime        `json:"created_at"`
	LastActivityAt       FlexTime        `json:"last_activity_at"`
	CustomAttributes     map[string]any  `json:"custom_attributes"`
}

// ContactListResponse is the response from list contacts.
type ContactListResponse struct {
	Payload []Contact      `json:"payload"`
	Meta    PaginationMeta `json:"meta"`
}

// ContactSearchResponse is the response from search contacts.
type ContactSearchResponse struct {
	Payload []Contact `json:"payload"`
}

// CreateContactRequest is the payload for creating a contact.
type CreateContactRequest struct {
	InboxID              int            `json:"inbox_id,omitempty"`
	Name                 string         `json:"name"`
	Email                string         `json:"email,omitempty"`
	PhoneNumber          string         `json:"phone_number,omitempty"`
	Identifier           string         `json:"identifier,omitempty"`
	Blocked              bool           `json:"blocked,omitempty"`
	Avatar               string         `json:"avatar,omitempty"`
	AvatarURL            string         `json:"avatar_url,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
	CustomAttributes     map[string]any `json:"custom_attributes,omitempty"`
}

// UpdateContactRequest is the payload for updating a contact.
type UpdateContactRequest struct {
	Name                 *string        `json:"name,omitempty"`
	Email                *string        `json:"email,omitempty"`
	PhoneNumber          *string        `json:"phone_number,omitempty"`
	Identifier           *string        `json:"identifier,omitempty"`
	Blocked              *bool          `json:"blocked,omitempty"`
	Avatar               *string        `json:"avatar,omitempty"`
	AvatarURL            *string        `json:"avatar_url,omitempty"`
	AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
	CustomAttributes     map[string]any `json:"custom_attributes,omitempty"`
}

// ContactFilterRequest is the payload for filtering contacts.
type ContactFilterRequest struct {
	Payload []ContactFilterPayload `json:"payload"`
	Page    *int                   `json:"-"`
}

// ContactFilterPayload describes a single filter condition for contacts.
type ContactFilterPayload struct {
	AttributeKey   string  `json:"attribute_key"`
	FilterOperator string  `json:"filter_operator"`
	Values         []any   `json:"values"`
	QueryOperator  *string `json:"query_operator,omitempty"`
}

// ContactConversationsResponse is the response listing conversations for a contact.
type ContactConversationsResponse struct {
	Payload []Conversation `json:"payload"`
}

// MergeContactsRequest is the payload for merging two contacts.
type MergeContactsRequest struct {
	BaseContactID   int `json:"base_contact_id"`
	MergeeContactID int `json:"mergee_contact_id"`
}

// ---------------------------------------------------------------------------
// Inbox
// ---------------------------------------------------------------------------

// Inbox represents a Chatwoot inbox.
//
// There is deliberately no "enabled" field: the Chatwoot CE inbox API does not return
// one, so a bool would always decode to false and read as "disabled" (INFRA-1340).
type Inbox struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	ChannelType         string `json:"channel_type"`
	AvatarURL           string `json:"avatar_url"`
	WebsiteURL          string `json:"website_url"`
	WidgetColor         string `json:"widget_color"`
	GreetingEnabled     bool   `json:"greeting_enabled"`
	WorkingHoursEnabled bool   `json:"working_hours_enabled"`
}

// InboxListResponse is the response from list inboxes.
type InboxListResponse struct {
	Payload []Inbox `json:"payload"`
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// Agent represents a Chatwoot agent.
type Agent struct {
	ID                 int    `json:"id"`
	AccountID          int    `json:"account_id"`
	Name               string `json:"name"`
	AvailableName      string `json:"available_name"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	AvailabilityStatus string `json:"availability_status"`
	AutoOffline        bool   `json:"auto_offline"`
	Confirmed          bool   `json:"confirmed"`
	Thumbnail          string `json:"thumbnail"`
	CustomRoleID       *int   `json:"custom_role_id"`
}

// ---------------------------------------------------------------------------
// Label
// ---------------------------------------------------------------------------

// Label represents a Chatwoot label.
type Label struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Color         string `json:"color"`
	ShowOnSidebar bool   `json:"show_on_sidebar"`
}

// ---------------------------------------------------------------------------
// Team
// ---------------------------------------------------------------------------

// Team represents a Chatwoot team.
type Team struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ---------------------------------------------------------------------------
// Canned Responses
// ---------------------------------------------------------------------------

// CannedResponse represents a saved canned response.
type CannedResponse struct {
	ID        int    `json:"id"`
	ShortCode string `json:"short_code"`
	Content   string `json:"content"`
}

// CreateCannedResponseRequest is the payload for creating a canned response.
type CreateCannedResponseRequest struct {
	ShortCode string `json:"short_code"`
	Content   string `json:"content"`
}

// UpdateCannedResponseRequest is the payload for updating a canned response.
type UpdateCannedResponseRequest struct {
	ShortCode *string `json:"short_code,omitempty"`
	Content   *string `json:"content,omitempty"`
}

// ---------------------------------------------------------------------------
// Custom Attributes
// ---------------------------------------------------------------------------

// AttributeModel is the model a custom attribute belongs to. Chatwoot returns
// it as an enum string ("conversation_attribute" / "contact_attribute") but
// older builds emitted the integer (0 / 1); both are accepted.
type AttributeModel string

const (
	AttributeModelConversation AttributeModel = "conversation_attribute"
	AttributeModelContact      AttributeModel = "contact_attribute"
)

func (m *AttributeModel) UnmarshalJSON(data []byte) error {
	s := string(data)
	switch s {
	case "null":
		*m = ""
		return nil
	case "0":
		*m = AttributeModelConversation
		return nil
	case "1":
		*m = AttributeModelContact
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("AttributeModel: cannot parse %s", s)
	}
	*m = AttributeModel(str)
	return nil
}

// IsContact reports whether the attribute applies to contacts (vs conversations).
func (m AttributeModel) IsContact() bool { return m == AttributeModelContact }

// CustomAttributeDefinition represents a custom attribute definition.
type CustomAttributeDefinition struct {
	ID                   int            `json:"id"`
	AttributeDisplayName string         `json:"attribute_display_name"`
	AttributeDisplayType string         `json:"attribute_display_type"`
	AttributeDescription string         `json:"attribute_description"`
	AttributeKey         string         `json:"attribute_key"`
	AttributeModel       AttributeModel `json:"attribute_model"`
	AttributeValues      []string       `json:"attribute_values"`
	RegexPattern         string         `json:"regex_pattern"`
	RegexCue             string         `json:"regex_cue"`
	DefaultValue         string         `json:"default_value"`
	CreatedAt            string         `json:"created_at"`
	UpdatedAt            string         `json:"updated_at"`
}

// CreateCustomAttributeRequest is the payload for creating a custom attribute definition.
type CreateCustomAttributeRequest struct {
	AttributeDisplayName string   `json:"attribute_display_name"`
	AttributeDisplayType int      `json:"attribute_display_type"`
	AttributeDescription string   `json:"attribute_description"`
	AttributeKey         string   `json:"attribute_key"`
	AttributeModel       int      `json:"attribute_model"`
	AttributeValues      []string `json:"attribute_values,omitempty"`
	RegexPattern         string   `json:"regex_pattern,omitempty"`
	RegexCue             string   `json:"regex_cue,omitempty"`
}

// ---------------------------------------------------------------------------
// Custom Filters
// ---------------------------------------------------------------------------

// CustomFilter represents a saved custom filter.
type CustomFilter struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	FilterType string          `json:"filter_type"`
	Query      json.RawMessage `json:"query"`
}

// CreateCustomFilterRequest is the payload for creating a custom filter.
type CreateCustomFilterRequest struct {
	Name       string          `json:"name"`
	FilterType string          `json:"filter_type"`
	Query      json.RawMessage `json:"query"`
}

// ---------------------------------------------------------------------------
// Automation Rules
// ---------------------------------------------------------------------------

// AutomationRule represents an automation rule.
type AutomationRule struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	EventName   string          `json:"event_name"`
	Conditions  json.RawMessage `json:"conditions"`
	Actions     json.RawMessage `json:"actions"`
	Active      bool            `json:"active"`
	CreatedAt   FlexTime        `json:"created_on"`
}

// CreateAutomationRuleRequest is the payload for creating an automation rule.
type CreateAutomationRuleRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	EventName   string          `json:"event_name"`
	Conditions  json.RawMessage `json:"conditions"`
	Actions     json.RawMessage `json:"actions"`
	Active      *bool           `json:"active,omitempty"`
}

// ---------------------------------------------------------------------------
// Webhooks
// ---------------------------------------------------------------------------

// Webhook represents a configured webhook.
type Webhook struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	AccountID     int      `json:"account_id"`
	Subscriptions []string `json:"subscriptions"`
}

// WebhookListResponse is the response from list webhooks: {"payload": {"webhooks": [...]}}.
type WebhookListResponse struct {
	Payload struct {
		Webhooks []Webhook `json:"webhooks"`
	} `json:"payload"`
}

// WebhookResponse is the response from create/update webhook: {"payload": {"webhook": {...}}}.
type WebhookResponse struct {
	Payload struct {
		Webhook Webhook `json:"webhook"`
	} `json:"payload"`
}

// CreateWebhookRequest is the payload for creating a webhook (sent wrapped as {"webhook": ...}).
type CreateWebhookRequest struct {
	Name          string   `json:"name,omitempty"`
	URL           string   `json:"url"`
	InboxID       *int     `json:"inbox_id,omitempty"`
	Subscriptions []string `json:"subscriptions,omitempty"`
}

// UpdateWebhookRequest is the payload for updating a webhook (sent wrapped as {"webhook": ...}).
type UpdateWebhookRequest struct {
	Name          *string  `json:"name,omitempty"`
	URL           *string  `json:"url,omitempty"`
	Subscriptions []string `json:"subscriptions,omitempty"`
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

// ReportSummary contains aggregate metrics for a report.
// Average times are seconds; Chatwoot returns them as JSON numbers (or null).
type ReportSummary struct {
	ConversationsCount    int                    `json:"conversations_count"`
	IncomingMessagesCount int                    `json:"incoming_messages_count"`
	OutgoingMessagesCount int                    `json:"outgoing_messages_count"`
	AvgFirstResponseTime  *float64               `json:"avg_first_response_time"`
	AvgResolutionTime     *float64               `json:"avg_resolution_time"`
	ResolutionsCount      int                    `json:"resolutions_count"`
	AvgReplyTime          *float64               `json:"reply_time"`
	Previous              *ReportSummaryPrevious `json:"previous,omitempty"`
}

// ReportSummaryPrevious contains the previous period's metrics for comparison.
type ReportSummaryPrevious struct {
	ConversationsCount    int      `json:"conversations_count"`
	IncomingMessagesCount int      `json:"incoming_messages_count"`
	OutgoingMessagesCount int      `json:"outgoing_messages_count"`
	AvgFirstResponseTime  *float64 `json:"avg_first_response_time"`
	AvgResolutionTime     *float64 `json:"avg_resolution_time"`
	ResolutionsCount      int      `json:"resolutions_count"`
	AvgReplyTime          *float64 `json:"reply_time"`
}

// SummaryReportEntry contains report metrics for an agent, inbox, or team.
// Average times are seconds; Chatwoot returns them as JSON numbers (or null).
type SummaryReportEntry struct {
	ID                         int      `json:"id"`
	ConversationsCount         int      `json:"conversations_count"`
	ResolvedConversationsCount int      `json:"resolved_conversations_count"`
	AvgResolutionTime          *float64 `json:"avg_resolution_time"`
	AvgFirstResponseTime       *float64 `json:"avg_first_response_time"`
	AvgReplyTime               *float64 `json:"avg_reply_time"`
}

// ChannelSummary contains conversation counts by status for one channel type.
// The channel summary endpoint returns a map keyed by channel type
// (e.g. "Channel::Email") whose values have this shape.
type ChannelSummary struct {
	Open     int `json:"open"`
	Resolved int `json:"resolved"`
	Pending  int `json:"pending"`
	Snoozed  int `json:"snoozed"`
	Total    int `json:"total"`
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

// Profile represents the authenticated user's profile.
type Profile struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	AccountID int    `json:"account_id"`
	AvatarURL string `json:"avatar_url"`
}

// ---------------------------------------------------------------------------
// Team Members
// ---------------------------------------------------------------------------

// TeamMember represents a member of a team (same structure as Agent).
type TeamMember struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	AvailabilityStatus string `json:"availability_status"`
}

// AddTeamMemberRequest is the payload for adding members to a team.
type AddTeamMemberRequest struct {
	UserIDs []int `json:"user_ids"`
}

// ---------------------------------------------------------------------------
// Inbox Members
// ---------------------------------------------------------------------------

// InboxMember represents a member of an inbox (same structure as Agent).
type InboxMember struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	AvailabilityStatus string `json:"availability_status"`
}

// AddInboxMemberRequest is the payload for adding members to an inbox.
type AddInboxMemberRequest struct {
	UserIDs []int `json:"user_ids"`
}

// ---------------------------------------------------------------------------
// Help Center - Portals
// ---------------------------------------------------------------------------

// Portal represents a help center portal.
type Portal struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ---------------------------------------------------------------------------
// Help Center - Articles
// ---------------------------------------------------------------------------

// Article represents a help center article.
type Article struct {
	ID          int              `json:"id"`
	Title       string           `json:"title"`
	Content     *string          `json:"content"`
	Description *string          `json:"description"`
	Slug        string           `json:"slug"`
	Status      string           `json:"status"`
	Position    int              `json:"position"`
	Views       int              `json:"views"`
	AccountID   int              `json:"account_id"`
	Category    *ArticleCategory `json:"category"`
	Author      *MetaAgent       `json:"author"`
	Meta        json.RawMessage  `json:"meta"`
}

// ArticleCategory is the category reference embedded in an article.
type ArticleCategory struct {
	ID     *int    `json:"id"`
	Name   *string `json:"name"`
	Slug   *string `json:"slug"`
	Locale *string `json:"locale"`
}

// ArticleResponse wraps a single article: {"payload": {...}}.
type ArticleResponse struct {
	Payload Article `json:"payload"`
}

// CategoryResponse wraps a single category: {"payload": {...}}.
type CategoryResponse struct {
	Payload Category `json:"payload"`
}

// ---------------------------------------------------------------------------
// Help Center - Categories
// ---------------------------------------------------------------------------

// Category represents a help center category.
type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Locale      string `json:"locale"`
	Position    int    `json:"position"`
	AccountID   int    `json:"account_id"`
}

// CreateArticleRequest is the payload for creating an article.
type CreateArticleRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	CategoryID  *int   `json:"category_id,omitempty"`
	AuthorID    *int   `json:"author_id,omitempty"`
}

// UpdateArticleRequest is the payload for updating an article.
type UpdateArticleRequest struct {
	Title       *string `json:"title,omitempty"`
	Content     *string `json:"content,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	CategoryID  *int    `json:"category_id,omitempty"`
}

// UpdatePortalRequest is the payload for updating a portal.
type UpdatePortalRequest struct {
	Name *string `json:"name,omitempty"`
	Slug *string `json:"slug,omitempty"`
}

// CreateCategoryRequest is the payload for creating a category.
type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Description string `json:"description,omitempty"`
	Locale      string `json:"locale,omitempty"`
	Position    *int   `json:"position,omitempty"`
	ParentID    *int   `json:"parent_category_id,omitempty"`
}

// UpdateCategoryRequest is the payload for updating a category.
type UpdateCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Locale      *string `json:"locale,omitempty"`
	Position    *int    `json:"position,omitempty"`
}

// ---------------------------------------------------------------------------
// Notifications
// ---------------------------------------------------------------------------

// Notification represents a Chatwoot notification.
type Notification struct {
	ID               int      `json:"id"`
	NotificationType string   `json:"notification_type"`
	PrimaryActorType string   `json:"primary_actor_type"`
	PrimaryActorID   int      `json:"primary_actor_id"`
	ReadAt           FlexTime `json:"read_at"`
	CreatedAt        FlexTime `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Account
// ---------------------------------------------------------------------------

// Account represents a Chatwoot account.
// Auto-resolve configuration lives under Settings (auto_resolve_after in minutes,
// auto_resolve_message, auto_resolve_ignore_waiting).
type Account struct {
	ID               int             `json:"id"`
	Name             string          `json:"name"`
	Locale           string          `json:"locale"`
	Domain           string          `json:"domain"`
	SupportEmail     string          `json:"support_email"`
	Status           string          `json:"status"`
	CreatedAt        string          `json:"created_at"`
	Features         json.RawMessage `json:"features"`
	Settings         map[string]any  `json:"settings"`
	CustomAttributes map[string]any  `json:"custom_attributes"`
}

// UpdateAccountRequest is the payload for updating an account.
type UpdateAccountRequest struct {
	Name             *string `json:"name,omitempty"`
	Locale           *string `json:"locale,omitempty"`
	Domain           *string `json:"domain,omitempty"`
	SupportEmail     *string `json:"support_email,omitempty"`
	AutoResolveAfter *int    `json:"auto_resolve_after,omitempty"`
}

// ---------------------------------------------------------------------------
// Audit Log
// ---------------------------------------------------------------------------

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID             int             `json:"id"`
	AuditableID    int             `json:"auditable_id"`
	AuditableType  string          `json:"auditable_type"`
	Auditable      json.RawMessage `json:"auditable"`
	Action         string          `json:"action"`
	AuditedChanges json.RawMessage `json:"audited_changes"`
	AssociatedID   *int            `json:"associated_id"`
	AssociatedType *string         `json:"associated_type"`
	UserID         *int            `json:"user_id"`
	UserType       string          `json:"user_type"`
	Username       *string         `json:"username"`
	Version        int             `json:"version"`
	Comment        *string         `json:"comment"`
	RequestUUID    string          `json:"request_uuid"`
	RemoteAddress  *string         `json:"remote_address"`
	CreatedAt      FlexTime        `json:"created_at"`
}

// AuditLogListResponse is the response from list audit logs.
type AuditLogListResponse struct {
	AuditLogs    []AuditLog `json:"audit_logs"`
	CurrentPage  int        `json:"current_page"`
	PerPage      int        `json:"per_page"`
	TotalEntries int        `json:"total_entries"`
}

// ---------------------------------------------------------------------------
// Agent Bot
// ---------------------------------------------------------------------------

// AgentBot represents an agent bot.
type AgentBot struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	OutgoingURL string          `json:"outgoing_url"`
	BotType     string          `json:"bot_type"`
	BotConfig   json.RawMessage `json:"bot_config"`
	AccountID   int             `json:"account_id"`
	Thumbnail   string          `json:"thumbnail"`
	AccessToken string          `json:"access_token"`
	SystemBot   bool            `json:"system_bot"`
	CreatedAt   FlexTime        `json:"created_at"`
	UpdatedAt   FlexTime        `json:"updated_at"`
}

// CreateAgentBotRequest is the payload for creating an agent bot.
type CreateAgentBotRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OutgoingURL string `json:"outgoing_url"`
	BotType     string `json:"bot_type,omitempty"`
}

// UpdateAgentBotRequest is the payload for updating an agent bot.
type UpdateAgentBotRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	OutgoingURL *string `json:"outgoing_url,omitempty"`
	BotType     *string `json:"bot_type,omitempty"`
}

// ---------------------------------------------------------------------------
// Integration
// ---------------------------------------------------------------------------

// IntegrationApp represents an available integration.
type IntegrationApp struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
	HookType    string            `json:"hook_type"`
	Hooks       []IntegrationHook `json:"hooks"`
}

// IntegrationHook represents an integration hook instance.
type IntegrationHook struct {
	ID          int             `json:"id"`
	AppID       string          `json:"app_id"`
	InboxID     *int            `json:"inbox_id"`
	AccountID   int             `json:"account_id"`
	Status      string          `json:"status"`
	HookType    string          `json:"hook_type"`
	Settings    json.RawMessage `json:"settings"`
	ReferenceID *string         `json:"reference_id"`
	CreatedAt   FlexTime        `json:"created_at"`
}

// CreateIntegrationHookRequest is the payload for creating an integration hook.
type CreateIntegrationHookRequest struct {
	AppID    string          `json:"app_id"`
	InboxID  *int            `json:"inbox_id,omitempty"`
	Settings json.RawMessage `json:"settings,omitempty"`
}

// UpdateIntegrationHookRequest is the payload for updating an integration hook.
type UpdateIntegrationHookRequest struct {
	Settings json.RawMessage `json:"settings,omitempty"`
}
