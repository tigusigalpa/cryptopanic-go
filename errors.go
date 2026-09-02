package cryptopanic

import (
	"errors"
	"fmt"
)

// APIError represents an error response returned by the CryptoPanic
// API. It is returned by every method when the API responds with a
// non-2xx HTTP status code. The error carries the status code, a
// safe response excerpt, and the request ID when available, but
// never includes the auth_token or a full sensitive request URL.
type APIError struct {
	// StatusCode is the HTTP status code returned by the API.
	StatusCode int
	// Message is a human-readable error message extracted from the
	// response body or a generated fallback.
	Message string
	// RawBody contains a truncated, redacted copy of the raw response
	// body for debugging purposes (max 512 bytes).
	RawBody []byte
	// RequestID is the value of the X-Request-ID response header, if
	// the server provides one.
	RequestID string
	// Retried is the number of retry attempts made before this error
	// was returned.
	Retried int
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	return fmt.Sprintf("cryptopanic: api error: status=%d message=%s", e.StatusCode, e.Message)
}

// Sentinel errors that can be checked with errors.Is against errors
// returned from methods. The underlying *APIError is always available
// via errors.As for accessing StatusCode, RawBody, etc.
var (
	// ErrMissingAuthToken is returned when no auth_token is provided.
	ErrMissingAuthToken = errors.New("cryptopanic: missing auth token — set CRYPTOPANIC_AUTH_TOKEN or pass a token to New")
	// ErrInvalidQuery is returned when local query validation fails.
	ErrInvalidQuery = errors.New("cryptopanic: invalid query parameters")
	// ErrUnauthorized is returned when the API responds with HTTP 401.
	ErrUnauthorized = errors.New("cryptopanic: unauthorized — check your auth token")
	// ErrForbidden is returned when the API responds with HTTP 403.
	ErrForbidden = errors.New("cryptopanic: forbidden — access or plan restriction")
	// ErrRateLimited is returned when the API responds with HTTP 429
	// and all retry attempts have been exhausted.
	ErrRateLimited = errors.New("cryptopanic: rate limit exceeded")
	// ErrServerError is returned when the API responds with HTTP 500.
	ErrServerError = errors.New("cryptopanic: server error")
)

// wrappedAPIError wraps an *APIError together with a sentinel error
// so that both errors.As(&APIError{}) and errors.Is(sentinel) work on
// the value returned to the caller.
type wrappedAPIError struct {
	*APIError
	sentinel error
}

// Unwrap allows errors.Is/errors.As to traverse to both the *APIError
// and the sentinel error. Go 1.20+ supports returning a slice for
// multi-error unwrapping.
func (w *wrappedAPIError) Unwrap() []error {
	return []error{w.APIError, w.sentinel}
}

// newAPIError builds an error value for the given status code, message
// and raw body, attaching the appropriate sentinel error when the
// status code is a well-known one (401, 403, 429, 500).
func newAPIError(statusCode int, message string, rawBody []byte, requestID string, retried int, authToken string) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    message,
		RawBody:    truncateBody(redactToken(rawBody, authToken)),
		RequestID:  requestID,
		Retried:    retried,
	}

	switch statusCode {
	case 401:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrUnauthorized}
	case 403:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrForbidden}
	case 429:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrRateLimited}
	case 500:
		return &wrappedAPIError{APIError: apiErr, sentinel: ErrServerError}
	default:
		return apiErr
	}
}

// truncateBody limits the raw body to 512 bytes for safe storage in
// error values.
func truncateBody(body []byte) []byte {
	if len(body) > 512 {
		return body[:512]
	}
	return body
}
