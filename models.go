package cryptopanic

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Filter represents a known posts filter value. The CryptoPanic API
// documents filters such as rising, hot, bullish, bearish, important,
// saved, and lol. Custom string values are allowed for forward
// compatibility.
type Filter string

// Filter values identify the documented post-filter options.
const (
	FilterRising    Filter = "rising"
	FilterHot       Filter = "hot"
	FilterBullish   Filter = "bullish"
	FilterBearish   Filter = "bearish"
	FilterImportant Filter = "important"
	FilterSaved     Filter = "saved"
	FilterLol       Filter = "lol"
)

// Kind represents a known posts kind value.
type Kind string

// Kind values identify the documented post kinds.
const (
	KindNews  Kind = "news"
	KindMedia Kind = "media"
	KindAll   Kind = "all"
)

// PanicPeriod represents a known panic period value.
type PanicPeriod string

// PanicPeriod values identify the supported panic-score time windows.
const (
	PanicPeriod1h  PanicPeriod = "1h"
	PanicPeriod6h  PanicPeriod = "6h"
	PanicPeriod24h PanicPeriod = "24h"
)

// PanicSort represents a known panic sort direction.
type PanicSort string

// PanicSort values identify the supported panic-score sort directions.
const (
	PanicSortAsc  PanicSort = "asc"
	PanicSortDesc PanicSort = "desc"
)

// PostsQuery holds the query parameters for the GET /posts/ endpoint.
// Authentication is configured on the Client, not in the query object.
// Only clear client-side syntax and cross-field rules are validated;
// the CryptoPanic server remains the authority for plan eligibility
// and evolving allowed values.
type PostsQuery struct {
	// Public is an optional boolean for public mode.
	Public *bool
	// Currencies is an optional list encoded as a comma-separated
	// set, e.g. BTC,ETH.
	Currencies []string
	// Regions is an optional region code or supported region
	// collection. Default is "en".
	Regions string
	// Filter is an optional known filter. Custom values are allowed.
	Filter Filter
	// Kind is an optional known kind. Default is "all".
	Kind Kind
	// Following is an optional boolean for strictly private usage.
	Following *bool
	// LastPull is an optional RFC 3339 / ISO-8601 timestamp.
	// Enterprise-gated in the reference.
	LastPull *time.Time
	// PanicPeriod is an optional panic period (1h, 6h, 24h).
	// Enterprise-gated in the reference.
	PanicPeriod PanicPeriod
	// PanicSort is an optional sort direction (asc, desc). Rejected
	// locally when PanicPeriod is absent. Enterprise-gated.
	PanicSort PanicSort
	// Size is an optional integer in the inclusive range 1–50.
	// Enterprise-gated in the reference.
	Size *int
	// Page is an optional integer in the inclusive range 1–50.
	// Default is 1.
	Page *int
	// WithContent is an optional boolean.
	// Enterprise-gated in the reference.
	WithContent *bool
	// Search is an optional non-empty search string.
	// Enterprise-gated in the reference.
	Search string
}

// Validate checks local client-side syntax and cross-field rules.
// The CryptoPanic server remains the authority for plan eligibility.
func (q *PostsQuery) Validate() error {
	if q == nil {
		return nil
	}
	if q.PanicSort != "" && q.PanicPeriod == "" {
		return ErrInvalidQuery
	}
	if q.Size != nil {
		if *q.Size < 1 || *q.Size > 50 {
			return ErrInvalidQuery
		}
	}
	if q.Page != nil {
		if *q.Page < 1 || *q.Page > 50 {
			return ErrInvalidQuery
		}
	}
	if q.Search != "" && strings.TrimSpace(q.Search) == "" {
		return ErrInvalidQuery
	}
	return nil
}

// ToValues converts the PostsQuery into url.Values suitable for use
// with the HTTP transport. Zero/empty values are omitted.
func (q *PostsQuery) ToValues() url.Values {
	v := url.Values{}
	if q == nil {
		return v
	}
	if q.Public != nil {
		v.Set("public", strconv.FormatBool(*q.Public))
	}
	if len(q.Currencies) > 0 {
		v.Set("currencies", strings.Join(q.Currencies, ","))
	}
	if q.Regions != "" {
		v.Set("regions", q.Regions)
	}
	if q.Filter != "" {
		v.Set("filter", string(q.Filter))
	}
	if q.Kind != "" {
		v.Set("kind", string(q.Kind))
	}
	if q.Following != nil {
		v.Set("following", strconv.FormatBool(*q.Following))
	}
	if q.LastPull != nil {
		v.Set("last_pull", q.LastPull.Format(time.RFC3339))
	}
	if q.PanicPeriod != "" {
		v.Set("panic_period", string(q.PanicPeriod))
	}
	if q.PanicSort != "" {
		v.Set("panic_sort", string(q.PanicSort))
	}
	if q.Size != nil {
		v.Set("size", strconv.Itoa(*q.Size))
	}
	if q.Page != nil {
		v.Set("page", strconv.Itoa(*q.Page))
	}
	if q.WithContent != nil {
		v.Set("with_content", strconv.FormatBool(*q.WithContent))
	}
	if q.Search != "" {
		v.Set("search", q.Search)
	}
	return v
}

// PostSource represents the source object nested in a Post.
type PostSource struct {
	// Title is the display title of the source.
	Title string `json:"title"`
	// Region is the region code of the source.
	Region string `json:"region"`
	// Domain is the domain name of the source.
	Domain string `json:"domain"`
	// Path is the URL path of the source article.
	Path string `json:"path"`
	// CreatedAt is the source creation timestamp.
	CreatedAt *time.Time `json:"created_at"`
}

// PostInstrument represents a currency instrument tagged on a post.
type PostInstrument struct {
	// ID is the numeric instrument identifier.
	ID int `json:"id"`
	// Code is the instrument ticker code, e.g. BTC.
	Code string `json:"code"`
	// Slug is the URL-friendly instrument slug.
	Slug string `json:"slug"`
	// Title is the display name of the instrument.
	Title string `json:"title"`
	// Volume is the 24-hour trading volume.
	Volume float64 `json:"volume"`
	// Change is the 24-hour price change percentage.
	Change float64 `json:"change"`
	// Currency is the currency code.
	Currency string `json:"currency"`
}

// PostVotes represents the vote counts on a post.
type PostVotes struct {
	// Positive is the count of positive votes.
	Positive int `json:"positive"`
	// Negative is the count of negative votes.
	Negative int `json:"negative"`
	// Important is the count of important votes.
	Important int `json:"important"`
	// Liked is the count of liked votes.
	Liked int `json:"liked"`
	// Disliked is the count of disliked votes.
	Disliked int `json:"disliked"`
	// Lol is the count of lol votes.
	Lol int `json:"lol"`
	// Toxic is the count of toxic votes.
	Toxic int `json:"toxic"`
	// Comments is the count of comments.
	Comments int `json:"comments"`
	// Saved is the count of saved votes.
	Saved int `json:"saved"`
}

// PostContent represents the content object nested in a post.
// This field is enterprise-gated and may be null or absent.
type PostContent struct {
	// Raw is the raw content text.
	Raw string `json:"raw"`
}

// PostAuthor represents the author of a post.
type PostAuthor struct {
	// ID is the numeric author identifier.
	ID int `json:"id"`
	// Name is the display name of the author.
	Name string `json:"name"`
	// Slug is the URL-friendly author slug.
	Slug string `json:"slug"`
	// URL is the author's profile URL.
	URL string `json:"url"`
	// Avatar is the author's avatar URL.
	Avatar string `json:"avatar"`
	// Twitter is the author's Twitter handle.
	Twitter string `json:"twitter"`
	// Facebook is the author's Facebook profile.
	Facebook string `json:"facebook"`
	// LinkedIn is the author's LinkedIn profile.
	LinkedIn string `json:"linkedin"`
}

// Post represents a single news post in the CryptoPanic API response.
// Plan-gated or potentially omitted fields are nullable/optional.
// Additive fields sent by the service are not rejected.
type Post struct {
	// ID is the unique numeric post identifier.
	ID int `json:"id"`
	// Slug is the URL-friendly post slug.
	Slug string `json:"slug"`
	// Title is the post title.
	Title string `json:"title"`
	// Description is the post description/summary.
	Description string `json:"description"`
	// PublishedAt is the publication timestamp.
	PublishedAt *time.Time `json:"published_at"`
	// CreatedAt is the post creation timestamp.
	CreatedAt *time.Time `json:"created_at"`
	// Kind is the post kind (news, media).
	Kind string `json:"kind"`
	// Source is the source object.
	Source *PostSource `json:"source"`
	// OriginalURL is the original article URL.
	OriginalURL string `json:"original_url"`
	// URL is the CryptoPanic post URL.
	URL string `json:"url"`
	// Image is the post image URL (optional).
	Image *string `json:"image"`
	// Instruments is the list of tagged instruments.
	Instruments []PostInstrument `json:"instruments"`
	// Votes is the vote counts object.
	Votes *PostVotes `json:"votes"`
	// PanicScore is the overall panic score (optional/enterprise-gated).
	PanicScore *int `json:"panic_score"`
	// PanicScore1h is the 1-hour panic score (optional/enterprise-gated).
	PanicScore1h *int `json:"panic_score_1h"`
	// Author is the post author (optional).
	Author *PostAuthor `json:"author"`
	// Content is the post content (optional/enterprise-gated).
	Content *PostContent `json:"content"`
}

// PostsPage represents the paginated envelope returned by GET /posts/.
// The Next and Previous fields are redacted: any auth_token query
// parameter is removed before exposure.
type PostsPage struct {
	// Results is the list of posts on the current page.
	Results []Post `json:"results"`
	// Next is the redacted URL for the next page, or empty if there
	// is no next page.
	Next string `json:"-"`
	// Previous is the redacted URL for the previous page, or empty if
	// there is no previous page.
	Previous string `json:"-"`
	// HasNext indicates whether a next page exists.
	HasNext bool `json:"-"`
	// HasPrevious indicates whether a previous page exists.
	HasPrevious bool `json:"-"`
	// NextPage extracts the page number from the next URL, if available.
	NextPage *int `json:"-"`
	// PreviousPage extracts the page number from the previous URL, if available.
	PreviousPage *int `json:"-"`
}

// postsPageRaw is used for initial JSON decoding. Next and Previous
// are pointers to strings because the API returns null when there is
// no adjacent page.
type postsPageRaw struct {
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []Post  `json:"results"`
}

// PortfolioResponse is an opaque/generic wrapper for the GET
// /portfolio/ endpoint. The public API reference does not specify a
// complete response schema, so the raw decoded JSON is retained
// instead of fabricating a domain model. Future schema expansion
// is backward compatible because callers access data through the
// Raw field.
type PortfolioResponse struct {
	// Raw contains the original decoded JSON from the portfolio
	// endpoint response.
	Raw json.RawMessage `json:"-"`
}

// RSSResponse represents a raw RSS feed response. The body is
// preserved as raw text/XML rather than being coerced into the JSON
// post page type.
type RSSResponse struct {
	// Body contains the raw RSS XML content.
	Body string
}
