# cryptopanic-go

[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/cryptopanic-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/cryptopanic-go)
[![CI](https://github.com/tigusigalpa/cryptopanic-go/actions/workflows/ci.yml/badge.svg)](https://github.com/tigusigalpa/cryptopanic-go/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A small, typed, context-aware Go client for the [CryptoPanic API](https://cryptopanic.com/developers/api/).

Use it when your application needs to read crypto news, filter posts by currencies or sentiment, consume RSS, or access the authenticated portfolio endpoint without hand-building URLs and decoding JSON in every service.

The SDK deliberately stays close to the public CryptoPanic API. It does not invent a portfolio schema, hide server-side plan rules, or force a framework on your application.

## Contents

- [Why this SDK](#why-this-sdk)
- [Requirements and installation](#requirements-and-installation)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Working with posts](#working-with-posts)
- [Other endpoints](#other-endpoints)
- [Pagination](#pagination)
- [Errors](#errors)
- [Retries and rate limits](#retries-and-rate-limits)
- [Contexts and timeouts](#contexts-and-timeouts)
- [Testing and development](#testing-and-development)

## Why this SDK?

Calling a news API should leave you time to work on your product, not make you maintain another layer of URL construction and response plumbing. `cryptopanic-go` handles the predictable parts and keeps the rest visible:

- **Typed JSON models** — work with `Post`, `PostsPage`, and related structs instead of decoding maps yourself.
- **A context-first API** — every network method accepts `context.Context`, so deadlines and cancellation fit naturally into HTTP handlers, workers, and CLI tools.
- **Safe credential handling** — the token is sent only where the API expects it and is redacted from pagination URLs and SDK errors.
- **Useful errors** — inspect common HTTP failures with `errors.Is`, and get structured details with `errors.As`.
- **Careful retries** — opt in to retries for rate limits, selected server errors, and transient network failures.
- **No runtime dependencies** — the package uses only Go's standard library.
- **Safe concurrent use** — a configured `Client` can be shared between goroutines.

## Requirements and installation

- Go 1.22 or newer.
- A CryptoPanic auth token for authenticated API calls.

```bash
go get github.com/tigusigalpa/cryptopanic-go
```

## Quick start

Set the token in the environment before starting your application:

```bash
export CRYPTOPANIC_AUTH_TOKEN="your-token"
```

Then create one client and reuse it:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    cryptopanic "github.com/tigusigalpa/cryptopanic-go"
)

func main() {
    if os.Getenv("CRYPTOPANIC_AUTH_TOKEN") == "" {
        log.Fatal("CRYPTOPANIC_AUTH_TOKEN is not set")
    }

    client, err := cryptopanic.NewFromEnv(
        cryptopanic.WithPlan(cryptopanic.PlanGrowth),
        cryptopanic.WithTimeout(10*time.Second),
        cryptopanic.WithRetryPolicy(cryptopanic.RetryPolicy{MaxAttempts: 3}),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    page, err := client.Posts(ctx, cryptopanic.PostsQuery{
        Currencies: []string{"BTC", "ETH"},
        Filter:     cryptopanic.FilterRising,
        Page:       cryptopanic.IntPtr(1),
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, post := range page.Results {
        fmt.Printf("[%s] %s\n", post.Kind, post.Title)
    }
}
```

If you already have the token in application configuration, pass it directly instead:

```go
client := cryptopanic.New(
    token,
    cryptopanic.WithPlan(cryptopanic.PlanGrowth),
)
```

`NewFromEnv` returns `ErrMissingAuthToken` when `CRYPTOPANIC_AUTH_TOKEN` is empty. `New` accepts the token directly and returns a client immediately.

## Configuration

### Environment variables

`NewFromEnv` reads the token from `CRYPTOPANIC_AUTH_TOKEN`. The package does not read the plan from the environment automatically; choose it explicitly with `WithPlan` so the URL used by the application is visible in code.

| Variable | Description | Default |
|----------|-------------|---------|
| `CRYPTOPANIC_AUTH_TOKEN` | CryptoPanic auth token; required by `NewFromEnv` | — |

Never commit a token, put it in a URL you log, or print it while debugging. The SDK does not log the token, include it in error messages, or expose it in pagination URLs.

### Client options

| Option | What it controls |
|--------|------------------|
| `WithPlan(plan)` | The plan segment used in the API path. |
| `WithBaseURL(url)` | The API base URL, primarily useful with a mock server. The SDK appends `/{plan}/v2/`. |
| `WithHTTPClient(client)` | A custom `*http.Client` for proxies, transports, TLS, or tests. |
| `WithTimeout(duration)` | The timeout on the underlying `http.Client`. |
| `WithRetryPolicy(policy)` | Retry limits and backoff settings. Retries are off by default. |
| `WithUserAgent(value)` | Replaces the default User-Agent header. |
| `WithSleeper(sleeper)` | Replaces retry sleeping, useful for deterministic tests. |

The defaults are a 30-second HTTP timeout, the `growth` plan, the `cryptopanic-go/1.0.0` User-Agent, and zero retry attempts.

## Working with posts

`Posts` accepts a `PostsQuery` value. Empty fields are omitted from the request, and the client validates only rules that are safe to check locally. The CryptoPanic server remains the authority for account permissions and values that may change over time.

```go
size := 20
pageNumber := 1

page, err := client.Posts(ctx, cryptopanic.PostsQuery{
    Currencies: []string{"BTC", "ETH", "SOL"},
    Regions:    "en",
    Kind:       cryptopanic.KindNews,
    Filter:     cryptopanic.FilterHot,
    Size:       &size,
    Page:       &pageNumber,
})
if err != nil {
    log.Fatal(err)
}

for _, post := range page.Results {
    fmt.Println(post.Title)
    if post.Source != nil {
        fmt.Println("source:", post.Source.Title)
    }
}
```

### Query fields

| Field | Type | What it does | Plan note |
|-------|------|--------------|-----------|
| `Public` | `*bool` | Requests public mode when set. | All listed plans |
| `Currencies` | `[]string` | Sends a comma-separated list such as `BTC,ETH`. | All listed plans |
| `Regions` | `string` | Sets a region code; the API default is `en`. | All listed plans |
| `Filter` | `Filter` | Known values include `rising`, `hot`, `bullish`, `bearish`, `important`, `saved`, and `lol`. Custom values are allowed. | All listed plans |
| `Kind` | `Kind` | `news`, `media`, or `all`. | All listed plans |
| `Following` | `*bool` | Private-only following mode. | All listed plans |
| `LastPull` | `*time.Time` | Sends an RFC 3339 / ISO-8601 timestamp. | Enterprise-gated |
| `PanicPeriod` | `PanicPeriod` | `1h`, `6h`, or `24h`. | Enterprise-gated |
| `PanicSort` | `PanicSort` | `asc` or `desc`; requires `PanicPeriod` locally. | Enterprise-gated |
| `Size` | `*int` | Page size from 1 through 50. | Enterprise-gated |
| `Page` | `*int` | Page number from 1 through 50; the API default is 1. | All listed plans |
| `WithContent` | `*bool` | Requests full post content. | Enterprise-gated |
| `Search` | `string` | Searches for a non-empty string. | Enterprise-gated |

For pointer fields such as `Page`, `Size`, `Public`, and `WithContent`, use the package helpers where convenient:

```go
page, err := client.Posts(ctx, cryptopanic.PostsQuery{
    Page:        cryptopanic.IntPtr(2),
    Size:        cryptopanic.IntPtr(25),
    WithContent: cryptopanic.BoolPtr(true),
})
```

Typed constants are available for filters, kinds, panic periods, panic sorting, and plans:

```go
cryptopanic.FilterRising
cryptopanic.KindNews
cryptopanic.PanicPeriod1h
cryptopanic.PanicSortDesc
cryptopanic.PlanEnterprise
```

The complete set is available in the package documentation on [pkg.go.dev](https://pkg.go.dev/github.com/tigusigalpa/cryptopanic-go).

## Other endpoints

### Portfolio

`Portfolio` returns `PortfolioResponse.Raw` as `json.RawMessage`. This is intentional: the public API reference does not define a complete stable portfolio schema, so the SDK preserves the server response instead of guessing at a model.

```go
portfolio, err := client.Portfolio(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(portfolio.Raw))
```

Decode the raw JSON into the shape your application needs:

```go
import "encoding/json"

var portfolioData map[string]any
if err := json.Unmarshal(portfolio.Raw, &portfolioData); err != nil {
    log.Fatal(err)
}
```

### RSS

RSS methods return raw XML in `RSSResponse.Body`. They do not convert XML into `PostsPage` or JSON models.

```go
feed, err := client.NewsRSS(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Println(feed.Body)
```

`PostsRSS` uses the same `PostsQuery` filters and adds `format=rss`. The public reference documents the RSS response as 20 items regardless of plan.

## Supported endpoints

The CryptoPanic public API documentation uses one shared API Reference page, so the links below intentionally point to the same page.

| Endpoint | Go method | Return value |
|----------|-----------|--------------|
| `GET /posts/` | `client.Posts(ctx, query)` | `*PostsPage` |
| `GET /portfolio/` | `client.Portfolio(ctx)` | `*PortfolioResponse` |
| `GET /posts/?format=rss` | `client.PostsRSS(ctx, query)` | `*RSSResponse` |
| `GET /news/rss/` | `client.NewsRSS(ctx)` | `*RSSResponse` |

See the [CryptoPanic API reference](https://cryptopanic.com/developers/api/) and [plans page](https://cryptopanic.com/developers/api/plans) for server-side behavior.

## Response models

### `PostsPage`

| Field | Meaning |
|-------|---------|
| `Results` | Posts returned for the current page. |
| `Next` / `Previous` | Redacted pagination URLs; empty when unavailable. |
| `HasNext` / `HasPrevious` | Whether another page exists. |
| `NextPage` / `PreviousPage` | Page numbers extracted from the pagination URLs, when present. |

### `Post`

A post includes its ID, slug, title, description, publication and creation timestamps, kind, source, original URL, CryptoPanic URL, image, tagged instruments, votes, author, and optional content. Enterprise-gated or potentially omitted values are pointers, so check them before use:

```go
for _, post := range page.Results {
    fmt.Println(post.Title)

    if post.PublishedAt != nil {
        fmt.Println("published:", post.PublishedAt.Format(time.RFC3339))
    }

    if post.Content != nil {
        // Content is available only when the account and query allow it.
    }
}
```

Unknown additive fields from the server do not make JSON decoding fail.

## Pagination

CryptoPanic returns `next` and `previous` URLs in the response envelope. The SDK exposes those URLs only after redacting the `auth_token` value, and extracts the page number when it is present.

The client does not accept a pagination URL directly. Request the next page using `NextPage` or build the next `PostsQuery` yourself:

```go
page, err := client.Posts(ctx, cryptopanic.PostsQuery{Page: cryptopanic.IntPtr(1)})
if err != nil {
    log.Fatal(err)
}

for page.HasNext && page.NextPage != nil {
    next, err := client.Posts(ctx, cryptopanic.PostsQuery{
        Page: page.NextPage,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, post := range next.Results {
        fmt.Println(post.Title)
    }
    page = next
}
```

## Errors

There are two useful ways to handle errors:

- Use `errors.Is` when you need a stable category such as unauthorized, forbidden, rate-limited, or invalid query.
- Use `errors.As` when you need details from `APIError`, such as the HTTP status, request ID, or retry count.

```go
page, err := client.Posts(ctx, query)
if err != nil {
    switch {
    case errors.Is(err, cryptopanic.ErrMissingAuthToken):
        // Configuration is incomplete.
    case errors.Is(err, cryptopanic.ErrInvalidQuery):
        // Fix the query before sending it again.
    case errors.Is(err, cryptopanic.ErrUnauthorized):
        // 401: check the token.
    case errors.Is(err, cryptopanic.ErrForbidden):
        // 403: check access and plan permissions.
    case errors.Is(err, cryptopanic.ErrRateLimited):
        // 429: slow down or wait for the server's limit window.
    case errors.Is(err, cryptopanic.ErrServerError):
        // 500: a server-side failure.
    }

    var apiErr *cryptopanic.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("status=%d message=%s request_id=%s retried=%d\n",
            apiErr.StatusCode,
            apiErr.Message,
            apiErr.RequestID,
            apiErr.Retried,
        )
    }
}
```

### Sentinel errors

| Sentinel | Meaning |
|----------|---------|
| `ErrMissingAuthToken` | `NewFromEnv` could not find a token. |
| `ErrInvalidQuery` | Local query validation failed. |
| `ErrUnauthorized` | The API returned HTTP 401. |
| `ErrForbidden` | The API returned HTTP 403. |
| `ErrRateLimited` | The API returned HTTP 429 after retries were exhausted. |
| `ErrServerError` | The API returned HTTP 500. |

`APIError` preserves `StatusCode`, a human-readable `Message`, a redacted `RawBody` truncated to 512 bytes, `RequestID` when supplied by the server, and the number of retry attempts in `Retried`. The token is not included in the error string or raw body.

Transport errors and context cancellation are returned as wrapped standard-library errors, so `errors.Is(err, context.DeadlineExceeded)` and similar checks remain useful.

## Retries and rate limits

Retries are **off by default**. Enable them for idempotent GET requests with `WithRetryPolicy`:

```go
client := cryptopanic.New(token,
    cryptopanic.WithRetryPolicy(cryptopanic.RetryPolicy{
        MaxAttempts: 3,              // Retries after the initial request.
        BaseDelay:   500 * time.Millisecond,
        MaxDelay:    30 * time.Second,
    }),
)
```

The policy counts retry attempts, not total requests. `MaxAttempts: 3` means up to four HTTP requests: one initial request plus three retries.

- **Retried HTTP statuses:** 429, 500, 502, and 503.
- **Retried network failures:** timeouts, temporary network errors, connection interruptions, and EOF-style read failures.
- **Not retried:** 400, 401, 403, 404, malformed URLs, unsupported schemes, TLS certificate failures, and other permanent errors.
- **Backoff:** exponential with jitter, capped by `MaxDelay`.
- **Server guidance:** `Retry-After` is honored when present and capped by `MaxDelay`.
- **Cancellation:** context cancellation stops waiting and retrying immediately.

Retries cannot fix an invalid token, an unavailable plan feature, or an invalid query. Handle those cases explicitly.

## Contexts and timeouts

Every network-facing method accepts `context.Context`. In a web server or background worker, give each operation a deadline:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

page, err := client.Posts(ctx, cryptopanic.PostsQuery{})
if errors.Is(err, context.DeadlineExceeded) {
    // The request exceeded the caller's deadline.
}
```

`WithTimeout` sets the default timeout on the underlying `http.Client`. A request context is still the best way to set a shorter, operation-specific deadline or cancel work when the caller disconnects.

## Custom HTTP clients and test servers

Use `WithHTTPClient` when your application needs a proxy, custom transport, TLS configuration, or test instrumentation:

```go
httpClient := &http.Client{Timeout: 20 * time.Second}
client := cryptopanic.New(token, cryptopanic.WithHTTPClient(httpClient))
```

Use `WithBaseURL` with `httptest.Server` or another mock server. The value should not include the plan segment or trailing slash; the SDK appends `/{plan}/v2/` automatically.

`WithSleeper` is available when tests need to observe retry delays without actually sleeping.

## Plan caveats

The free Developer API plan was discontinued. The SDK still exposes the documented `PlanDeveloper` value for compatibility, but it does not claim that an account has access to it.

Parameters such as `last_pull`, `panic_period`, `size`, `with_content`, and `search` may be plan-gated. The SDK validates obvious syntax locally, while CryptoPanic remains the authority for account entitlements and evolving server rules. Check your [account dashboard](https://cryptopanic.com/account/) and the [plans page](https://cryptopanic.com/developers/api/plans) for current commercial details.

## Testing and development

Run the quality checks from the package directory:

```bash
go test ./... -v -count=1
gofmt -l .
go vet ./...
```

The test suite uses `httptest.Server` and fixtures under `testdata/`; it does not make live API calls. The `Sleeper` abstraction keeps retry tests fast and deterministic.

For local development:

```bash
git clone https://github.com/tigusigalpa/cryptopanic-go.git
cd cryptopanic-go
go test ./...
```

## Release, security, and contribution

This project follows [Semantic Versioning](https://semver.org/). See [CHANGELOG.md](CHANGELOG.md) for release history, [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines, and [SECURITY.md](SECURITY.md) for vulnerability reporting and credential-safety guidance.

## License

MIT — See [LICENSE](LICENSE) for details.

## Author

**Igor Sazonov**

- GitHub: [https://github.com/tigusigalpa](https://github.com/tigusigalpa)
- Email: sovletig@gmail.com
