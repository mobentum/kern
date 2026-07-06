# Multi-framework benchmark

This benchmark suite compares equivalent fixed-path workloads across:

- kern
- mach
- chi
- gin
- fiber
- raw fasthttp

Benchmark files are organized by scenario:

- [benchmark_test.go](benchmark_test.go) (suite entrypoints)
- [helpers_benchmark_test.go](helpers_benchmark_test.go)
- [plaintext_benchmark_test.go](plaintext_benchmark_test.go)
- [query_benchmark_test.go](query_benchmark_test.go)
- [path_params_benchmark_test.go](path_params_benchmark_test.go)
- [decode_json_benchmark_test.go](decode_json_benchmark_test.go)

Utility scripts/artifacts:

- [scripts/bench_periodic.sh](scripts/bench_periodic.sh)
- [results/](results/)

## How to run

From this directory:

```bash
go test -bench=BenchmarkFramework -benchmem -run='^$' -benchtime=3s
```

From the repo root:

```bash
go test ./benchmarks/fourway -bench=BenchmarkFramework -benchmem -run='^$' -benchtime=3s
```

Notes:

- `-run='^$'` skips regular tests and runs only benchmarks.
- `-benchmem` includes allocation metrics.
- Increase `-benchtime` (for example, `5s` or `10s`) for more stable numbers.

## Makefile shortcuts

From repo root, use the workflow targets in [Makefile](../../Makefile):

```bash
make bench-full
make bench-save
make bench-kern-query
make bench-cpu BENCH_PATTERN='BenchmarkFrameworkQueryAccess/kern$' CPU_PROFILE=benchmarks/fourway/results/profiles/kern_query.cpu.out
make bench-mem BENCH_PATTERN='BenchmarkFrameworkQueryAccess/kern$' MEM_PROFILE=benchmarks/fourway/results/profiles/kern_query.mem.out
make pprof-top-cpu CPU_PROFILE=benchmarks/fourway/results/profiles/kern_query.cpu.out
make pprof-top-mem MEM_PROFILE=benchmarks/fourway/results/profiles/kern_query.mem.out
make pprof-list-cpu CPU_PROFILE=benchmarks/fourway/results/profiles/kern_query.cpu.out PPROF_SYMBOL='github.com/mobentum/kern.lookupRawQueryPairRaw'
make clean-artifacts
make test
```

Run `make help` for the full list.

## Latest results

Date: 2026-06-24  
Machine: Apple M3 Pro (darwin/arm64)  
Go: 1.25.0

### Plaintext

| Framework | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| kern | 86.97 | 0 | 0 |
| mach | 132.7 | 16 | 1 |
| chi | 129.0 | 370 | 3 |
| gin | 71.94 | 48 | 1 |
| fiber | 51.18 | 0 | 0 |
| fasthttp | 16.91 | 0 | 0 |

### Plaintext + middleware

| Framework | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| kern | 100.3 | 0 | 0 |
| mach | 182.5 | 64 | 3 |
| chi | 153.7 | 370 | 3 |
| gin | 116.2 | 64 | 2 |
| fiber | 102.0 | 0 | 0 |
| fasthttp | 59.06 | 0 | 0 |

### Query access

| Framework | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| kern | 108.0 | 0 | 0 |
| mach | 389.7 | 480 | 7 |
| chi | 343.3 | 808 | 7 |
| gin | 296.8 | 485 | 6 |
| fiber | 98.36 | 8 | 1 |
| fasthttp | 33.85 | 0 | 0 |

### Decode JSON

| Framework | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| kern | 546.3 | 272 | 7 |
| mach | 615.1 | 968 | 9 |
| chi | 679.9 | 1336 | 11 |
| gin | 664.4 | 969 | 9 |
| fiber | 506.2 | 280 | 7 |
| fasthttp | 454.7 | 272 | 7 |

### Path params

| Framework | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| kern | 213.0 | 48 | 2 |
| mach | 301.2 | 96 | 5 |
| chi | 299.9 | 712 | 5 |
| gin | 112.7 | 56 | 2 |
| fiber | 122.4 | 8 | 1 |
| fasthttp | 42.63 | 0 | 0 |

### RequestGuard overhead (kern only)

Date: 2026-06-25  
Machine: Apple M3 Pro (darwin/arm64)  
Go: 1.25.0  
Benchtime: 3s

| Scenario | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| unguarded | 62.78 | 0 | 0 |
| guarded (`RequestGuard`) | 252.3 | 112 | 4 |

Delta (`guarded - unguarded`):

- +189.52 ns/op
- +112 B/op
- +4 allocs/op

Artifact: [results/bench-request-guard-20260625-221916.txt](results/bench-request-guard-20260625-221916.txt)

### ResponseLimit overhead (kern only)

Date: 2026-06-25  
Machine: Apple M3 Pro (darwin/arm64)  
Go: 1.25.0  
Benchtime: 3s

| Scenario | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| unguarded | 71.68 | 2 | 1 |
| limited (`ResponseLimit`) | 95.29 | 66 | 2 |

Delta (`limited - unguarded`):

- +23.61 ns/op
- +64 B/op
- +1 allocs/op

Artifact: [results/bench-response-limit-20260625-223121.txt](results/bench-response-limit-20260625-223121.txt)

## pprof-driven optimization process

Use this workflow whenever you see a slowdown or an unexpected allocation count.

### 1) Lock down the target benchmark

Run only the benchmark you want to improve first:

```bash
go test -bench='BenchmarkFrameworkQueryAccess/kern$' -benchmem -run='^$'
```

### 2) Capture allocation profile

```bash
go test -bench='BenchmarkFrameworkQueryAccess/kern$' -benchmem -run='^$' -memprofile=/tmp/kern_query.mem.out
```

### 3) Inspect top allocators

```bash
go tool pprof -top /tmp/kern_query.mem.out
```

This tells you which function owns the allocation space.

### 4) Drill into exact lines

```bash
go tool pprof -list=github.com/mobentum/kern/benchmarks/fourway.benchmarkKernQueryAccess.func1 /tmp/kern_query.mem.out
```

This pinpoints the hot line so you can separate framework costs from benchmark-handler costs.

### 5) Verify whether allocation is framework or benchmark artifact

In this repo, one middleware allocation was coming from benchmark code using `w.Header().Set(...)`, not kern internals.

We confirm by profiling the middleware benchmark alone:

```bash
go test -bench='BenchmarkFrameworkPlaintextMiddleware/kern$' -benchmem -run='^$' -memprofile=/tmp/kern_mw.mem.out
go tool pprof -top /tmp/kern_mw.mem.out
go tool pprof -list=github.com/mobentum/kern/benchmarks/fourway.benchmarkKernPlaintextMiddleware.func1 /tmp/kern_mw.mem.out
```

For the path-params benchmark, `pprof` currently attributes the remaining allocations mainly to stdlib route matching (`net/http.(*routingNode).matchPath`), which is outside kern's handler code path:

```bash
go test -bench='BenchmarkFrameworkPathParams/kern$' -benchmem -run='^$' -memprofile=/tmp/kern_path.mem.out
go tool pprof -top /tmp/kern_path.mem.out
```

### 6) Apply targeted change, not broad rewrites

Examples from current pass:

- Added `Context.TextPair(...)` to avoid variadic formatting allocations in a hot query response path.
- Added `Context.QueryPairDefault(...)` to consolidate multi-field query reads in one helper path.
- Added `Context.QueryPairDefaultRaw(...)` as an opt-in no-decode fast path for simple ASCII query workloads.
- Updated benchmark middleware header write to avoid benchmark-only `Header.Set` allocation noise.

### 7) Re-run full multi-framework suite

```bash
go test -bench=. -benchmem -run='^$'
```

### 8) Record deltas and keep history

- Save intentional outputs under `benchmarks/fourway/results/` with `make bench-save`.
- Prefer comparing at least 3 runs before deciding a regression is real.
- Use longer benchtime (`-benchtime=5s` or `10s`) for higher confidence.

## CPU and memory improvement loop (practical)

1. Pick one scenario (for example query access) and benchmark it in isolation.
2. Capture CPU profile and allocation profile for the exact same benchmark.
3. Read `pprof -top` first, then `pprof -list` for the top symbol.
4. Determine whether cost is in framework code, benchmark handler code, or stdlib routing.
5. Apply a targeted change only for that hotspot.
6. Re-run isolated benchmark, then full suite, then `go test ./...`.
7. Keep a before/after note in this file and store profiles under `benchmarks/fourway/results/profiles/`.

Patterns validated in this repo:

- Prefer one-shot query helpers in hot handlers (`QueryPairDefaultRaw`, `TextPair`) to reduce branch and formatting overhead.
- Use route-level guards only where needed to avoid global hot-path tax.
- Validate benchmark artifacts (for example middleware header writes) before attributing overhead to framework internals.

## Practical guardrails

- Optimize one benchmark target at a time.
- Keep functionality equivalent while comparing frameworks.
- If an optimization changes API usage, document that explicitly.
- Always run `go test ./...` after performance edits.

## Interpretation

- fasthttp is the fastest baseline in all scenarios.
- fiber and gin are strong on raw routing/query throughput, with fiber still leading query among full-featured frameworks.
- kern reaches zero allocations on middleware and query benchmark paths in this suite.
- kern outperforms mach and chi on all measured scenarios in this run.
- As workload complexity rises (for example, JSON decoding), framework overhead differences narrow.
