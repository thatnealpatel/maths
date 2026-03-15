# PLAN.md — Statistical Intuition Visualizations

## Overview

Five interactive visualizations that build visual intuition for the statistical methods used in a robustness framework for evaluating backtested options trading strategies. Each becomes a bespoke template in `cmd/site/templates/` with corresponding handler(s) in `cmd/site/handlers/`.

These follow the architecture established in the main `cmd/site` PLAN.md: Go HTML templates, HTMX for server-side compute, shared stylesheet from `cmd/design/`, minimal client-side JS for canvas rendering only.

## Prerequisites

**`stage2_portable.go`**: Contains the CART tree builder, bagged ensemble, and 5-fold CV loop. The user will copy this file into the module before Claude Code runs. Expect it at:

```
internal/ml/stage2_portable.go
```

If it lands elsewhere, adjust import paths accordingly. Do **not** reimplement the tree builder, bagger, or CV loop — use what's in that file directly. The classifier spec is: depth-5 CART trees, Gini impurity, 100 trees per ensemble, bootstrap aggregated, 5-fold CV, accuracy tested against Binomial(n_test, 0.5).

**HTMX**: Already vendored in `cmd/site/static/js/htmx.min.js`.

**Shared stylesheet**: Symlinked at `cmd/site/static/css/style.css` from `cmd/design/`. All templates use the classes defined in DESIGN.md.

## Compute Strategy (applies to all visualizations)

These are not guidelines — they are hard requirements.

**Precompute gram matrices.** For any visualization involving permutation tests or distance-based statistics (viz 2, 3, 5), compute the full pairwise distance or kernel matrix once. Each permutation re-indexes into the precomputed matrix. No recomputation of distances per permutation.

**Parallelize independent cells.** For grid sweeps (viz 3: ρ × p heatmap), each cell is independent. Use a semaphore pattern (`chan struct{}` of fixed capacity) to bound concurrency. Reasonable default: `runtime.NumCPU()`.

**Log progress to stderr.** For any computation that takes more than a few seconds, log progress lines to stderr: which cell is running, how many remain. Use `log.Printf` or `fmt.Fprintf(os.Stderr, ...)`. No progress bars. No TUI libraries.

**Precompute on server start.** Long-running computations (viz 3 grid sweep, viz 5 permutation tests) run at server startup. Results are cached in memory and served instantly via HTMX. The server logs progress to stderr during startup. The user sees the pages only after precomputation completes. If total startup time exceeds ~60s, log estimated remaining time.

**Random seed.** Use a fixed seed (e.g., `rand.NewSource(42)`) for all sample generation so results are reproducible across restarts. Each visualization should use its own distinct seed to avoid correlated streams.

## Visualization 1: DD-Plot Behavior Under Distributional Shifts

**Route:** `GET /robustness/dd-plots`
**Template:** `templates/dd-plots.tmpl`

### What it shows

A DD-plot (depth-vs-depth plot) plots D(z; F₁) vs D(z; F₂) for every observation z in the pooled sample. When F₁ = F₂, points cluster along the diagonal. Different types of distributional shift produce different departures from the diagonal.

### Data generation

Generate two 2D Gaussian samples, n=200 each. F₁ is always N(0, I₂).

Three panels:
- **(a) Location shift:** F₂ = N([1.5, 0], I₂). Shift of +1.5 in dim0 only.
- **(b) Scale shift:** F₂ = N(0, diag(4, 1)). 2× standard deviation in dim0 (so 4× variance).
- **(c) Correlation shift:** F₂ = N(0, Σ) where Σ has σ₁² = σ₂² = 1 and ρ = 0.8. F₁ has independent components.

### Depth computation

Use approximate halfspace (Tukey) depth. For 2D data, this is tractable:

Use approximate halfspace (Tukey) depth. For 2D data, this is tractable:

For each point z, for K=1,000 uniformly spaced directions u₁, ..., u_K on the unit circle, project all reference sample points and z onto each direction, then compute the fraction of the reference sample at or below z in that projection:

```
depth(z; X) = min_k  (1/n) × |{i : u_k'X_i ≤ u_k'z}|
```

For a point near the center of a symmetric distribution, all directions give fractions near 0.5, so depth ≈ 0.5. For a point on the boundary, some direction gives a fraction near 0. 1,000 directions for consistency with the ported codebase (which uses 1,000 at p=18). Overkill at p=2 but fast regardless.

**Important:** do NOT use `min(fraction_left, fraction_right)` — that formulation coincides with the correct answer for symmetric distributions but diverges for asymmetric cases, which is exactly the regime panels (b) and (c) test.

Compute D(z; F₁) = depth of z relative to sample 1, D(z; F₂) = depth of z relative to sample 2, for every z in the pooled set of 400 observations.

### Layout

Three canvas panels arranged horizontally. Each panel is a scatterplot:
- X-axis: D(z; F₁)
- Y-axis: D(z; F₂)
- Diagonal reference line (y = x) in `--rule` color
- Points from sample 1 in one color, sample 2 in another
- Panel label above each: "Location shift (Δμ = 1.5)", "Scale shift (2× σ)", "Correlation shift (ρ = 0.8)"

### Interactivity

Slider controls for each shift parameter:
- Panel (a): shift magnitude, range [0, 3], default 1.5
- Panel (b): scale multiplier, range [1, 4], default 2
- Panel (c): correlation ρ, range [0, 0.95], default 0.8

When a slider changes, HTMX hits the handler, which regenerates the sample with the new parameter, recomputes depths, and returns JSON point data. Client JS redraws the affected canvas. The other two panels remain unchanged (each slider only affects its panel).

Use `hx-trigger="input changed delay:200ms"` — depth computation for n=200 in 2D with 500 directions should complete in <500ms per panel.

### Handler endpoints

- `GET /robustness/dd-plots/data?panel=a&shift=1.5` → JSON: `{points: [{x: depth1, y: depth2, group: 0|1}, ...], panel: "a"}`
- Same pattern for `panel=b&scale=2.0` and `panel=c&rho=0.8`

Precompute default state on startup. Slider changes trigger recomputation.

---

## Visualization 2: Concentration of Measure

**Route:** `GET /robustness/concentration`
**Template:** `templates/concentration.tmpl`

### What it shows

As dimensionality p increases, pairwise Euclidean distances between random points concentrate around their expected value. All points become approximately equidistant. This is why distance-based two-sample tests lose power in high dimensions.

### Data generation

Fix n=200. For each p in {2, 5, 10, 18, 50}, draw n points from N(0, I_p). Compute all n(n-1)/2 = 19,900 pairwise Euclidean distances.

### Layout

Single canvas, full width. Overlaid histograms (or kernel density estimates) of pairwise distances, one curve per value of p. Each dimension gets a distinct color. Use ~50 bins or a Gaussian KDE.

**Normalize x-axis:** Divide all distances by √p so curves are on a comparable scale and the concentration effect is visible as narrowing rather than shifting. Label x-axis as "d(x,y) / √p".

Y-axis: density (normalized so area under each curve is 1).

Legend: one entry per dimension value.

### Interactivity

Toggle individual dimension curves on/off by clicking legend entries. This is client-side JS only — the data is precomputed once and embedded in the page.

Optional slider for n (50, 100, 200, 500) — if included, this triggers a server recomputation via HTMX.

### Handler endpoints

- `GET /robustness/concentration/data?n=200` → JSON: `{dimensions: {2: [distances...], 5: [...], ...}}` (or precomputed histogram bin counts to avoid sending 5 × 19,900 floats)

Prefer sending histogram bin counts + edges rather than raw distances. ~50 bins per dimension = ~500 numbers total.

Precompute default (n=200) on startup.

---

## Visualization 3: C2ST Power Surface

**Route:** `GET /robustness/c2st`
**Template:** `templates/c2st.tmpl`

### What it shows

The Classifier Two-Sample Test (C2ST) trains a classifier to distinguish samples from F₁ vs F₂. If it succeeds (accuracy significantly above 50%), the distributions differ. This heatmap shows how test power depends on the strength of the distributional difference (ρ) and the dimensionality (p).

### Data generation and test procedure

For each cell (ρ, p):
1. Draw n=150 from F₁ = N(0, I_p) (class 0)
2. Draw n=150 from F₂ = N(0, Σ_p) where Σ_p has unit diagonal, ρ correlation between dim0 and dim1, and zero correlation elsewhere (class 1)
3. Combine into 300 observations with labels
4. Run 5-fold CV using the bagged CART ensemble from `stage2_portable.go` (100 depth-5 trees, Gini, bootstrap aggregated)
5. Record mean CV accuracy across folds
6. Test whether accuracy significantly exceeds 0.5 using a one-sided Binomial test: p-value = P(Binomial(n_test, 0.5) ≥ observed_correct). Here n_test = total test observations across all folds (= 300, since each observation is tested exactly once in 5-fold CV)
7. Reject if p-value < 0.05

**Note on the independence assumption:** The binomial test treats the 300 pooled test predictions as independent Bernoulli(0.5) trials. They are not strictly independent — predictions from different folds come from classifiers trained on different subsets. This is a known approximation (see Stage 2 implementation review, Checklist B). It is intentional and consistent with the ported code. Do not attempt to correct for it.

Repeat 30 times per cell. Rejection rate = fraction of 30 reps where p-value < 0.05.

### Grid

- ρ axis: [0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9] — 7 values
- p axis: [3, 5, 8, 12, 16, 20] — 6 values
- Total: 42 cells × 30 reps = 1,260 ensemble fits

### Parallelization

Each of the 42 cells is independent. Use a semaphore of capacity `runtime.NumCPU()`. Log to stderr as each cell completes:

```
c2st: cell (ρ=0.30, p=3) complete, rejection rate = 0.13 [1/42]
c2st: cell (ρ=0.30, p=5) complete, rejection rate = 0.10 [2/42]
...
```

### Layout

Single heatmap on canvas. X-axis: ρ. Y-axis: p (low at top or bottom — your call, just be consistent). Cell color encodes rejection rate from 0.0 (no power, light) to 1.0 (full power, dark). Use a sequential single-hue colormap — terra cotta (`#C0522A`) at full intensity for 1.0, fading to near-white for 0.0.

Numeric rejection rate printed inside each cell.

### Interactivity

This is precomputed. The heatmap is static. Hover on a cell to see a tooltip with: ρ, p, rejection rate, mean accuracy ± std across the 30 reps.

Tooltip is client-side JS, no HTMX needed.

### Handler endpoints

- `GET /robustness/c2st/data` → JSON: `{cells: [{rho: 0.3, p: 3, rejection_rate: 0.13, mean_acc: 0.52, std_acc: 0.03}, ...], rho_values: [...], p_values: [...]}`

All precomputed on server start.

---

## Visualization 4: Shared-Denominator Spurious Correlation

**Route:** `GET /robustness/spurious-correlation`
**Template:** `templates/spurious-correlation.tmpl`

### What it shows

Dividing independent random variables by a shared random denominator induces positive correlation. This is relevant for PnL analysis where raw returns are often normalized by a shared factor (portfolio value, volatility estimate, notional). The correlation matrix before division shows near-zero off-diagonals. After division, substantial positive correlation appears everywhere.

### Data generation

1. Generate X: a 5×1000 matrix of independent N(0,1) draws (five independent PnL components, 1000 observations each)
2. Generate D: a 1×1000 vector of Lognormal(0, 0.3) draws (shared normalizing quantity). Lognormal ensures strictly positive.
3. Compute Y = X ./ D (element-wise: each row of X divided by D)
4. Compute 5×5 correlation matrix of X (rows are variables, columns are observations) → "Before"
5. Compute 5×5 correlation matrix of Y → "After"

### Layout

Two 5×5 heatmaps side by side. Left: "Before division". Right: "After division".

Color scale: diverging, centered at 0. Use teal (`#2E7D8C`) for positive, terra cotta (`#C0522A`) for negative, white/paper for zero. Diagonal is always 1.0.

Numeric correlation value printed inside each cell (2 decimal places).

Axis labels: generic component names (C₁, C₂, C₃, C₄, C₅).

### Interactivity

Slider for the lognormal σ parameter (controls how variable the shared denominator is). Range [0.01, 1.0], default 0.3. Higher σ → more variable denominator → stronger spurious correlation.

Slider for number of components (2–10, default 5). Changing this resizes the matrices.

HTMX triggers recomputation. Response is JSON with two flattened correlation matrices plus dimensions.

### Handler endpoints

- `GET /robustness/spurious-correlation/data?sigma=0.3&components=5` → JSON: `{before: [[...], ...], after: [[...], ...], labels: ["C₁", "C₂", ...], sigma: 0.3}`

Fast computation — matrix multiply and correlation on 5×1000. No precomputation needed; compute per request.

---

## Visualization 5: Energy Distance vs. MMD Sensitivity

**Route:** `GET /robustness/energy-vs-mmd`
**Template:** `templates/energy-vs-mmd.tmpl`

### What it shows

Energy distance and MMD (Maximum Mean Discrepancy) are both kernel/distance-based two-sample test statistics, but they have different sensitivity profiles. This visualization shows which test detects which type of distributional difference more readily.

### Data generation and test procedure

Four shift types, each producing a pair of 2D samples (n=200 per group):

**(a) Location shift:** F₁ = N(0, I₂), F₂ = N([1.0, 0], I₂)

**(b) Scale shift:** F₁ = N(0, I₂), F₂ = N(0, diag(4, 1)). 2× std dev in dim0.

**(c) Correlation shift:** F₁ = N(0, I₂), F₂ = N(0, Σ) with ρ=0.7 and unit marginal variances.

**(d) Tail weight shift:** F₁ = N(0, I₂), F₂ = t₅ scaled to unit variance. Draw raw t₅ samples and multiply by √((ν-2)/ν) = √(3/5) ≈ 0.7746 so each marginal has variance 1, matching F₁. No existing codebase convention — this is the standard scaling.

### Statistics

**Energy distance (Székely-Rizzo):** E(F₁, F₂) = 2E||X-Y||₂ - E||X-X'||₂ - E||Y-Y'||₂ where expectations are replaced by sample averages over all pairs. Euclidean (L2) norm throughout — do not confuse with the L1 norm used in the MMD kernel below.

**MMD (Laplacian kernel):** K(x,y) = exp(-||x-y||₁ / σ) where σ is the median of all pairwise L1 distances in the pooled sample (median bandwidth heuristic). MMD² = E[K(X,X')] + E[K(Y,Y')] - 2E[K(X,Y)], estimated by U-statistics over the sample.

### Permutation testing

For each shift type and each statistic:
1. Precompute the gram matrix (Euclidean distance matrix for energy, kernel matrix for MMD). This is a 400×400 matrix.
2. Compute the observed test statistic from the gram matrix using the original group labels.
3. Permute group labels B=500 times. For each permutation, recompute the test statistic by re-indexing into the precomputed gram matrix. No distance recomputation.
4. p-value = (1 + number of permutation statistics ≥ observed) / (1 + B).

### Repeated runs

Run 50 repetitions per shift type. Record the p-value from each. Rejection rate = fraction with p-value < 0.05.

Total: 4 shift types × 2 statistics × 50 reps × 501 evaluations (1 observed + 500 permutations) per rep. The gram matrix precomputation is the expensive part; permutation re-indexing is cheap.

### Parallelization

Parallelize across the 4 × 2 × 50 = 400 independent (shift, statistic, rep) triples. Semaphore of `runtime.NumCPU()`. Log progress to stderr.

### Layout

A grouped bar chart. X-axis: four shift types (Location, Scale, Correlation, Tail). For each shift type, two bars side by side: energy distance rejection rate (terra cotta) and MMD rejection rate (teal). Y-axis: rejection rate [0, 1]. Horizontal reference line at 0.05 (nominal size).

Below the chart: a small table showing exact rejection rates and mean p-values for each (shift, statistic) pair.

### Interactivity

Precomputed and static. Hover on bars for exact values. No sliders — the point is the comparison, not parameter exploration.

Optional toggle: show p-value distributions (histograms of the 50 p-values) for each (shift, statistic) pair. This is client-side — data is embedded, JS toggles visibility.

### Handler endpoints

- `GET /robustness/energy-vs-mmd/data` → JSON: `{shifts: ["location", "scale", "correlation", "tail"], results: [{shift: "location", energy: {rejection_rate: ..., mean_pvalue: ..., pvalues: [...]}, mmd: {rejection_rate: ..., mean_pvalue: ..., pvalues: [...]}}, ...]}`

All precomputed on server start.

---

## File Summary

```
cmd/site/
├── main.go                              # add routes + startup precomputation calls
├── handlers/
│   └── robustness.go                    # all five robustness visualizations
├── templates/
│   ├── layout.tmpl                      # (existing)
│   ├── index.tmpl                       # add links to five new pages
│   ├── dd-plots.tmpl
│   ├── concentration.tmpl
│   ├── c2st.tmpl
│   ├── spurious-correlation.tmpl
│   └── energy-vs-mmd.tmpl
└── static/                              # (existing, no changes)

internal/
└── ml/
    └── stage2_portable.go               # user-provided, pre-existing
```

## Startup Sequence

In `main.go`, before starting the HTTP server:

```
log.Println("precomputing visualizations...")

// These can run concurrently
log.Println("  dd-plots: computing default state...")
ddplotsData := handlers.PrecomputeDDPlots()

log.Println("  concentration: computing pairwise distances...")
concentrationData := handlers.PrecomputeConcentration()

log.Println("  c2st: computing power surface (this takes a while)...")
c2stData := handlers.PrecomputeC2ST()    // logs per-cell progress to stderr

log.Println("  energy-vs-mmd: computing permutation tests...")
energyMMDData := handlers.PrecomputeEnergyMMD()  // logs progress to stderr

// spurious correlation is fast, computed per-request

log.Println("precomputation complete, starting server")
```

Pass precomputed data to handlers via struct fields or closure — not global variables. Each handler owns its precomputed state.

## Index Page Updates

Add to `templates/index.tmpl`:

```
DD-plots under distributional shifts        /robustness/dd-plots
Concentration of measure                     /robustness/concentration
C2ST power surface                           /robustness/c2st
Shared-denominator spurious correlation      /robustness/spurious-correlation
Energy distance vs. MMD sensitivity          /robustness/energy-vs-mmd
```

## Canvas Rendering Notes

All five visualizations render on `<canvas>`. Client JS receives JSON from precomputed endpoints or HTMX responses and draws. The following applies to all:

- Canvas axis labels rendered at 13px in IBM Plex Mono (load via `new FontFace` or accept system monospace fallback on canvas)
- Grid lines: `--rule` color (`#D6D2C9`)
- Axis lines: `--ink-faint` (`#A8A49C`)
- Axis tick labels: `--ink-faint`
- All displayed numbers rounded to appropriate precision (2 decimals for correlations and p-values, 3 for rejection rates, integers for counts)

### Visualization color assignments

| Viz | Element | Color |
|-----|---------|-------|
| 1 (DD-plots) | Sample 1 points | `#2E7D8C` (teal) |
| 1 (DD-plots) | Sample 2 points | `#C0522A` (terra cotta) |
| 1 (DD-plots) | Diagonal reference | `#D6D2C9` (rule) |
| 2 (Concentration) | p=2 | `#C0522A` (terra cotta) |
| 2 (Concentration) | p=5 | `#2E7D8C` (teal) |
| 2 (Concentration) | p=10 | `#6B6963` (ink-light) |
| 2 (Concentration) | p=18 | `#8B6914` (amber) |
| 2 (Concentration) | p=50 | `#534AB7` (purple) |
| 3 (C2ST) | Low rejection | near-`#F5F2EB` (paper) |
| 3 (C2ST) | High rejection | `#C0522A` (terra cotta) |
| 4 (Spurious) | Positive correlation | `#2E7D8C` (teal) |
| 4 (Spurious) | Negative correlation | `#C0522A` (terra cotta) |
| 4 (Spurious) | Zero correlation | `#F5F2EB` (paper) |
| 5 (Energy vs MMD) | Energy distance bars | `#C0522A` (terra cotta) |
| 5 (Energy vs MMD) | MMD bars | `#2E7D8C` (teal) |
| 5 (Energy vs MMD) | Nominal size line | `#A8A49C` (ink-faint) |

## What This Does Not Decide

- Expression parsing or a general math evaluator (not needed for these visualizations)
- The future JSON API for component composition
- Voice input or runtime LLM integration
- HTMX interaction patterns beyond what's specified per visualization
- How `stage2_portable.go` is internally structured (it's treated as a dependency)
