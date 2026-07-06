# Roadmap Status

Last updated: 2026-06-25

This file is the tracking source for roadmap completion in this repository.

## Quick Wins (1-2 days)

- [x] Typed framework errors (`Error` type + `c.Error` helper)
- [x] `BindQuery`, `BindForm`, `BindHeader` helpers
- [x] Test client helper API for concise tests
- [x] BearerAuth middleware
- [x] BasicAuth middleware
- [x] Body size limit guard

## Medium Term (1-2 weeks)

- [x] Unified `Bind` API with validation tags
- [x] Lifecycle hooks (`OnRoute`, `OnListen`, `OnError`, `OnShutdown`)
- [x] Structured logger config (json/text, fields, output)
- [x] Rate limiter middleware
- [x] Clearer first-party middleware set (optional middleware package with JWT, CSRF, timeout, rate limiter, request id, compression, security headers)

## Strategic (1-2 months)

- [x] Full middleware suite
  - [x] JWT auth
  - [x] CSRF
  - [x] Session middleware
  - [x] Timeout middleware
  - [x] Helmet-style security headers middleware
- [x] Routing improvements
  - [x] Route naming and introspection (`RouteNamed`, `Routes`, `RouteByName`)
  - [x] Typed path constraints
- [x] Streaming and advanced request controls
  - [x] File streaming/download, range handling, body limit
  - [x] Additional advanced request controls
    - [x] Header count / total header bytes guard middleware
    - [x] Strict parsing toggles (invalid query/form handling policy)
    - [x] Per-route request guards (method/content-type/body requirements)
    - [x] Route-level body size limits via request guard middleware
- [x] Docs jump
  - [x] Performance workflow docs
  - [x] Recipes docs
  - [x] Middleware catalog expansion
  - [x] Migration guide

## Next recommended work

- [x] Add observability docs and examples for guard/session failure handling.
- [x] Benchmark guarded route overhead vs unguarded routes in benchmarks/fourway.
- [x] Consider optional per-route response-size limits for streaming endpoints.

## Prioritized middleware roadmap

### P1 (next)

- [ ] ETag middleware
- [ ] Cache middleware
- [ ] Idempotency middleware
- [ ] HostAuthorization middleware

### P2

- [ ] Healthcheck middleware
- [ ] APIKeyAuth middleware
- [ ] Rewrite/Redirect middleware

### P3

- [ ] SSE helper middleware

## Active checklist (2026-06-25)

- [x] Document `RequestGuard` deny-path observability in docs recipes.
- [x] Document session cookie/decryption failure observability pattern in docs recipes.
- [x] Review examples and add a dedicated observability example if missing.
- [x] Add `BenchmarkRequestGuard` (guarded vs unguarded) in `benchmarks/fourway`.
- [x] Capture benchmark artifact with ns/op delta and link from docs/benchmark notes.
- [x] Prototype per-route response-size limits and measure allocation impact.

Benchmark note (2026-06-25, benchtime=3s):
- `BenchmarkRequestGuard/unguarded`: 62.78 ns/op, 0 B/op, 0 allocs/op
- `BenchmarkRequestGuard/guarded`: 252.3 ns/op, 112 B/op, 4 allocs/op
- Delta (guarded - unguarded): +189.52 ns/op, +112 B/op, +4 allocs/op
- Artifact: `benchmarks/fourway/results/bench-request-guard-20260625-221916.txt`

Response limit benchmark note (2026-06-25, benchtime=3s):
- `BenchmarkResponseLimit/unguarded`: 71.68 ns/op, 2 B/op, 1 allocs/op
- `BenchmarkResponseLimit/limited`: 95.29 ns/op, 66 B/op, 2 allocs/op
- Delta (limited - unguarded): +23.61 ns/op, +64 B/op, +1 allocs/op
- Artifact: `benchmarks/fourway/results/bench-response-limit-20260625-223121.txt`
