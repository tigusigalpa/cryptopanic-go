package cryptopanic

import (
	"net/http"
	"os"
	"time"
)

// Option configures a Client. Options are applied in the order they
// are passed to New.
type Option func(*Client)

// WithPlan sets the API plan segment used in the base URL. The
// default is PlanGrowth. Valid values are PlanDeveloper,
// PlanGrowth, and PlanEnterprise. Check the plans page at
// https://cryptopanic.com/developers/api/plans for commercial
// details.
func WithPlan(plan Plan) Option {
	return func(c *Client) {
		if plan != "" {
			c.plan = plan
		}
	}
}

// WithBaseURL overrides the default CryptoPanic API base URL. This is
// primarily useful for testing against a mock server. The URL should
// not include the plan segment or trailing slash; the SDK appends
// /{plan}/v2/ automatically.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = url
		}
	}
}

// WithHTTPClient sets a custom *http.Client used to perform requests.
// This allows callers to configure transport-level behaviour such as
// proxies, TLS settings, or custom RoundTrippers.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithTimeout sets the timeout applied to the underlying HTTP client
// for every request made by the Client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// WithRetryPolicy configures automatic retry behaviour for
// idempotent GET requests. Retries are applied for transient network
// failures, HTTP 429, and selected 5xx responses. The Retry-After
// header is honored when present. Exponential backoff with jitter is
// used otherwise. Retries are disabled by default.
func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *Client) {
		if policy.MaxAttempts > 0 {
			c.retry.MaxAttempts = policy.MaxAttempts
		}
		if policy.BaseDelay > 0 {
			c.retry.BaseDelay = policy.BaseDelay
		}
		if policy.MaxDelay > 0 {
			c.retry.MaxDelay = policy.MaxDelay
		}
	}
}

// WithUserAgent sets the User-Agent header value. The SDK name and
// version are included by default; this option overrides the entire
// value.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// WithSleeper sets a custom Sleeper used for retry delays. This is
// primarily useful for testing to avoid real delays.
func WithSleeper(s Sleeper) Option {
	return func(c *Client) {
		if s != nil {
			c.sleeper = s
		}
	}
}

// osGetenv wraps os.Getenv so it can be swapped in tests.
var osGetenv = func(key string) string {
	return os.Getenv(key)
}
