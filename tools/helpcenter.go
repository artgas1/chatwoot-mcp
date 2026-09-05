package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/gobenpark/chatwoot-mcp/chatwoot"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Input types ---
//
// Chatwoot's help center API addresses portals by slug, not by numeric ID.
// Every portal-scoped input therefore takes portal_slug; portal_id is accepted
// as a convenience and resolved to the slug via list_portals.

type ListPortalsInput struct{}

type PortalRef struct {
	PortalSlug string `json:"portal_slug,omitempty"`
	PortalID   int    `json:"portal_id,omitempty"`
}

type UpdatePortalInput struct {
	PortalRef
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type ListArticlesInput struct {
	PortalRef
}

type CreateArticleInput struct {
	PortalRef
	Title       string `json:"title"`
	Content     string `json:"content"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	CategoryID  *int   `json:"category_id,omitempty"`
	AuthorID    *int   `json:"author_id,omitempty"`
}

type UpdateArticleInput struct {
	PortalRef
	ArticleID   int    `json:"article_id"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	CategoryID  *int   `json:"category_id,omitempty"`
}

type DeleteArticleInput struct {
	PortalRef
	ArticleID int `json:"article_id"`
}

type ListCategoriesInput struct {
	PortalRef
}

type CreateCategoryInput struct {
	PortalRef
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Description string `json:"description,omitempty"`
	Locale      string `json:"locale,omitempty"`
	Position    *int   `json:"position,omitempty"`
	ParentID    *int   `json:"parent_category_id,omitempty"`
}

type UpdateCategoryInput struct {
	PortalRef
	CategoryID  int    `json:"category_id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Locale      string `json:"locale,omitempty"`
	Position    *int   `json:"position,omitempty"`
}

type DeleteCategoryInput struct {
	PortalRef
	CategoryID int `json:"category_id"`
}

const portalRefHelp = "Identify the portal with portal_slug (preferred; see list_portals) or portal_id."

// resolvePortalSlug returns the portal slug for a PortalRef, looking the slug up
// by ID when only portal_id was provided.
func resolvePortalSlug(ctx context.Context, client *chatwoot.Client, ref PortalRef) (string, error) {
	if ref.PortalSlug != "" {
		return ref.PortalSlug, nil
	}
	if ref.PortalID <= 0 {
		return "", fmt.Errorf("portal_slug (or portal_id) is required")
	}
	portals, err := client.ListPortals(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve portal %d: %w", ref.PortalID, err)
	}
	for _, p := range portals {
		if p.ID == ref.PortalID {
			return p.Slug, nil
		}
	}
	return "", fmt.Errorf("portal with id %d not found (use list_portals)", ref.PortalID)
}

// RegisterHelpCenterTools registers help center (portals, articles, categories) tools.
func RegisterHelpCenterTools(server *mcp.Server, client *chatwoot.Client) {

	// --- list_portals ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_portals",
		Description: "List all help center portals in the account. Other help center tools address a portal by its slug.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListPortalsInput) (*mcp.CallToolResult, any, error) {
		portals, err := client.ListPortals(ctx)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		for _, p := range portals {
			sb.WriteString(fmt.Sprintf("- [%d] %s (slug: %s)\n", p.ID, p.Name, p.Slug))
		}
		if sb.Len() == 0 {
			sb.WriteString("No portals found.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- update_portal ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_portal",
		Description: "Update a help center portal's name or slug. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdatePortalInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		updateReq := chatwoot.UpdatePortalRequest{}
		if input.Name != "" {
			updateReq.Name = &input.Name
		}
		if input.Slug != "" {
			updateReq.Slug = &input.Slug
		}
		portal, err := client.UpdatePortal(ctx, slug, updateReq)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Portal #%d updated! Name: %s, Slug: %s", portal.ID, portal.Name, portal.Slug)), nil, nil
	})

	// --- list_articles ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_articles",
		Description: "List articles in a help center portal. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListArticlesInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		articles, err := client.ListArticles(ctx, slug)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Articles in portal %q:\n\n", slug))
		for _, a := range articles {
			category := ""
			if a.Category != nil && a.Category.Name != nil && *a.Category.Name != "" {
				category = " — category: " + *a.Category.Name
			}
			sb.WriteString(fmt.Sprintf("- [%d] %s (status: %s, views: %d)%s\n", a.ID, a.Title, a.Status, a.Views, category))
		}
		if len(articles) == 0 {
			sb.WriteString("No articles found.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- create_article ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_article",
		Description: "Create a new article in a help center portal. Requires title and content; status is draft, published or archived (default draft). " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateArticleInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		article, err := client.CreateArticle(ctx, slug, chatwoot.CreateArticleRequest{
			Title:       input.Title,
			Content:     input.Content,
			Description: input.Description,
			Status:      input.Status,
			CategoryID:  input.CategoryID,
			AuthorID:    input.AuthorID,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Article created! ID: %d, Title: %s, Status: %s", article.ID, article.Title, article.Status)), nil, nil
	})

	// --- update_article ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_article",
		Description: "Update an article in a help center portal. Provide only fields you want to change. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateArticleInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		updateReq := chatwoot.UpdateArticleRequest{
			CategoryID: input.CategoryID,
		}
		if input.Title != "" {
			updateReq.Title = &input.Title
		}
		if input.Content != "" {
			updateReq.Content = &input.Content
		}
		if input.Description != "" {
			updateReq.Description = &input.Description
		}
		if input.Status != "" {
			updateReq.Status = &input.Status
		}
		article, err := client.UpdateArticle(ctx, slug, input.ArticleID, updateReq)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Article #%d updated! Title: %s, Status: %s", article.ID, article.Title, article.Status)), nil, nil
	})

	// --- delete_article ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_article",
		Description: "Delete an article from a help center portal. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteArticleInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		if err := client.DeleteArticle(ctx, slug, input.ArticleID); err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Article #%d deleted from portal %q.", input.ArticleID, slug)), nil, nil
	})

	// --- list_categories ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_categories",
		Description: "List categories in a help center portal. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListCategoriesInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		categories, err := client.ListCategories(ctx, slug)
		if err != nil {
			return errorResult(err), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Categories in portal %q:\n\n", slug))
		for _, c := range categories {
			sb.WriteString(fmt.Sprintf("- [%d] %s (slug: %s, locale: %s)\n", c.ID, c.Name, c.Slug, c.Locale))
			if c.Description != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", c.Description))
			}
		}
		if len(categories) == 0 {
			sb.WriteString("No categories found.")
		}
		return textResult(sb.String()), nil, nil
	})

	// --- create_category ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_category",
		Description: "Create a new category in a help center portal. Requires name; optional slug, description, locale, position, parent_category_id. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateCategoryInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		category, err := client.CreateCategory(ctx, slug, chatwoot.CreateCategoryRequest{
			Name:        input.Name,
			Slug:        input.Slug,
			Description: input.Description,
			Locale:      input.Locale,
			Position:    input.Position,
			ParentID:    input.ParentID,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Category created! ID: %d, Name: %s, Slug: %s", category.ID, category.Name, category.Slug)), nil, nil
	})

	// --- update_category ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_category",
		Description: "Update a category in a help center portal. Provide only fields you want to change. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateCategoryInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		updateReq := chatwoot.UpdateCategoryRequest{
			Position: input.Position,
		}
		if input.Name != "" {
			updateReq.Name = &input.Name
		}
		if input.Description != "" {
			updateReq.Description = &input.Description
		}
		if input.Locale != "" {
			updateReq.Locale = &input.Locale
		}
		category, err := client.UpdateCategory(ctx, slug, input.CategoryID, updateReq)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Category #%d updated! Name: %s", category.ID, category.Name)), nil, nil
	})

	// --- delete_category ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_category",
		Description: "Delete a category from a help center portal. " + portalRefHelp,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteCategoryInput) (*mcp.CallToolResult, any, error) {
		slug, err := resolvePortalSlug(ctx, client, input.PortalRef)
		if err != nil {
			return errorResult(err), nil, nil
		}
		if err := client.DeleteCategory(ctx, slug, input.CategoryID); err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(fmt.Sprintf("Category #%d deleted from portal %q.", input.CategoryID, slug)), nil, nil
	})
}
