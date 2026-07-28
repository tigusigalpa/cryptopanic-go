package cryptopanic

import (
	"context"
	"encoding/json"
	"fmt"
)

// Posts retrieves a paginated list of news posts from the CryptoPanic
// API (GET /posts/). The query object controls filtering, currencies,
// regions, kind, pagination, and enterprise-gated parameters.
//
// The auth_token is injected from the client configuration and never
// passed in the query object. Pagination URLs in the response are
// redacted: any auth_token parameter is removed before exposure.
//
// Documentation: https://cryptopanic.com/developers/api/
func (c *Client) Posts(ctx context.Context, query PostsQuery) (*PostsPage, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	values := query.ToValues()

	var raw postsPageRaw
	if _, err := c.doGet(ctx, c.apiPath("/posts/"), values, &raw); err != nil {
		return nil, err
	}

	page := &PostsPage{
		Results: raw.Results,
	}

	var nextURL, prevURL string
	if raw.Next != nil && *raw.Next != "" {
		nextURL = redactURL(*raw.Next)
		page.HasNext = true
		page.NextPage = parsePageFromURL(nextURL)
	}
	if raw.Previous != nil && *raw.Previous != "" {
		prevURL = redactURL(*raw.Previous)
		page.HasPrevious = true
		page.PreviousPage = parsePageFromURL(prevURL)
	}
	page.Next = nextURL
	page.Previous = prevURL

	return page, nil
}

// Portfolio retrieves the authenticated user's portfolio from the
// CryptoPanic API (GET /portfolio/). This endpoint is available under
// Growth and Enterprise plans according to the public reference.
//
// Because the public API reference does not specify a complete
// response schema, the response is returned as an opaque
// PortfolioResponse containing the raw decoded JSON. No domain model
// is fabricated.
//
// Documentation: https://cryptopanic.com/developers/api/
func (c *Client) Portfolio(ctx context.Context) (*PortfolioResponse, error) {
	var raw json.RawMessage
	if _, err := c.doGet(ctx, c.apiPath("/portfolio/"), nil, &raw); err != nil {
		return nil, err
	}
	return &PortfolioResponse{Raw: raw}, nil
}

// PostsRSS retrieves the posts endpoint with format=rss, returning
// raw RSS XML content. The reference says format=rss returns 20
// items regardless of plan. RSS content is preserved as XML and is
// not coerced into the JSON post page type.
//
// Documentation: https://cryptopanic.com/developers/api/
func (c *Client) PostsRSS(ctx context.Context, query PostsQuery) (*RSSResponse, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	values := query.ToValues()
	values.Set("format", "rss")

	body, _, err := c.doGetRaw(ctx, c.apiPath("/posts/"), values)
	if err != nil {
		return nil, err
	}

	return &RSSResponse{Body: body}, nil
}

// NewsRSS retrieves the public CryptoPanic RSS news feed
// (GET /news/rss/). This is a public endpoint that does not require
// authentication, but the auth_token is still sent as documented.
// Returns raw RSS XML content.
//
// Documentation: https://cryptopanic.com/developers/api/
func (c *Client) NewsRSS(ctx context.Context) (*RSSResponse, error) {
	body, _, err := c.doGetRaw(ctx, c.apiPath("/news/rss/"), nil)
	if err != nil {
		return nil, fmt.Errorf("cryptopanic: failed to fetch news RSS: %w", err)
	}

	return &RSSResponse{Body: body}, nil
}
