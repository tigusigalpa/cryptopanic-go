# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in `cryptopanic-go`, please report it responsibly:

- Email: sovletig@gmail.com
- Do not open a public GitHub issue for security vulnerabilities.

## API Key Safety

- **Never** hardcode your CryptoPanic `auth_token` in source code.
- **Always** load tokens from environment variables or a secure secret manager.
- The SDK never logs, exposes in error messages, or embeds the token in user-facing URLs.
- Pagination URLs returned by the SDK have the `auth_token` parameter redacted to `[REDACTED]`.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x     | Yes       |

## Disclosure Timeline

- We acknowledge receipt within 48 hours.
- We aim to release a fix within 7 days for critical vulnerabilities.
- We coordinate disclosure with the reporter.
