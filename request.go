package cryptopanic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ResponseMetadata holds non-body information from an HTTP response.
type ResponseMetadata struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// RequestID is the value of the X-Request-ID header, if present.
	RequestID string
	// Duration is the time taken for the HTTP round-trip.
	Duration time.Duration
}

// apiPath builds the full API path for a given endpoint.
func (c *Client) apiPath(endpoint string) string {
	return fmt.Sprintf("/%s/v2%s", c.plan, endpoint)
}

// buildURL constructs the full request URL with query parameters.
// The auth_token is always appended as a query parameter.
func (c *Client) buildURL(path string, query url.Values) string {
	fullURL := c.baseURL + path

	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}
	q.Set("auth_token", c.authToken)

	if len(q) > 0 {
		fullURL += "?" + q.Encode()
	}

	return fullURL
}

// buildUserAgent returns the User-Agent header value.
func (c *Client) buildUserAgent() string {
	return c.userAgent
}

// doGet performs an HTTP GET request against the CryptoPanic API and
// decodes the JSON response into out. It handles authentication,
// query string construction, error mapping, response metadata
// capture, and automatic retry with exponential backoff for
// transient failures (HTTP 429 and selected 5xx responses).
func (c *Client) doGet(ctx context.Context, path string, query url.Values, out interface{}) (*ResponseMetadata, error) {
	return c.doGetWithRetry(ctx, path, query, out, 0)
}

// doGetWithRetry is the internal retry loop for GET requests.
func (c *Client) doGetWithRetry(ctx context.Context, path string, query url.Values, out interface{}, retried int) (*ResponseMetadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	fullURL := c.buildURL(path, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cryptopanic: failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.buildUserAgent())

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if c.shouldRetryNetwork(err) && retried < c.retry.MaxAttempts {
			if c.sleepWithCtx(ctx, c.retryDelay(retried)) {
				return c.doGetWithRetry(ctx, path, query, out, retried+1)
			}
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("cryptopanic: request failed: %w", err)
	}
	defer resp.Body.Close()

	meta := &ResponseMetadata{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("X-Request-ID"),
		Duration:   elapsed,
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return meta, fmt.Errorf("cryptopanic: failed to read response body: %w", readErr)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out != nil && len(body) > 0 {
			if err := json.Unmarshal(body, out); err != nil {
				return meta, fmt.Errorf("cryptopanic: failed to decode response: %w", err)
			}
		}
		return meta, nil
	}

	if c.shouldRetryStatus(resp.StatusCode) && retried < c.retry.MaxAttempts {
		delay := c.retryDelayFromHeaders(resp.Header, retried)
		if c.sleepWithCtx(ctx, delay) {
			return c.doGetWithRetry(ctx, path, query, out, retried+1)
		}
		return meta, ctx.Err()
	}

	return meta, newAPIError(resp.StatusCode, extractMessage(body, c.authToken), body, meta.RequestID, retried, c.authToken)
}

// doGetRaw performs an HTTP GET request and returns the raw body as
// a string. Used for RSS endpoints that return XML.
func (c *Client) doGetRaw(ctx context.Context, path string, query url.Values) (string, *ResponseMetadata, error) {
	return c.doGetRawWithRetry(ctx, path, query, 0)
}

// doGetRawWithRetry is the internal retry loop for raw GET requests.
func (c *Client) doGetRawWithRetry(ctx context.Context, path string, query url.Values, retried int) (string, *ResponseMetadata, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	fullURL := c.buildURL(path, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("cryptopanic: failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")
	req.Header.Set("User-Agent", c.buildUserAgent())

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		if c.shouldRetryNetwork(err) && retried < c.retry.MaxAttempts {
			if c.sleepWithCtx(ctx, c.retryDelay(retried)) {
				return c.doGetRawWithRetry(ctx, path, query, retried+1)
			}
			return "", nil, ctx.Err()
		}
		return "", nil, fmt.Errorf("cryptopanic: request failed: %w", err)
	}
	defer resp.Body.Close()

	meta := &ResponseMetadata{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("X-Request-ID"),
		Duration:   elapsed,
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", meta, fmt.Errorf("cryptopanic: failed to read response body: %w", readErr)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return string(body), meta, nil
	}

	if c.shouldRetryStatus(resp.StatusCode) && retried < c.retry.MaxAttempts {
		delay := c.retryDelayFromHeaders(resp.Header, retried)
		if c.sleepWithCtx(ctx, delay) {
			return c.doGetRawWithRetry(ctx, path, query, retried+1)
		}
		return "", meta, ctx.Err()
	}

	return "", meta, newAPIError(resp.StatusCode, extractMessage(body, c.authToken), body, meta.RequestID, retried, c.authToken)
}

// shouldRetryStatus returns true for HTTP status codes that warrant
// a retry: 429 and selected 5xx responses.
func (c *Client) shouldRetryStatus(statusCode int) bool {
	return statusCode == 429 || statusCode == 500 || statusCode == 502 || statusCode == 503
}

// shouldRetryNetwork returns true for transient network errors that
// warrant a retry (e.g. connection timeouts, resets, temporary DNS
// failures). Permanent errors — such as malformed URLs, unsupported
// protocol schemes, or TLS certificate validation failures — are not
// retried since retrying them cannot succeed.
func (c *Client) shouldRetryNetwork(err error) bool {
	if err == nil {
		return false
	}

	// http.Client.Do wraps transport errors in *url.Error. Unwrap to
	// inspect the underlying cause.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		if temporary, ok := netErr.(interface{ Temporary() bool }); ok {
			return temporary.Temporary()
		}
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// retryDelay computes the delay before the next retry attempt using
// exponential backoff with jitter.
func (c *Client) retryDelay(attempt int) time.Duration {
	base := c.retry.BaseDelay
	if base <= 0 {
		base = DefaultRetryBaseDelay
	}
	max := c.retry.MaxDelay
	if max <= 0 {
		max = DefaultRetryMaxDelay
	}
	if base > max {
		base = max
	}
	delay := base
	for i := 0; i < attempt && delay < max; i++ {
		if delay > max/2 {
			delay = max
			break
		}
		delay *= 2
	}
	if delay <= 1 {
		return 0
	}
	jitter := time.Duration(rand.Int63n(int64(delay) / 2))
	return delay/2 + jitter
}

// retryDelayFromHeaders honors the Retry-After header when present,
// otherwise falls back to exponential backoff with jitter.
func (c *Client) retryDelayFromHeaders(headers http.Header, attempt int) time.Duration {
	ra := headers.Get("Retry-After")
	if ra != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(ra)); err == nil && secs >= 0 {
			delay := time.Duration(secs) * time.Second
			max := c.retry.MaxDelay
			if max > 0 && delay > max {
				delay = max
			}
			return delay
		}
	}
	return c.retryDelay(attempt)
}

// sleepWithCtx sleeps for the given duration using the client's
// Sleeper, returning false if the context is cancelled or the
// sleeper indicates the sleep was interrupted.
func (c *Client) sleepWithCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	if err := ctx.Err(); err != nil {
		return false
	}
	if sleeper, ok := c.sleeper.(interface {
		SleepContext(context.Context, time.Duration) bool
	}); ok {
		return sleeper.SleepContext(ctx, d)
	}
	return c.sleeper.Sleep(d) && ctx.Err() == nil
}

// extractMessage attempts to pull a human-readable message from a
// JSON error response body. If the body cannot be parsed, a generic
// message with a truncated snippet is returned.
func extractMessage(body []byte, authToken string) string {
	body = redactToken(body, authToken)
	var parsed struct {
		Message string `json:"message"`
		Msg     string `json:"msg"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if parsed.Message != "" {
			return parsed.Message
		}
		if parsed.Msg != "" {
			return parsed.Msg
		}
		if parsed.Error != "" {
			return parsed.Error
		}
	}
	if len(body) == 0 {
		return "unknown error"
	}
	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	return snippet
}

// redactToken removes the configured authentication token from data supplied
// by the server before it is exposed in an error. Although the API should not
// echo credentials, protecting both error messages and RawBody prevents an
// accidental server echo from leaking a caller's secret into logs.
func redactToken(body []byte, authToken string) []byte {
	if len(body) == 0 || authToken == "" {
		return body
	}

	redacted := string(body)
	for _, value := range []string{authToken, url.QueryEscape(authToken), url.PathEscape(authToken)} {
		if value != "" {
			redacted = strings.ReplaceAll(redacted, value, "[REDACTED]")
		}
	}
	return []byte(redacted)
}

// redactURL removes the auth_token query parameter from a URL string
// to prevent credential leakage in logs and error messages. The
// replacement value [REDACTED] is inserted without URL-encoding so
// it remains human-readable.
func redactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	if q.Has("auth_token") {
		q.Set("auth_token", "[REDACTED]")
		u.RawQuery = q.Encode()
		// url.Values.Encode() percent-encodes brackets; replace
		// the encoded form with the literal for readability.
		u.RawQuery = strings.ReplaceAll(u.RawQuery, "%5BREDACTED%5D", "[REDACTED]")
	}
	return u.String()
}

// parsePageFromURL extracts the "page" query parameter value from a
// URL string, returning nil if the URL is null, empty, or lacks a
// page parameter.
func parsePageFromURL(rawURL string) *int {
	if rawURL == "" || rawURL == "null" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	pageStr := u.Query().Get("page")
	if pageStr == "" {
		return nil
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return nil
	}
	return &page
}
