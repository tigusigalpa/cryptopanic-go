// Package cryptopanic provides a Go SDK for the CryptoPanic API
// (https://cryptopanic.com/developers/api/).
//
// The SDK exposes a configurable, context-aware Client with typed
// methods for the publicly documented endpoints: Posts, Portfolio,
// and RSS feeds. All network-facing methods accept context.Context
// as their first parameter.
package cryptopanic

import (
	"net/http"
	"time"
)

// DefaultBaseURL is the default CryptoPanic API base URL. The {plan}
// segment is replaced at construction time based on the configured
// Plan option.
const DefaultBaseURL = "https://cryptopanic.com/api"

// DefaultTimeout is the default HTTP client timeout used unless
// overridden with WithTimeout.
const DefaultTimeout = 30 * time.Second

// DefaultUserAgent is the default User-Agent header value sent with
// every request.
const DefaultUserAgent = "cryptopanic-go/1.0.0"

// DefaultMaxRetryAttempts is the default number of retry attempts
// (excluding the initial request) for transient failures. By default
// retries are disabled.
const DefaultMaxRetryAttempts = 0

// DefaultRetryBaseDelay is the default initial backoff delay for
// retry attempts.
const DefaultRetryBaseDelay = 500 * time.Millisecond

// DefaultRetryMaxDelay is the maximum delay between retry attempts.
const DefaultRetryMaxDelay = 30 * time.Second

// Plan represents a CryptoPanic API plan segment used in the base URL.
type Plan string

const (
	// PlanDeveloper is the developer plan segment. Note: the free
	// Developer API plan was discontinued. Check your account dashboard
	// and the plans page for current commercial details.
	PlanDeveloper Plan = "developer"
	// PlanGrowth is the growth plan segment.
	PlanGrowth Plan = "growth"
	// PlanEnterprise is the enterprise plan segment.
	PlanEnterprise Plan = "enterprise"
)

// Client is the entry point of the CryptoPanic Go SDK. It holds HTTP
// configuration and exposes methods for the publicly documented API
// endpoints. A Client is safe for concurrent use by multiple
// goroutines.
type Client struct {
	authToken  string
	plan       Plan
	baseURL    string
	httpClient *http.Client
	userAgent  string
	retry      RetryPolicy
	sleeper    Sleeper
}

// RetryPolicy controls the retry behaviour for idempotent GET
// requests. Retries are disabled by default (MaxAttempts = 0).
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts (excluding
	// the initial request). Set to 0 to disable retries.
	MaxAttempts int
	// BaseDelay is the initial backoff delay.
	BaseDelay time.Duration
	// MaxDelay caps the delay between retries.
	MaxDelay time.Duration
}

// Sleeper is an interface for sleeping between retry attempts. It
// allows tests to control timing without real delays.
type Sleeper interface {
	Sleep(d time.Duration) bool
}

// defaultSleeper uses time.Sleep and always returns true.
type defaultSleeper struct{}

func (defaultSleeper) Sleep(d time.Duration) bool {
	time.Sleep(d)
	return true
}

// New creates a new CryptoPanic API client authenticated with the
// given auth_token. Behaviour can be customized via Option values such
// as WithPlan, WithBaseURL, WithHTTPClient, WithTimeout,
// WithRetryPolicy, and WithUserAgent.
//
// The auth_token is sent as the auth_token query parameter on every
// request, as required by the CryptoPanic API. It is never placed in
// a URL path, user agent, error message, or log record.
func New(authToken string, opts ...Option) *Client {
	c := &Client{
		authToken:  authToken,
		plan:       PlanGrowth,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: DefaultTimeout},
		userAgent:  DefaultUserAgent,
		retry: RetryPolicy{
			MaxAttempts: DefaultMaxRetryAttempts,
			BaseDelay:   DefaultRetryBaseDelay,
			MaxDelay:    DefaultRetryMaxDelay,
		},
		sleeper: defaultSleeper{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NewFromEnv reads the auth_token from the CRYPTOPANIC_AUTH_TOKEN
// environment variable and returns a new CryptoPanic client. Additional
// Option values can be passed to override plan, timeouts, retries, etc.
func NewFromEnv(opts ...Option) (*Client, error) {
	token := osGetenv("CRYPTOPANIC_AUTH_TOKEN")
	if token == "" {
		return nil, ErrMissingAuthToken
	}
	return New(token, opts...), nil
}
