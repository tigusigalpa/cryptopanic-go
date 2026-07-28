# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-15

### Added
- Initial release of `cryptopanic-go`.
- Core `Client` with functional options: `WithPlan`, `WithBaseURL`, `WithHTTPClient`, `WithTimeout`, `WithRetryPolicy`, `WithUserAgent`, `WithSleeper`.
- `New` and `NewFromEnv` constructors.
- Configurable plan segment (`developer`, `growth`, `enterprise`) in the base URL.
- `Posts` method (GET /posts/) with typed `PostsQuery` supporting all documented query parameters.
- `Portfolio` method (GET /portfolio/) returning opaque `PortfolioResponse` with raw JSON.
- `PostsRSS` method (GET /posts/?format=rss) returning raw RSS XML.
- `NewsRSS` method (GET /news/rss/) returning raw RSS XML.
- Typed response models: `Post`, `PostSource`, `PostInstrument`, `PostVotes`, `PostContent`, `PostAuthor`, `PostsPage`.
- Exported error types: `APIError`, `ErrMissingAuthToken`, `ErrInvalidQuery`, `ErrUnauthorized`, `ErrForbidden`, `ErrRateLimited`, `ErrServerError`.
- `errors.Is` and `errors.As` support for status-class sentinel errors.
- Automatic retry on HTTP 429, 500, 502, 503 with `Retry-After` header support and exponential backoff with jitter.
- Token redaction in pagination URLs and error messages.
- `Sleeper` interface for testable retry timing.
- Helper functions: `BoolPtr`, `IntPtr`, `TimePtr`, `StringPtr`.
- Comprehensive test suite (49 tests) covering success parsing, null/missing fields, query encoding, plan selection, token injection, invalid query combinations, pagination redaction, malformed JSON, context cancellation, timeouts, error mapping, retry behavior, and token leak prevention.
- Test fixtures under `testdata/`.
- GitHub Actions CI workflow.
- MIT LICENSE, CHANGELOG.md, CONTRIBUTING.md, SECURITY.md.
