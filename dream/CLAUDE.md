# Concurrency & Style

- Use `wg.Go(func() { ... })`, not `wg.Add(1)` + `go func() { defer wg.Done(); ... }()`.
- Acquire the semaphore before launching the goroutine, not inside the closure.
- Group related declarations into `const` and `var` blocks.
- Use `runtime.GOMAXPROCS(0)`, not `runtime.NumCPU()`.
- Protect shared slice writes with a mutex; writing to `slice[idx]` from concurrent goroutines is a data race even when indices don't overlap.

# Go Conventions

- Pass `context.Context` as the first parameter to long-running computations; check `ctx.Done()` in inner loops.
- Use `NewFoo(ctx, ...)` constructor pattern when the struct requires background work or cancellation.
- Prefer `var` blocks with zero values and `:=` inside function bodies; avoid `new()`.
- Use `range n` (integer range) instead of `for i := 0; i < n; i++`.
- Name receiver variables with short single-letter abbreviations (`h`, `s`), not `self` or `this`.

# Implementation

- Allocate all rows of a 2D slice before writing cross-references; `mat[j][i] = v` panics if row `j` hasn't been allocated yet.
- When multiple HTMX inputs must send their values together, use `hx-include` with the sibling element IDs.
- Use `hx-sync="this:replace"` on interactive controls to abort in-flight requests when a new one fires.
- Precompute expensive results at startup; serve fast endpoints (like spurious correlation) per-request.
- Use fixed RNG seeds per computation unit for deterministic sample generation; accept non-determinism from library internals (permutation shuffles) that use the global source.
