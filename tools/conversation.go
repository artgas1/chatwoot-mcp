package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gobenpark/chatwoot-mcp/chatwoot"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Input types ---

type ListConversationsInput struct {
	Status       string   `json:"status,omitempty"`
	AssigneeType string   `json:"assignee_type,omitempty"`
	Q            string   `json:"q,omitempty"`
	InboxID      int      `json:"inbox_id,omitempty"`
	TeamID       int      `json:"team_id,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Page         int      `json:"page,omitempty"`
}

type GetConversationInput struct {
	ConversationID int `json:"conversation_id"`
}

type CreateConversationInput struct {
	InboxID    int    `json:"inbox_id"`
	ContactID  int    `json:"contact_id"`
	Message    string `json:"message,omitempty"`
	Status     string `json:"status,omitempty"`
	AssigneeID int    `json:"assignee_id,omitempty"`
	TeamID     int    `json:"team_id,omitempty"`
}

type FilterConversationsInput struct {
	Payload []FilterPayloadInput `json:"payload"`
	Page    int                  `json:"page,omitempty"`
}

type FilterPayloadInput struct {
	AttributeKey   string   `json:"attribute_key"`
	FilterOperator string   `json:"filter_operator"`
	Values         []string `json:"values"`
	QueryOperator  string   `json:"query_operator,omitempty"`
}

type GetConversationMetaInput struct{}

type UpdateConversationInput struct {
	ConversationID   int            `json:"conversation_id"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

type GetMessagesInput struct {
	ConversationID int `json:"conversation_id"`
	Before         int `json:"before,omitempty"`
	After          int `json:"after,omitempty"`
}

type SendMessageInput struct {
	ConversationID int    `json:"conversation_id"`
	Content        string `json:"content"`
	MessageType    string `json:"message_type,omitempty"`
	Private        bool   `json:"private,omitempty"`
}

type DeleteMessageInput struct {
	ConversationID int `json:"conversation_id"`
	MessageID      int `json:"message_id"`
}

type ToggleStatusInput struct {
	ConversationID int    `json:"conversation_id"`
	Status         string `json:"status"`
}

type TogglePriorityInput struct {
	ConversationID int    `json:"conversation_id"`
	Priority       string `json:"priority"`
}

type AssignConversationInput struct {
	ConversationID int  `json:"conversation_id"`
	AssigneeID     *int `json:"assignee_id,omitempty"`
	TeamID         *int `json:"team_id,omitempty"`
}

type UpdateLabelsInput struct {
	ConversationID int      `json:"conversation_id"`
	Labels         []string `json:"labels"`
}

// RegisterConversationTools registers all conversation-related tools on the MCP server.
func RegisterConversationTools(server *mcp.Server, client *chatwoot.Client) {

	// --- list_conversations ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_conversations",
		Description: "List conversations in Chatwoot (25 per page, default status=open). Filters: status (open/resolved/pending/snoozed/all), assignee_type (me/assigned/unassigned/all), q (full-text search over message content; note Chatwoot ignores the status filter when q is set), inbox_id, team_id, labels (array of label names), page.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListConversationsInput) (*mcp.CallToolResult, any, error) {
		resp, err := client.ListConversations(ctx, input.Status, input.AssigneeType, input.Q, input.InboxID, input.TeamID, input.Labels, input.Page)
		if err != nil {
			return errorResult(err), nil, nil
		}
		page := input.Page
		if page < 1 {
			page = 1
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Page %d: %d shown of %d matching conversations\n\n", page, len(resp.Data.Payload), resp.Data.Meta.AllCount))
		for _, conv := range resp.Data.Payload {
			sb.WriteString(formatConversationLine(conv))
		}
		if len(resp.Data.Payload) == 0 {
			sb.WriteString("No conversations found.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- get_conversation ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_conversation",
		Description: "Get detailed information about a specific conversation by its ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetConversationInput) (*mcp.CallToolResult, any, error) {
		conv, err := client.GetConversation(ctx, input.ConversationID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Conversation #%d\n", conv.ID))
		sb.WriteString(fmt.Sprintf("Status: %s\n", conv.Status))
		sb.WriteString(fmt.Sprintf("Inbox ID: %d\n", conv.InboxID))
		sb.WriteString(fmt.Sprintf("Contact: %s (ID: %d)\n", conv.Meta.Sender.Name, conv.Meta.Sender.ID))
		if conv.Meta.Assignee != nil {
			sb.WriteString(fmt.Sprintf("Assignee: %s (ID: %d)\n", conv.Meta.Assignee.Name, conv.Meta.Assignee.ID))
		} else {
			sb.WriteString("Assignee: (unassigned)\n")
		}
		if conv.Meta.Team != nil {
			sb.WriteString(fmt.Sprintf("Team: %s (ID: %d)\n", conv.Meta.Team.Name, conv.Meta.Team.ID))
		}
		if conv.Priority != nil {
			sb.WriteString(fmt.Sprintf("Priority: %s\n", *conv.Priority))
		}
		if len(conv.Labels) > 0 {
			sb.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(conv.Labels, ", ")))
		}
		sb.WriteString(fmt.Sprintf("Unread messages: %d (use get_messages for the message list)\n", conv.UnreadCount))
		if conv.CreatedAt.Valid {
			sb.WriteString(fmt.Sprintf("Created: %s\n", conv.CreatedAt.Format(time.RFC3339)))
		}
		if conv.LastActivityAt.Valid {
			sb.WriteString(fmt.Sprintf("Last activity: %s\n", conv.LastActivityAt.Format(time.RFC3339)))
		}
		if len(conv.CustomAttributes) > 0 {
			sb.WriteString("Custom attributes:\n")
			for k, v := range conv.CustomAttributes {
				sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
			}
		}
		return textResult(sb.String()), nil, nil
	})

	// --- create_conversation ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_conversation",
		Description: "Create a new conversation. Requires inbox_id and contact_id. Optionally provide an initial message, status, assignee_id, or team_id.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateConversationInput) (*mcp.CallToolResult, any, error) {
		mbReq := chatwoot.CreateConversationRequest{
			InboxID:   input.InboxID,
			ContactID: input.ContactID,
		}
		if input.Status != "" {
			mbReq.Status = &input.Status
		}
		if input.Message != "" {
			mbReq.Message = &chatwoot.ConversationInitialMessage{Content: input.Message}
		}
		if input.AssigneeID > 0 {
			mbReq.AssigneeID = &input.AssigneeID
		}
		if input.TeamID > 0 {
			mbReq.TeamID = &input.TeamID
		}
		conv, err := client.CreateConversation(ctx, mbReq)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Conversation created! #%d (inbox: %d, status: %s)", conv.ID, conv.InboxID, conv.Status)), nil, nil
	})

	// --- filter_conversations ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "filter_conversations",
		Description: "Filter conversations using advanced criteria. Each filter has attribute_key (status, assignee_id, inbox_id, team_id, label, priority, created_at, last_activity_at, etc.), filter_operator (equal_to, not_equal_to, contains, is_greater_than, is_less_than, days_before, etc.), values (array), and query_operator (AND/OR) to chain filters.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input FilterConversationsInput) (*mcp.CallToolResult, any, error) {
		payload := make([]chatwoot.ConversationFilterPayload, len(input.Payload))
		for i, p := range input.Payload {
			values := make([]any, len(p.Values))
			for j, v := range p.Values {
				if n, err := strconv.Atoi(v); err == nil {
					values[j] = n
				} else {
					values[j] = v
				}
			}
			fp := chatwoot.ConversationFilterPayload{
				AttributeKey:   p.AttributeKey,
				FilterOperator: p.FilterOperator,
				Values:         values,
			}
			if p.QueryOperator != "" {
				fp.QueryOperator = &p.QueryOperator
			}
			payload[i] = fp
		}
		filterReq := chatwoot.ConversationFilterRequest{
			Payload: payload,
		}
		if input.Page > 0 {
			filterReq.Page = &input.Page
		}
		resp, err := client.FilterConversations(ctx, filterReq)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Filter results: %d matching conversations, %d shown on this page\n\n", resp.Meta.AllCount, len(resp.Payload)))
		for _, conv := range resp.Payload {
			sb.WriteString(formatConversationLine(conv))
		}
		if len(resp.Payload) == 0 {
			sb.WriteString("No conversations match the filter.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- get_conversation_counts ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_conversation_counts",
		Description: "Get conversation counts grouped by status (open, pending, resolved, snoozed, all).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetConversationMetaInput) (*mcp.CallToolResult, any, error) {
		meta, err := client.GetConversationMeta(ctx)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString("Conversation counts:\n")
		sb.WriteString(fmt.Sprintf("  All: %d\n", meta.Meta.AllCount))
		sb.WriteString(fmt.Sprintf("  Mine: %d\n", meta.Meta.MineCount))
		sb.WriteString(fmt.Sprintf("  Assigned: %d\n", meta.Meta.AssignedCount))
		sb.WriteString(fmt.Sprintf("  Unassigned: %d\n", meta.Meta.UnassignedCount))
		return textResult(sb.String()), nil, nil
	})

	// --- update_conversation ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_conversation",
		Description: "Replace a conversation's custom attributes. Provide the full map of custom_attributes to store (keys not included are removed).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateConversationInput) (*mcp.CallToolResult, any, error) {
		if input.CustomAttributes == nil {
			return errorResult(fmt.Errorf("custom_attributes is required")), nil, nil
		}
		attrs, err := client.UpdateConversationCustomAttributes(ctx, input.ConversationID, input.CustomAttributes)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Conversation #%d custom attributes updated:\n", input.ConversationID))
		for k, v := range attrs {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
		if len(attrs) == 0 {
			sb.WriteString("  (none)\n")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- get_messages ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_messages",
		Description: "Get messages in a conversation, oldest first. Without a cursor Chatwoot returns only the latest 20 messages. To page backwards pass before=<oldest message id shown>; to fetch newer messages pass after=<message id> (up to 100). Returns id, timestamp, sender, type, attachments and content.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetMessagesInput) (*mcp.CallToolResult, any, error) {
		messages, err := client.GetMessages(ctx, input.ConversationID, input.Before, input.After)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Messages in conversation #%d (%d returned", input.ConversationID, len(messages)))
		if input.Before == 0 && input.After == 0 {
			sb.WriteString("; latest 20 max, pass before=<message id> for older ones")
		}
		sb.WriteString("):\n\n")

		for _, msg := range messages {
			msgType := messageTypeName(msg.MessageType)
			senderName := "(system)"
			if msg.Sender != nil {
				senderName = msg.Sender.Name
			}
			content := "(no content)"
			if msg.Content != nil && *msg.Content != "" {
				content = *msg.Content
			}
			private := ""
			if msg.Private {
				private = " [private]"
			}
			ts := time.Unix(msg.CreatedAt, 0).Format("2006-01-02 15:04")
			sb.WriteString(fmt.Sprintf("[%s] (id %d) %s (%s)%s: %s\n", ts, msg.ID, senderName, msgType, private, content))
			for _, att := range msg.Attachments {
				label := att.FileType
				if att.FallbackTitle != "" {
					label += " " + att.FallbackTitle
				}
				if att.DataURL != "" {
					label += " " + att.DataURL
				}
				sb.WriteString(fmt.Sprintf("    [attachment: %s]\n", label))
			}
		}
		if len(messages) == 0 {
			sb.WriteString("No messages found.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- send_message ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "send_message",
		Description: "Send a message to a conversation. message_type: 'outgoing' (to customer) or 'incoming'. Set private=true for internal notes visible only to agents.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SendMessageInput) (*mcp.CallToolResult, any, error) {
		msgType := input.MessageType
		if msgType == "" {
			msgType = "outgoing"
		}
		msg, err := client.SendMessage(ctx, input.ConversationID, chatwoot.SendMessageRequest{
			Content:     input.Content,
			MessageType: msgType,
			Private:     input.Private,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Message sent! (ID: %d, conversation: #%d)", msg.ID, input.ConversationID)), nil, nil
	})

	// --- delete_message ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_message",
		Description: "Delete a specific message from a conversation.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteMessageInput) (*mcp.CallToolResult, any, error) {
		if err := client.DeleteMessage(ctx, input.ConversationID, input.MessageID); err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Message #%d deleted from conversation #%d.", input.MessageID, input.ConversationID)), nil, nil
	})

	// --- toggle_conversation_status ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "toggle_conversation_status",
		Description: "Change the status of a conversation. Valid statuses: open, resolved, pending, snoozed.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ToggleStatusInput) (*mcp.CallToolResult, any, error) {
		if err := client.ToggleStatus(ctx, input.ConversationID, input.Status); err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Conversation #%d status changed to '%s'!", input.ConversationID, input.Status)), nil, nil
	})

	// --- toggle_conversation_priority ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "toggle_conversation_priority",
		Description: "Set the priority of a conversation. Valid priorities: urgent, high, medium, low. Pass 'none' to clear the priority.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input TogglePriorityInput) (*mcp.CallToolResult, any, error) {
		if err := client.TogglePriority(ctx, input.ConversationID, input.Priority); err != nil {
			return errorResult(err), nil, nil
		}
		if input.Priority == "" || input.Priority == "none" {
			return textResult(fmt.Sprintf("Conversation #%d priority cleared.", input.ConversationID)), nil, nil
		}
		return textResult(fmt.Sprintf("Conversation #%d priority set to '%s'!", input.ConversationID, input.Priority)), nil, nil
	})

	// --- assign_conversation ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "assign_conversation",
		Description: "Assign a conversation to an agent (assignee_id) and/or a team (team_id). Providing only team_id leaves the current agent untouched. Call with neither to unassign the agent.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AssignConversationInput) (*mcp.CallToolResult, any, error) {
		if err := client.AssignConversation(ctx, input.ConversationID, chatwoot.AssignConversationRequest{
			AssigneeID: input.AssigneeID,
			TeamID:     input.TeamID,
		}); err != nil {
			return errorResult(err), nil, nil
		}
		result := fmt.Sprintf("Conversation #%d assignment updated!", input.ConversationID)
		if input.AssigneeID != nil {
			result += fmt.Sprintf(" Agent ID: %d", *input.AssigneeID)
		}
		if input.TeamID != nil {
			result += fmt.Sprintf(" Team ID: %d", *input.TeamID)
		}
		return textResult(result), nil, nil
	})

	// --- update_conversation_labels ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_conversation_labels",
		Description: "Update labels on a conversation. Provide the full list of labels to set (replaces existing).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateLabelsInput) (*mcp.CallToolResult, any, error) {
		if err := client.UpdateConversationLabels(ctx, input.ConversationID, input.Labels); err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Conversation #%d labels updated to: %s", input.ConversationID, strings.Join(input.Labels, ", "))), nil, nil
	})
}

// formatConversationLine renders one conversation as a list entry.
func formatConversationLine(conv chatwoot.Conversation) string {
	assignee := "(unassigned)"
	if conv.Meta.Assignee != nil {
		assignee = conv.Meta.Assignee.Name
	}
	labels := ""
	if len(conv.Labels) > 0 {
		labels = " [" + strings.Join(conv.Labels, ", ") + "]"
	}
	priority := ""
	if conv.Priority != nil && *conv.Priority != "" {
		priority = " !" + *conv.Priority
	}
	return fmt.Sprintf("- #%d [%s]%s %s → %s%s (unread: %d)\n",
		conv.ID, conv.Status, priority, conv.Meta.Sender.Name, assignee, labels, conv.UnreadCount)
}

func messageTypeName(t int) string {
	switch t {
	case 0:
		return "incoming"
	case 1:
		return "outgoing"
	case 2:
		return "activity"
	default:
		return fmt.Sprintf("type-%d", t)
	}
}
