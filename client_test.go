package cryptopanic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testToken = "test-secret-token-12345"

// --- Test helpers ---

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return b
}

func newTestClient(server *httptest.Server, opts ...Option) *Client {
	defaultOpts := []Option{
		WithBaseURL(server.URL),
		WithPlan(PlanGrowth),
		WithSleeper(noDelaySleeper{}),
	}
	defaultOpts = append(defaultOpts, opts...)
	return New(testToken, defaultOpts...)
}

type noDelaySleeper struct{}

func (noDelaySleeper) Sleep(d time.Duration) bool {
	return true
}

// --- Client construction tests ---

func TestNew_Defaults(t *testing.T) {
	c := New(testToken)
	if c.authToken != testToken {
		t.Errorf("expected authToken %q, got %q", testToken, c.authToken)
	}
	if c.plan != PlanGrowth {
		t.Errorf("expected default plan %q, got %q", PlanGrowth, c.plan)
	}
	if c.baseURL != DefaultBaseURL {
		t.Errorf("expected default baseURL %q, got %q", DefaultBaseURL, c.baseURL)
	}
	if c.userAgent != DefaultUserAgent {
		t.Errorf("expected default userAgent %q, got %q", DefaultUserAgent, c.userAgent)
	}
	if c.retry.MaxAttempts != DefaultMaxRetryAttempts {
		t.Errorf("expected default MaxAttempts %d, got %d", DefaultMaxRetryAttempts, c.retry.MaxAttempts)
	}
}

func TestNew_WithOptions(t *testing.T) {
	c := New(testToken,
		WithPlan(PlanEnterprise),
		WithBaseURL("https://custom.example.com"),
		WithTimeout(5*time.Second),
		WithUserAgent("myapp/1.0"),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second}),
	)
	if c.plan != PlanEnterprise {
		t.Errorf("expected plan %q, got %q", PlanEnterprise, c.plan)
	}
	if c.baseURL != "https://custom.example.com" {
		t.Errorf("expected baseURL %q, got %q", "https://custom.example.com", c.baseURL)
	}
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", c.httpClient.Timeout)
	}
	if c.userAgent != "myapp/1.0" {
		t.Errorf("expected userAgent %q, got %q", "myapp/1.0", c.userAgent)
	}
	if c.retry.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", c.retry.MaxAttempts)
	}
}

func TestNewFromEnv_Success(t *testing.T) {
	osGetenv = func(key string) string {
		if key == "CRYPTOPANIC_AUTH_TOKEN" {
			return "env-token"
		}
		return ""
	}
	defer func() { osGetenv = os.Getenv }()

	c, err := NewFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.authToken != "env-token" {
		t.Errorf("expected token from env, got %q", c.authToken)
	}
}

func TestNewFromEnv_MissingToken(t *testing.T) {
	osGetenv = func(key string) string { return "" }
	defer func() { osGetenv = os.Getenv }()

	_, err := NewFromEnv()
	if !errors.Is(err, ErrMissingAuthToken) {
		t.Errorf("expected ErrMissingAuthToken, got %v", err)
	}
}

// --- PostsQuery validation tests ---

func TestPostsQuery_Validate_PanicSortWithoutPeriod(t *testing.T) {
	q := PostsQuery{PanicSort: PanicSortAsc}
	if !errors.Is(q.Validate(), ErrInvalidQuery) {
		t.Error("expected ErrInvalidQuery for panic_sort without panic_period")
	}
}

func TestPostsQuery_Validate_PanicSortWithPeriod(t *testing.T) {
	q := PostsQuery{PanicSort: PanicSortAsc, PanicPeriod: PanicPeriod1h}
	if err := q.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPostsQuery_Validate_SizeOutOfRange(t *testing.T) {
	q := PostsQuery{Size: IntPtr(0)}
	if !errors.Is(q.Validate(), ErrInvalidQuery) {
		t.Error("expected ErrInvalidQuery for size=0")
	}
	q2 := PostsQuery{Size: IntPtr(51)}
	if !errors.Is(q2.Validate(), ErrInvalidQuery) {
		t.Error("expected ErrInvalidQuery for size=51")
	}
}

func TestPostsQuery_Validate_SizeValid(t *testing.T) {
	q := PostsQuery{Size: IntPtr(50)}
	if err := q.Validate(); err != nil {
		t.Errorf("unexpected error for size=50: %v", err)
	}
}

func TestPostsQuery_Validate_PageOutOfRange(t *testing.T) {
	q := PostsQuery{Page: IntPtr(0)}
	if !errors.Is(q.Validate(), ErrInvalidQuery) {
		t.Error("expected ErrInvalidQuery for page=0")
	}
	q2 := PostsQuery{Page: IntPtr(51)}
	if !errors.Is(q2.Validate(), ErrInvalidQuery) {
		t.Error("expected ErrInvalidQuery for page=51")
	}
}

func TestPostsQuery_Validate_PageValid(t *testing.T) {
	q := PostsQuery{Page: IntPtr(1)}
	if err := q.Validate(); err != nil {
		t.Errorf("unexpected error for page=1: %v", err)
	}
}

func TestPostsQuery_Validate_NilQuery(t *testing.T) {
	var q *PostsQuery
	if err := q.Validate(); err != nil {
		t.Errorf("unexpected error for nil query: %v", err)
	}
}

// --- PostsQuery ToValues tests ---

func TestPostsQuery_ToValues(t *testing.T) {
	q := PostsQuery{
		Public:      BoolPtr(true),
		Currencies:  []string{"BTC", "ETH"},
		Regions:     "en",
		Filter:      FilterRising,
		Kind:        KindNews,
		Following:   BoolPtr(false),
		Size:        IntPtr(20),
		Page:        IntPtr(2),
		WithContent: BoolPtr(true),
		Search:      "bitcoin",
	}
	v := q.ToValues()
	if v.Get("public") != "true" {
		t.Errorf("expected public=true, got %q", v.Get("public"))
	}
	if v.Get("currencies") != "BTC,ETH" {
		t.Errorf("expected currencies=BTC,ETH, got %q", v.Get("currencies"))
	}
	if v.Get("regions") != "en" {
		t.Errorf("expected regions=en, got %q", v.Get("regions"))
	}
	if v.Get("filter") != "rising" {
		t.Errorf("expected filter=rising, got %q", v.Get("filter"))
	}
	if v.Get("kind") != "news" {
		t.Errorf("expected kind=news, got %q", v.Get("kind"))
	}
	if v.Get("following") != "false" {
		t.Errorf("expected following=false, got %q", v.Get("following"))
	}
	if v.Get("size") != "20" {
		t.Errorf("expected size=20, got %q", v.Get("size"))
	}
	if v.Get("page") != "2" {
		t.Errorf("expected page=2, got %q", v.Get("page"))
	}
	if v.Get("with_content") != "true" {
		t.Errorf("expected with_content=true, got %q", v.Get("with_content"))
	}
	if v.Get("search") != "bitcoin" {
		t.Errorf("expected search=bitcoin, got %q", v.Get("search"))
	}
}

func TestPostsQuery_ToValues_Empty(t *testing.T) {
	q := PostsQuery{}
	v := q.ToValues()
	if len(v) > 0 {
		t.Errorf("expected empty values, got %v", v)
	}
}

func TestPostsQuery_ToValues_LastPull(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	q := PostsQuery{LastPull: TimePtr(ts)}
	v := q.ToValues()
	if v.Get("last_pull") != ts.Format(time.RFC3339) {
		t.Errorf("expected last_pull=%s, got %q", ts.Format(time.RFC3339), v.Get("last_pull"))
	}
}

// --- Posts success tests ---

func TestPosts_Success(t *testing.T) {
	postsData := mustReadFile(t, "testdata/posts.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("auth_token") != testToken {
			t.Errorf("expected auth_token in query, got %q", r.URL.Query().Get("auth_token"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(postsData)
	}))
	defer server.Close()

	client := newTestClient(server)
	page, err := client.Posts(context.Background(), PostsQuery{
		Currencies: []string{"BTC", "ETH"},
		Filter:     FilterRising,
		Page:       IntPtr(1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(page.Results))
	}

	post := page.Results[0]
	if post.ID != 12345 {
		t.Errorf("expected ID 12345, got %d", post.ID)
	}
	if post.Title != "Bitcoin Surges Past $100K" {
		t.Errorf("expected title, got %q", post.Title)
	}
	if post.Source == nil || post.Source.Title != "CoinDesk" {
		t.Errorf("expected source title CoinDesk, got %v", post.Source)
	}
	if post.Votes == nil || post.Votes.Positive != 120 {
		t.Errorf("expected votes positive 120, got %v", post.Votes)
	}
	if post.PanicScore == nil || *post.PanicScore != 85 {
		t.Errorf("expected panic_score 85, got %v", post.PanicScore)
	}
	if post.Author == nil || post.Author.Name != "crypto_journalist" {
		t.Errorf("expected author name, got %v", post.Author)
	}
	if post.Content == nil || post.Content.Raw != "Full article content here..." {
		t.Errorf("expected content raw, got %v", post.Content)
	}
	if post.Image == nil || *post.Image == "" {
		t.Error("expected non-empty image URL")
	}
	if len(post.Instruments) != 1 || post.Instruments[0].Code != "BTC" {
		t.Errorf("expected 1 instrument BTC, got %v", post.Instruments)
	}
	if post.PublishedAt == nil {
		t.Error("expected non-nil PublishedAt")
	}
}

func TestPosts_PaginationRedaction(t *testing.T) {
	postsData := mustReadFile(t, "testdata/posts.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(postsData)
	}))
	defer server.Close()

	client := newTestClient(server)
	page, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !page.HasNext {
		t.Error("expected HasNext=true")
	}
	if strings.Contains(page.Next, "SECRET_TOKEN") {
		t.Errorf("next URL leaks token: %s", page.Next)
	}
	if strings.Contains(page.Next, testToken) {
		t.Errorf("next URL leaks test token: %s", page.Next)
	}
	if !strings.Contains(page.Next, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in next URL: %s", page.Next)
	}
	if page.HasPrevious {
		t.Error("expected HasPrevious=false")
	}
	if page.NextPage == nil || *page.NextPage != 2 {
		t.Errorf("expected NextPage=2, got %v", page.NextPage)
	}
}

func TestPosts_NullFields(t *testing.T) {
	postsData := mustReadFile(t, "testdata/posts_null_fields.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(postsData)
	}))
	defer server.Close()

	client := newTestClient(server)
	page, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(page.Results))
	}
	post := page.Results[0]
	if post.PublishedAt != nil {
		t.Error("expected nil PublishedAt")
	}
	if post.CreatedAt != nil {
		t.Error("expected nil CreatedAt")
	}
	if post.Source != nil {
		t.Error("expected nil Source")
	}
	if post.Votes != nil {
		t.Error("expected nil Votes")
	}
	if post.PanicScore != nil {
		t.Error("expected nil PanicScore")
	}
	if post.Author != nil {
		t.Error("expected nil Author")
	}
	if post.Content != nil {
		t.Error("expected nil Content")
	}
	if post.Image != nil {
		t.Error("expected nil Image")
	}
	if page.HasPrevious {
		t.Error("expected HasPrevious=false")
	}
	if page.PreviousPage != nil {
		t.Error("expected nil PreviousPage")
	}
}

func TestPosts_InvalidQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for invalid query")
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{PanicSort: PanicSortAsc})
	if !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("expected ErrInvalidQuery, got %v", err)
	}
}

func TestPosts_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": [invalid}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestPosts_QueryEncoding(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{
		Currencies: []string{"BTC", "ETH"},
		Filter:     FilterHot,
		Kind:       KindMedia,
		Page:       IntPtr(3),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedQuery, "currencies=BTC%2CETH") {
		t.Errorf("expected encoded currencies in query: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "filter=hot") {
		t.Errorf("expected filter=hot in query: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "kind=media") {
		t.Errorf("expected kind=media in query: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "page=3") {
		t.Errorf("expected page=3 in query: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "auth_token=") {
		t.Errorf("expected auth_token in query: %s", capturedQuery)
	}
}

func TestPosts_PlanInURL(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithPlan(PlanEnterprise))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedPath, "/enterprise/v2/posts/") {
		t.Errorf("expected enterprise plan in path: %s", capturedPath)
	}
}

// --- Portfolio tests ---

func TestPortfolio_Success(t *testing.T) {
	portfolioData := mustReadFile(t, "testdata/portfolio.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/portfolio/") {
			t.Errorf("expected portfolio in path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(portfolioData)
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.Portfolio(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Raw) == 0 {
		t.Error("expected non-empty raw portfolio data")
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(resp.Raw, &decoded); err != nil {
		t.Fatalf("failed to decode portfolio raw: %v", err)
	}
	if decoded["currency"] != "USD" {
		t.Errorf("expected currency USD, got %v", decoded["currency"])
	}
}

// --- RSS tests ---

func TestPostsRSS_Success(t *testing.T) {
	rssData := mustReadFile(t, "testdata/rss.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "rss" {
			t.Errorf("expected format=rss, got %q", r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write(rssData)
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.PostsRSS(context.Background(), PostsQuery{Currencies: []string{"BTC"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp.Body, "<rss") {
		t.Errorf("expected RSS XML content, got: %s", resp.Body[:100])
	}
	if !strings.Contains(resp.Body, "Bitcoin Surges Past") {
		t.Error("expected RSS item title in content")
	}
}

func TestNewsRSS_Success(t *testing.T) {
	rssData := mustReadFile(t, "testdata/rss.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write(rssData)
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.NewsRSS(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp.Body, "<rss") {
		t.Errorf("expected RSS XML content")
	}
}

// --- Error mapping tests ---

func TestPosts_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid auth token"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected *APIError")
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestPosts_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "Plan access restricted"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestPosts_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "Rate limit exceeded"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestPosts_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if !errors.Is(err, ErrServerError) {
		t.Errorf("expected ErrServerError, got %v", err)
	}
}

func TestPosts_OtherStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Not found"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected *APIError")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

// --- Token redaction tests ---

func TestErrorDoesNotLeakToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid token"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if strings.Contains(errStr, testToken) {
		t.Errorf("error string leaks token: %s", errStr)
	}
}

func TestAPIError_RawBodyDoesNotContainToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "access denied for test-secret-token-12345"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected *APIError")
	}
	bodyStr := string(apiErr.RawBody)
	if strings.Contains(bodyStr, testToken) {
		t.Errorf("raw body leaks token: %s", bodyStr)
	}
	if strings.Contains(apiErr.Message, testToken) {
		t.Errorf("error message leaks token: %s", apiErr.Message)
	}
}

func TestRedactURL(t *testing.T) {
	raw := "https://cryptopanic.com/api/growth/v2/posts/?auth_token=SECRET&page=2"
	redacted := redactURL(raw)
	if strings.Contains(redacted, "SECRET") {
		t.Errorf("redacted URL still contains secret: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in URL: %s", redacted)
	}
}

// --- Context cancellation tests ---

func TestPosts_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Posts(ctx, PostsQuery{})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestPosts_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithTimeout(50*time.Millisecond))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Retry tests ---

func TestPosts_RetryOn429(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithRetryPolicy(RetryPolicy{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond}))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestPosts_RetryExhausted(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithRetryPolicy(RetryPolicy{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond}))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 total attempts (1 initial + 2 retries), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestPosts_RetryOn500(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithRetryPolicy(RetryPolicy{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond}))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
}

func TestPosts_NoRetryOn401(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithRetryPolicy(RetryPolicy{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond}))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt (no retry on 401), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestPosts_RetryAfterHeader(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithRetryPolicy(RetryPolicy{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond}))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
}

func TestPosts_RetryCancelledDuringSleep(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	client := newTestClient(server,
		WithRetryPolicy(RetryPolicy{MaxAttempts: 5, BaseDelay: 10 * time.Second}),
		WithSleeper(cancelSleeper{}),
	)
	cancel()
	_, err := client.Posts(ctx, PostsQuery{})
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

type cancelSleeper struct{}

func (cancelSleeper) Sleep(d time.Duration) bool {
	return false
}

func TestPosts_DefaultRetrySleepRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	client := New(testToken,
		WithBaseURL(server.URL),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1, BaseDelay: time.Second}),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := client.Posts(ctx, PostsQuery{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("retry sleep ignored context cancellation: %s", elapsed)
	}
}

func TestRetryDelayHandlesTinyBaseDelayAndLargeAttempt(t *testing.T) {
	client := New(testToken, WithRetryPolicy(RetryPolicy{
		MaxAttempts: 100,
		BaseDelay:   time.Nanosecond,
		MaxDelay:    time.Second,
	}))

	if delay := client.retryDelay(0); delay != 0 {
		t.Fatalf("expected zero jittered delay for one nanosecond base delay, got %s", delay)
	}
	if delay := client.retryDelay(100); delay < 0 || delay > time.Second {
		t.Fatalf("expected bounded delay for large retry attempt, got %s", delay)
	}
}

func TestWithBaseURL_TrimsTrailingSlash(t *testing.T) {
	client := New(testToken, WithBaseURL("https://example.test/"))
	if got := client.buildURL(client.apiPath("/posts/"), nil); got != "https://example.test/growth/v2/posts/?auth_token="+testToken {
		t.Fatalf("unexpected URL: %s", got)
	}
}

// --- Additive fields test ---

func TestPosts_AdditiveFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"next": null,
			"previous": null,
			"results": [{
				"id": 1,
				"slug": "test",
				"title": "Test",
				"description": "Desc",
				"published_at": "2025-01-15T10:30:00Z",
				"created_at": "2025-01-15T10:31:00Z",
				"kind": "news",
				"source": null,
				"original_url": "",
				"url": "",
				"new_undocumented_field": "should not cause error"
			}],
			"new_envelope_field": "additive"
		}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	page, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("unexpected error for additive fields: %v", err)
	}
	if len(page.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(page.Results))
	}
}

// --- Token injection test ---

func TestPosts_TokenInjectedInQuery(t *testing.T) {
	var tokenSeen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenSeen = r.URL.Query().Get("auth_token")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenSeen != testToken {
		t.Errorf("expected token %q in query, got %q", testToken, tokenSeen)
	}
}

// --- parsePageFromURL tests ---

func TestParsePageFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want *int
	}{
		{"https://example.com/posts/?page=5", intPtr(5)},
		{"https://example.com/posts/?page=1", intPtr(1)},
		{"https://example.com/posts/", nil},
		{"", nil},
		{"null", nil},
		{"https://example.com/posts/?page=abc", nil},
	}
	for _, tt := range tests {
		got := parsePageFromURL(tt.url)
		if (got == nil) != (tt.want == nil) {
			t.Errorf("parsePageFromURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
		if got != nil && *got != *tt.want {
			t.Errorf("parsePageFromURL(%q) = %d, want %d", tt.url, *got, *tt.want)
		}
	}
}

func intPtr(i int) *int {
	return &i
}

// --- User agent test ---

func TestPosts_UserAgent(t *testing.T) {
	var ua string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server, WithUserAgent("my-test-app/2.0"))
	_, err := client.Posts(context.Background(), PostsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ua != "my-test-app/2.0" {
		t.Errorf("expected user agent my-test-app/2.0, got %q", ua)
	}
}

// --- HTTP client customization test ---

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 3 * time.Second}
	c := New(testToken, WithHTTPClient(customClient))
	if c.httpClient != customClient {
		t.Error("expected custom HTTP client")
	}
}

// --- RequestID test ---

func TestPosts_RequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-abc-123")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "forbidden"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected *APIError")
	}
	if apiErr.RequestID != "req-abc-123" {
		t.Errorf("expected request ID req-abc-123, got %q", apiErr.RequestID)
	}
}

// --- Empty body error test ---

func TestPosts_EmptyBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Posts(context.Background(), PostsQuery{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected *APIError")
	}
	if apiErr.Message != "unknown error" {
		t.Errorf("expected 'unknown error', got %q", apiErr.Message)
	}
}

// --- Portfolio error test ---

func TestPortfolio_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.Portfolio(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// --- RSS error test ---

func TestPostsRSS_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.PostsRSS(context.Background(), PostsQuery{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// --- Multi-error unwrap test ---

func TestWrappedAPIError_Unwrap(t *testing.T) {
	wrapped := &wrappedAPIError{
		APIError: &APIError{StatusCode: 429, Message: "rate limited"},
		sentinel: ErrRateLimited,
	}
	unwrapped := wrapped.Unwrap()
	if len(unwrapped) != 2 {
		t.Fatalf("expected 2 unwrapped errors, got %d", len(unwrapped))
	}
}

// --- Ensure no panic on nil query ---

func TestPosts_NilQueryValues(t *testing.T) {
	var q *PostsQuery
	v := q.ToValues()
	if len(v) > 0 {
		t.Errorf("expected empty values for nil query, got %v", v)
	}
}

// --- io.ReadAll simulation for coverage ---

func TestDoGet_NilOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.doGet(context.Background(), client.apiPath("/posts/"), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Format query in RSS ---

func TestPostsRSS_QueryEncoding(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0"?><rss></rss>`))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.PostsRSS(context.Background(), PostsQuery{Currencies: []string{"BTC"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedQuery, "format=rss") {
		t.Errorf("expected format=rss in query: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "currencies=BTC") {
		t.Errorf("expected currencies=BTC in query: %s", capturedQuery)
	}
}

// --- shouldRetryNetwork tests ---

func TestShouldRetryNetwork(t *testing.T) {
	c := New(testToken)

	// Timeouts and other net.Error implementations are retryable.
	if !c.shouldRetryNetwork(&net.DNSError{IsTimeout: true, Err: "timeout"}) {
		t.Error("expected timeout net.Error to be retryable")
	}

	// url.Error wrapping a net.Error (as produced by http.Client.Do)
	// should also be retryable.
	wrapped := &url.Error{Op: "Get", URL: "https://example.com", Err: &net.DNSError{IsTimeout: true, Err: "timeout"}}
	if !c.shouldRetryNetwork(wrapped) {
		t.Error("expected url.Error wrapping a timeout to be retryable")
	}

	// Permanent errors (e.g. unsupported protocol scheme) must not be
	// retried since retrying cannot succeed.
	permanent := &url.Error{Op: "Get", URL: "ftp://example.com", Err: errors.New("unsupported protocol scheme")}
	if c.shouldRetryNetwork(permanent) {
		t.Error("expected permanent error to not be retryable")
	}

	if c.shouldRetryNetwork(nil) {
		t.Error("expected nil error to not be retryable")
	}
}

// --- Ensure fmt is used (prevents unused import in certain build configs) ---

var _ = fmt.Sprintf
var _ = io.ReadAll
