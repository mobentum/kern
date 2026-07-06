SHELL := /bin/bash

BENCH_DIR := benchmarks/fourway
RESULTS_DIR := $(BENCH_DIR)/results
PROFILE_DIR := $(RESULTS_DIR)/profiles
GRPC_DIR := extensions/grpc
GRPC_EXAMPLE_DIR := $(GRPC_DIR)/examples/kern-integration

BENCHTIME ?= 5s
BENCH_PATTERN ?= BenchmarkFramework
RESULT_STAMP ?= $(shell date +%Y%m%d-%H%M%S)
RESULT_FILE ?= $(RESULTS_DIR)/bench-$(RESULT_STAMP).txt
CPU_PROFILE ?= $(PROFILE_DIR)/bench_cpu.out
MEM_PROFILE ?= $(PROFILE_DIR)/bench_mem.out
PPROF_SYMBOL ?= github.com/mobentum/kern/benchmarks/fourway.benchmarkKernQueryAccess.func1

.PHONY: help prepare bench bench-short bench-full bench-save bench-kern-query bench-kern-path bench-cpu bench-mem pprof-top-cpu pprof-top-mem pprof-list-cpu pprof-list-mem test test-grpc test-grpc-example build-grpc ci-grpc check-buf grpc-buf-lint grpc-buf-build grpc-buf-generate grpc-buf ci-proto ci-all ci-all-local clean-profiles clean-artifacts ci-go ci-benchmark-smoke docs-install docs-lint docs-typecheck docs-build ci-docs go-vet go-lint go-vuln go-quality check-go-tools

help:
	@echo "Benchmark and pprof workflow"
	@echo ""
	@echo "Common:"
	@echo "  make bench-full                  # full four-way benchmark suite (default 5s)"
	@echo "  make bench-save                  # save full benchmark suite output to $(RESULTS_DIR)/"
	@echo "  make bench-kern-query            # focus query benchmark (kern)"
	@echo "  make bench-cpu BENCH_PATTERN='BenchmarkFrameworkQueryAccess/kern$$' CPU_PROFILE=$(PROFILE_DIR)/kern_query.cpu.out"
	@echo "  make bench-mem BENCH_PATTERN='BenchmarkFrameworkQueryAccess/kern$$' MEM_PROFILE=$(PROFILE_DIR)/kern_query.mem.out"
	@echo "  make pprof-top-cpu CPU_PROFILE=$(PROFILE_DIR)/kern_query.cpu.out"
	@echo "  make pprof-list-cpu CPU_PROFILE=$(PROFILE_DIR)/kern_query.cpu.out PPROF_SYMBOL='github.com/mobentum/kern.lookupRawQueryPairRaw'"
	@echo "  make pprof-top-mem MEM_PROFILE=$(PROFILE_DIR)/kern_query.mem.out"
	@echo "  make pprof-list-mem MEM_PROFILE=$(PROFILE_DIR)/kern_query.mem.out"
	@echo "  make ci-go                       # CI build + test for the root Go module"
	@echo "  make go-quality                  # go vet + golangci-lint + govulncheck"
	@echo "  make check-go-tools              # verify golangci-lint and govulncheck are installed"
	@echo "  make ci-grpc                     # CI build + test for extensions/grpc"
	@echo "  make ci-benchmark-smoke          # CI benchmark sanity check for the benchmark module"
	@echo "  make ci-proto                    # CI proto lint + build (Buf) for grpc example"
	@echo "  make ci-all                      # run ci-go + ci-grpc + ci-benchmark-smoke + ci-proto"
	@echo "  make ci-all-local                # run full checks, skip ci-proto when buf is not installed"
	@echo "  make grpc-buf-generate           # regenerate grpc example stubs with Buf"
	@echo "  make ci-docs                     # CI install + lint + type-check + build for docs"
	@echo "  make clean-artifacts             # remove generated benchmark/test artifacts"
	@echo ""
	@echo "Variables:"
	@echo "  BENCHTIME=5s BENCH_PATTERN=BenchmarkFramework RESULT_FILE=... CPU_PROFILE=... MEM_PROFILE=... PPROF_SYMBOL=..."

prepare:
	@mkdir -p $(PROFILE_DIR)

bench: prepare
	go -C $(BENCH_DIR) test -bench='$(BENCH_PATTERN)' -benchmem -run='^$$' -benchtime=$(BENCHTIME)

bench-short:
	$(MAKE) bench BENCHTIME=3s

bench-full:
	$(MAKE) bench BENCH_PATTERN='BenchmarkFramework'

bench-save: prepare
	go -C $(BENCH_DIR) test -bench='BenchmarkFramework' -benchmem -run='^$$' -benchtime=$(BENCHTIME) | tee $(RESULT_FILE)

bench-kern-query:
	$(MAKE) bench BENCH_PATTERN='BenchmarkFrameworkQueryAccess/kern$$'

bench-kern-path:
	$(MAKE) bench BENCH_PATTERN='BenchmarkFrameworkPathParams/kern$$'

bench-cpu: prepare
	go -C $(BENCH_DIR) test -bench='$(BENCH_PATTERN)' -benchmem -run='^$$' -benchtime=$(BENCHTIME) -cpuprofile=$(CPU_PROFILE)

bench-mem: prepare
	go -C $(BENCH_DIR) test -bench='$(BENCH_PATTERN)' -benchmem -run='^$$' -benchtime=$(BENCHTIME) -memprofile=$(MEM_PROFILE)

pprof-top-cpu:
	go tool pprof -top $(CPU_PROFILE)

pprof-top-mem:
	go tool pprof -top $(MEM_PROFILE)

pprof-list-cpu:
	go tool pprof -list='$(PPROF_SYMBOL)' $(CPU_PROFILE)

pprof-list-mem:
	go tool pprof -list='$(PPROF_SYMBOL)' $(MEM_PROFILE)

test:
	go test ./...

build-grpc:
	go -C $(GRPC_DIR) build ./...

test-grpc:
	go -C $(GRPC_DIR) test ./...

test-grpc-example:
	go -C $(GRPC_EXAMPLE_DIR) test ./...

ci-grpc: build-grpc test-grpc test-grpc-example

check-buf:
	@command -v buf >/dev/null 2>&1 || (echo "buf not found in PATH. Install with: brew install bufbuild/buf/buf" && exit 1)

grpc-buf-lint: check-buf
	buf lint $(GRPC_EXAMPLE_DIR)

grpc-buf-build: check-buf
	buf build $(GRPC_EXAMPLE_DIR)

grpc-buf-generate: check-buf
	buf generate $(GRPC_EXAMPLE_DIR)

grpc-buf: grpc-buf-lint grpc-buf-build

ci-proto: grpc-buf

ci-all: ci-go ci-grpc ci-benchmark-smoke ci-proto

ci-all-local: ci-go ci-grpc ci-benchmark-smoke
	@if command -v buf >/dev/null 2>&1; then \
		$(MAKE) ci-proto; \
	else \
		echo "Skipping ci-proto: buf not found in PATH (install: brew install bufbuild/buf/buf)"; \
	fi

ci-go:
	$(MAKE) go-quality
	go build ./...
	go test -v ./...

go-vet:
	go vet ./...

check-go-tools:
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found in PATH. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@command -v govulncheck >/dev/null 2>&1 || (echo "govulncheck not found in PATH. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest" && exit 1)

go-lint: check-go-tools
	golangci-lint run ./...

go-vuln: check-go-tools
	govulncheck ./...

go-quality: go-vet go-lint go-vuln

ci-benchmark-smoke:
	go -C $(BENCH_DIR) test -bench='BenchmarkFrameworkQueryAccess/kern$$' -benchmem -run='^$$' -benchtime=1x

docs-install:
	cd docs && bun install --frozen-lockfile

docs-lint:
	cd docs && bun run lint

docs-typecheck:
	cd docs && bun run types:check

docs-build:
	cd docs && bun run build

ci-docs: docs-install docs-lint docs-typecheck docs-build

clean-profiles:
	rm -f $(PROFILE_DIR)/*.out

clean-artifacts: clean-profiles
	rm -f *.test $(BENCH_DIR)/*.test $(RESULTS_DIR)/bench-*.txt
