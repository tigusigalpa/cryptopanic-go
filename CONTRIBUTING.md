# Contributing to cryptopanic-go

Thank you for your interest in contributing! Please follow these guidelines to ensure a smooth process.

## Development Setup

1. Clone the repository.
2. Ensure you have Go 1.22 or later installed.
3. Run `go test ./...` to verify the test suite passes.

## Code Style

- Run `gofmt -w .` before committing.
- Run `go vet ./...` and fix any issues.
- Follow idiomatic Go conventions.
- Use `context.Context` as the first parameter in all network-facing methods.
- Never log, expose, or embed the auth token in error messages, URLs, or test output.

## Testing

- Add tests for all new functionality.
- Use `httptest.Server` and fixture-based tests.
- Ensure tests are fast and deterministic — use the `Sleeper` interface to control retry timing.
- Run `go test ./... -v -count=1` before submitting.

## Pull Request Process

1. Fork the repository and create a feature branch.
2. Write tests for your changes.
3. Ensure `gofmt`, `go vet`, and `go test` all pass.
4. Update the CHANGELOG.md if applicable.
5. Submit a pull request with a clear description of the changes.

## Security

- Never commit real API credentials.
- Never hardcode tokens in examples or tests.
- Report security vulnerabilities to sovletig@gmail.com — do not open public issues.
