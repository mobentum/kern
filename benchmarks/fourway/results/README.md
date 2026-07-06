# Benchmark Results Folder

This folder stores benchmark artifacts generated while iterating on performance work.

## Layout

- `bench-YYYYMMDD-HHMMSS.txt`
  - Timestamped benchmark output snapshots.
  - Useful for before/after comparisons across code changes.
- `profiles/`
  - CPU and memory profiles captured with `-cpuprofile` and `-memprofile`.
  - Recommended names:
    - `kern_query.cpu.out`
    - `kern_query.mem.out`
    - `kern_path.cpu.out`
    - `kern_path.mem.out`

## What to keep

Keep:
- The latest benchmark snapshot used in docs updates.
- Any benchmark snapshots tied to meaningful performance changes.
- Profiles that explain a specific optimization or regression.

Prune:
- Repeated ad hoc runs with no decision value.
- Old profiles after findings have been documented in `benchmarks/fourway/README.md` or `handoff/WORK_CONTEXT.md`.

## Recommended workflow

1. Run focused benchmark targets while iterating.
2. Save only the runs that represent a baseline or an improvement.
3. Save matching CPU/memory profiles under `profiles/` when investigating hotspots.
4. Update `benchmarks/fourway/README.md` after meaningful benchmark changes.

## Example commands

From repo root:

```bash
make bench-save
make bench-full
make bench-cpu BENCH_PATTERN='BenchmarkFrameworkQueryAccess/kern$' CPU_PROFILE=benchmarks/fourway/results/profiles/kern_query.cpu.out
make bench-mem BENCH_PATTERN='BenchmarkFrameworkQueryAccess/kern$' MEM_PROFILE=benchmarks/fourway/results/profiles/kern_query.mem.out
make clean-artifacts
```
