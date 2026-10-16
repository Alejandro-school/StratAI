---
name: python-performance-optimization
description: Profile a slow StratAI Python workload and verify CPU or memory improvements. Use for measured backend or AI Coach performance problems, not routine Python edits.
metadata:
  author: StratAI
---

# Python performance investigation

Choose a representative, bounded workload and record input size, interpreter version, output destination and expected result. Use the project's configured Python interpreter from README.md. Keep profiling outputs in a new temporary directory and preserve retained evidence under AGENTS.md.

## Measure before changing

Use standard-library tools first:

- [cProfile and pstats](https://docs.python.org/3.11/library/profile.html) to locate expensive call paths. Profile the actual entry point, including worker processes separately when needed.
- time.perf_counter or timeit for repeated unprofiled timing of the same workload. Profiling overhead is not a speedup baseline.
- [tracemalloc](https://docs.python.org/3.11/library/tracemalloc.html) snapshots for Python allocation growth. It does not account for all native or GPU memory; use process memory measurements when that distinction matters.

For a script whose output behaviour has already been checked, the command shape is:

```powershell
& '.\backend\venv\Scripts\python.exe' -m cProfile -o '<new-temp-dir>\profile.prof' '<script.py>' <existing-safe-arguments>
& '.\backend\venv\Scripts\python.exe' -m pstats '<new-temp-dir>\profile.prof'
```

Replace placeholders with the discovered workload and verified temporary destination. Inspect cumulative time, self time and call count. Distinguish parsing, joins, validation, serialisation and I/O before selecting a fix.

## Optimise the observed bottleneck

Prefer eliminating repeated work, indexing repeated joins and bounding intermediate allocations. Preserve ordering, precision, missing-data semantics, provenance and deterministic outputs. A cache needs a complete key, bounded lifetime and invalidation; it must not mix users, matches or contract versions. Use concurrency only where measurements justify it and the workload's ordering and side effects permit it. On Windows, account for process spawning and serialisation cost.

For FastAPI, distinguish blocking CPU work from waiting on I/O; adding async does not make CPU work parallel. Introduce profilers, native extensions or numerical libraries only when the observed problem needs them and their maintenance cost is justified.

Compare outputs and repeat unprofiled timing with the same input and environment; report variation, runtime and memory tradeoffs. Use the relevant root checks; AI Coach tests are documented separately in ai_coach/README.md. Changes to producers, derived facts or data contracts also require the one-demo gate in AGENTS.md. A performance request does not change the phase's agreed game criteria.
