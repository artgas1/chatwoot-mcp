package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gobenpark/chatwoot-mcp/chatwoot"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Input types ---

type GetReportsSummaryInput struct {
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}

type GetAgentSummaryInput struct {
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}

type GetTeamSummaryInput struct {
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}

type GetInboxSummaryInput struct {
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}

type GetChannelSummaryInput struct {
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}

// fmtMetric renders a Chatwoot average metric (seconds, or nil when no data) as a duration.
func fmtMetric(v *float64) string {
	if v == nil {
		return "N/A"
	}
	return fmtDuration(*v)
}

// fmtDuration renders seconds as a compact human-readable duration.
func fmtDuration(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second)).Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
}

func formatSummaryEntry(sb *strings.Builder, label string, e chatwoot.SummaryReportEntry) {
	sb.WriteString(fmt.Sprintf("- [%d] %s\n", e.ID, label))
	sb.WriteString(fmt.Sprintf("    Conversations: %d, Resolved: %d\n", e.ConversationsCount, e.ResolvedConversationsCount))
	sb.WriteString(fmt.Sprintf("    Avg FRT: %s, Avg Resolution: %s\n", fmtMetric(e.AvgFirstResponseTime), fmtMetric(e.AvgResolutionTime)))
}

// RegisterReportTools registers report-related tools on the MCP server.
func RegisterReportTools(server *mcp.Server, client *chatwoot.Client) {

	// --- get_reports_summary ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_reports_summary",
		Description: "Get account-level report summary with metrics like avg first response time, avg resolution time, conversations count, and message counts. Provide since/until as dates (YYYY-MM-DD). Defaults to last 7 days.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetReportsSummaryInput) (*mcp.CallToolResult, any, error) {
		since, until := parseDateRange(input.Since, input.Until)
		summary, err := client.GetReportsSummary(ctx, since, until, "account")
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString("Account Report Summary\n")
		sb.WriteString(fmt.Sprintf("Period: %s to %s\n\n", time.Unix(since, 0).Format("2006-01-02"), time.Unix(until, 0).Format("2006-01-02")))
		sb.WriteString(fmt.Sprintf("Conversations: %d\n", summary.ConversationsCount))
		sb.WriteString(fmt.Sprintf("Resolutions: %d\n", summary.ResolutionsCount))
		sb.WriteString(fmt.Sprintf("Incoming messages: %d\n", summary.IncomingMessagesCount))
		sb.WriteString(fmt.Sprintf("Outgoing messages: %d\n", summary.OutgoingMessagesCount))
		sb.WriteString(fmt.Sprintf("Avg first response time: %s\n", fmtMetric(summary.AvgFirstResponseTime)))
		sb.WriteString(fmt.Sprintf("Avg resolution time: %s\n", fmtMetric(summary.AvgResolutionTime)))
		sb.WriteString(fmt.Sprintf("Avg reply time: %s\n", fmtMetric(summary.AvgReplyTime)))
		if summary.Previous != nil {
			p := summary.Previous
			sb.WriteString(fmt.Sprintf("\nPrevious period (same length): %d conversations, %d resolutions, %d incoming / %d outgoing messages\n",
				p.ConversationsCount, p.ResolutionsCount, p.IncomingMessagesCount, p.OutgoingMessagesCount))
			sb.WriteString(fmt.Sprintf("  Avg first response: %s, Avg resolution: %s, Avg reply: %s\n",
				fmtMetric(p.AvgFirstResponseTime), fmtMetric(p.AvgResolutionTime), fmtMetric(p.AvgReplyTime)))
		}
		return textResult(sb.String()), nil, nil
	})

	// --- get_agent_summary ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_agent_summary",
		Description: "Get per-agent performance metrics including conversations count, response time, and resolution time. Provide since/until as dates (YYYY-MM-DD). Defaults to last 7 days.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetAgentSummaryInput) (*mcp.CallToolResult, any, error) {
		since, until := parseDateRange(input.Since, input.Until)
		entries, err := client.GetAgentSummary(ctx, since, until)
		if err != nil {
			return errorResult(err), nil, nil
		}
		// Build agent name map
		agentNames := map[int]string{}
		if agents, err := client.ListAgents(ctx); err == nil {
			for _, a := range agents {
				agentNames[a.ID] = fmt.Sprintf("%s <%s>", a.Name, a.Email)
			}
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Agent Summary (%s to %s)\n\n", time.Unix(since, 0).Format("2006-01-02"), time.Unix(until, 0).Format("2006-01-02")))
		for _, e := range entries {
			label := agentNames[e.ID]
			if label == "" {
				label = fmt.Sprintf("Agent #%d", e.ID)
			}
			formatSummaryEntry(&sb, label, e)
		}
		if len(entries) == 0 {
			sb.WriteString("No agent data available.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- get_team_summary ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_team_summary",
		Description: "Get per-team performance metrics. Provide since/until as dates (YYYY-MM-DD). Defaults to last 7 days.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetTeamSummaryInput) (*mcp.CallToolResult, any, error) {
		since, until := parseDateRange(input.Since, input.Until)
		entries, err := client.GetTeamSummary(ctx, since, until)
		if err != nil {
			return errorResult(err), nil, nil
		}
		// Build team name map
		teamNames := map[int]string{}
		if teams, err := client.ListTeams(ctx); err == nil {
			for _, t := range teams {
				teamNames[t.ID] = t.Name
			}
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Team Summary (%s to %s)\n\n", time.Unix(since, 0).Format("2006-01-02"), time.Unix(until, 0).Format("2006-01-02")))
		for _, e := range entries {
			label := teamNames[e.ID]
			if label == "" {
				label = fmt.Sprintf("Team #%d", e.ID)
			}
			formatSummaryEntry(&sb, label, e)
		}
		if len(entries) == 0 {
			sb.WriteString("No team data available.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- get_inbox_summary ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_inbox_summary",
		Description: "Get per-inbox performance metrics. Provide since/until as dates (YYYY-MM-DD). Defaults to last 7 days.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetInboxSummaryInput) (*mcp.CallToolResult, any, error) {
		since, until := parseDateRange(input.Since, input.Until)
		entries, err := client.GetInboxSummary(ctx, since, until)
		if err != nil {
			return errorResult(err), nil, nil
		}
		// Build inbox name map
		inboxNames := map[int]string{}
		if inboxes, err := client.ListInboxes(ctx); err == nil {
			for _, i := range inboxes {
				inboxNames[i.ID] = i.Name
			}
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Inbox Summary (%s to %s)\n\n", time.Unix(since, 0).Format("2006-01-02"), time.Unix(until, 0).Format("2006-01-02")))
		for _, e := range entries {
			label := inboxNames[e.ID]
			if label == "" {
				label = fmt.Sprintf("Inbox #%d", e.ID)
			}
			formatSummaryEntry(&sb, label, e)
		}
		if len(entries) == 0 {
			sb.WriteString("No inbox data available.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- get_channel_summary ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_channel_summary",
		Description: "Get conversation counts by status (open/resolved/pending/snoozed/total) grouped by channel type (Channel::Email, Channel::WebWidget, Channel::Api, ...) for conversations created in the period. Provide since/until as dates (YYYY-MM-DD, max 6 months apart). Defaults to last 7 days.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetChannelSummaryInput) (*mcp.CallToolResult, any, error) {
		since, until := parseDateRange(input.Since, input.Until)
		channels, err := client.GetChannelSummary(ctx, since, until)
		if err != nil {
			return errorResult(err), nil, nil
		}
		names := make([]string, 0, len(channels))
		for name := range channels {
			names = append(names, name)
		}
		sort.Strings(names)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Channel Summary (%s to %s)\n\n", time.Unix(since, 0).Format("2006-01-02"), time.Unix(until, 0).Format("2006-01-02")))
		for _, name := range names {
			ch := channels[name]
			sb.WriteString(fmt.Sprintf("- %s: total %d (open: %d, resolved: %d, pending: %d, snoozed: %d)\n",
				name, ch.Total, ch.Open, ch.Resolved, ch.Pending, ch.Snoozed))
		}
		if len(channels) == 0 {
			sb.WriteString("No conversations were created in this period.")
		}
		return textResult(sb.String()), nil, nil
	})
}

func parseDateRange(sinceStr, untilStr string) (int64, int64) {
	now := time.Now()
	until := now.Unix()
	since := now.AddDate(0, 0, -7).Unix()

	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t.Unix()
		}
	}
	if untilStr != "" {
		if t, err := time.Parse("2006-01-02", untilStr); err == nil {
			until = t.Unix()
		}
	}
	return since, until
}
