# PLAN.md — Deflated Sharpe Ratio Explorer

## Overview

Interactive visualization for exploring the Deflated Sharpe Ratio (Bailey & López de Prado 2014) across the parameter space relevant to SPX options strategy permutation runs. The DSR answers: "is the best observed Sharpe ratio significant after correcting for non-normality and the number of strategies tested?" This visualization makes the tradeoffs between those corrections tangible.

## Prerequisites

Same architecture as the existing visualizations: Go HTML template, HTMX for server-side recomputation, shared stylesheet from `cmd/design/`, canvas rendering for charts.

No external dependencies. The DSR is a closed-form formula — no permutation tests, no tree ensembles, no precomputation needed. All computation is per-request and completes in microseconds.

## The DSR Formula

The Probabilistic Sharpe Ratio (PSR) tests whether an observed Sharpe ratio SR̂ exceeds a reference value SR*:

```
PSR(SR*) = Φ((SR̂ − SR*) / √(V / T))
```

where:

```
V = 1 − γ₃·SR̂ + (γ₄ − 1)/4 · SR̂²
```

γ₃ = skewness, γ₄ = kurtosis, T = number of return observations, Φ = standard normal CDF.

The deflation comes from setting SR* to the expected maximum Sharpe ratio under the null across N independent trials:

```
SR* ≈ √(V/T) · ((1 − γ) · Φ⁻¹(1 − 1/N) + γ · Φ⁻¹(1 − 1/(N·e)))
```

where γ ≈ 0.5772 (Euler-Mascheroni constant), N = number of strategies tested.

A strategy passes Stage 1 if PSR(SR*) exceeds a significance threshold (e.g., 0.95 for α = 0.05 one-sided).

The Minimum Track Record Length (MinTRL) is the smallest T such that PSR(SR*) ≥ 0.95 for given SR̂, γ₃, γ₄, N:

```
MinTRL = min T such that (SR̂ − SR*) / √(V/T) ≥ Φ⁻¹(0.95)
```

Solve numerically (bisection on T).

## Route

`GET /dsr`

## Handler and Template

`handlers/dsr.go`, `templates/dsr.tmpl`

## Degrees of Freedom

Six input parameters, each with a slider or numeric input:

| Parameter | Symbol | Range | Default | Step | Meaning |
|---|---|---|---|---|---|
| Observed Sharpe | SR̂ | 0.0 – 4.0 | 1.5 | 0.05 | The best Sharpe ratio from the permutation run |
| Number of strategies | N | 1 – 10,000 | 1,000 | 1 (log-scale slider) | Total strategies tested in the permutation run |
| Return observations | T | 50 – 5,000 | 2,000 | 10 | Number of daily returns per strategy (~252/year × years) |
| Skewness | γ₃ | −6.0 – 0.0 | −1.43 | 0.01 | Return distribution skewness (negative for short premium) |
| Kurtosis | γ₄ | 3.0 – 40.0 | 8.63 | 0.1 | Return distribution kurtosis (3.0 = normal) |
| Significance level | α | 0.01 – 0.10 | 0.05 | 0.01 | One-sided test threshold |

Default γ₃ and γ₄ correspond to the WPUT (weekly, 2–9 DTE proxy) from Bondarenko 2019. Preset buttons for other DTE buckets:

| Preset | γ₃ | γ₄ | Label |
|---|---|---|---|
| Normal | 0.0 | 3.0 | "Gaussian (reference)" |
| 2–9 DTE | −1.43 | 8.63 | "WPUT weekly" |
| ~45 DTE | −2.09 | 12.58 | "PUT monthly" |
| 0–1 DTE | −4.5 | 33.6 | "0DTE straddle sellers" |

Clicking a preset sets γ₃ and γ₄ and triggers recomputation. All other parameters remain unchanged.

## Layout

Four panels arranged in a 2×2 grid. All four update simultaneously when any slider changes.

### Panel 1 (top-left): DSR Verdict

Large single-value display showing:

- **SR*** (the deflated threshold) — large font
- **PSR(SR*)** (the probability that the observed SR exceeds SR* by chance) — large font
- **Pass / Fail** indicator (PSR ≥ 1 − α → Pass, else Fail) — color-coded (teal = pass, terra cotta = fail)
- **V** (the non-normality-adjusted variance) — smaller, below
- **SE inflation** — ratio √(V / V_normal) where V_normal = 1 + SR̂²/2, showing how much non-normality inflates the standard error

No canvas — this is HTML text, styled with the shared stylesheet.

### Panel 2 (top-right): SR* as a function of N

Line chart on canvas. X-axis: N from 1 to 10,000 (log scale). Y-axis: SR*. A single curve showing how the required threshold grows with N. Vertical dashed line at the current N slider value. Horizontal dashed line at the current SR̂ value. The region where SR̂ > SR* is shaded lightly in teal (pass region). The region where SR̂ < SR* is unshaded (fail region).

The intersection of the SR̂ horizontal and the SR*(N) curve is the maximum N at which this Sharpe ratio would still pass — label this point with its N value.

Redraws when any of SR̂, T, γ₃, γ₄, α change (since SR* depends on all of them via V).

### Panel 3 (bottom-left): MinTRL as a function of SR̂

Line chart on canvas. X-axis: SR̂ from 0.5 to 4.0. Y-axis: MinTRL (number of daily observations needed). Two curves:

1. MinTRL under the current (γ₃, γ₄) — terra cotta
2. MinTRL under normality (γ₃ = 0, γ₄ = 3) — dashed, `--ink-faint`

The gap between the two curves shows the cost of non-normality in terms of additional track record required. Vertical dashed line at the current SR̂ value. Label the two MinTRL values at that SR̂.

At the current N. Redraws when N, γ₃, γ₄, α change.

### Panel 4 (bottom-right): PSR sensitivity to skewness and kurtosis

Heatmap on canvas. X-axis: γ₃ from −6 to 0. Y-axis: γ₄ from 3 to 40. Cell color: PSR(SR*) value from 0 (terra cotta) to 1 (teal), with the α threshold contour drawn as a bold line (the boundary between pass and fail).

Fixed at current SR̂, N, T. The four DTE presets are marked on the heatmap as labeled points. A crosshair shows the current (γ₃, γ₄) position.

This panel answers: "for my observed Sharpe, my N, and my T, which parts of the skewness-kurtosis space would pass?"

## Interactivity

All sliders use `hx-trigger="input changed delay:100ms"` — the DSR computation is microseconds, so near-instant response.

HTMX hits a single endpoint that returns all four panels' data:

```
GET /dsr/data?sr=1.5&n=1000&t=2000&skew=-1.43&kurt=8.63&alpha=0.05
```

Response JSON:

```json
{
  "verdict": {
    "sr_star": 1.23,
    "psr": 0.87,
    "pass": false,
    "v": 3.14,
    "se_inflation": 1.45
  },
  "sr_star_curve": {
    "n_values": [1, 2, 5, 10, ...],
    "sr_star_values": [0.0, 0.12, ...]
  },
  "mintrl_curves": {
    "sr_values": [0.5, 0.55, ...],
    "mintrl_actual": [4500, 3800, ...],
    "mintrl_normal": [2100, 1800, ...]
  },
  "psr_heatmap": {
    "skew_values": [-6.0, -5.8, ...],
    "kurt_values": [3.0, 4.0, ...],
    "psr_grid": [[0.99, 0.98, ...], ...],
    "pass_contour": [[skew, kurt], ...]
  }
}
```

All computed server-side per request. No precomputation needed.

## Canvas Rendering

Same conventions as other visualizations:

- Axis labels: 13px IBM Plex Mono
- Grid lines: `--rule` (`#D6D2C9`)
- Axis lines/labels: `--ink-faint` (`#A8A49C`)

| Element | Color |
|---|---|
| SR*(N) curve | `#2E7D8C` (teal) |
| Pass region shading | `#2E7D8C` at 10% opacity |
| Fail indicator | `#C0522A` (terra cotta) |
| Pass indicator | `#2E7D8C` (teal) |
| MinTRL (non-normal) | `#C0522A` (terra cotta) |
| MinTRL (normal reference) | `#A8A49C` dashed (ink-faint) |
| Heatmap low PSR | `#C0522A` (terra cotta) |
| Heatmap high PSR | `#2E7D8C` (teal) |
| Heatmap pass/fail contour | `#3D3A35` bold (ink) |
| DTE preset markers | `#3D3A35` (ink) with label |
| Current value crosshairs | `#3D3A35` dashed (ink) |

## Compute Notes

The DSR formula involves only Φ (standard normal CDF) and Φ⁻¹ (inverse normal CDF). Use `gonum/stat/distuv` Normal distribution or implement directly — both are trivial.

MinTRL requires solving for T numerically. Bisection on T in [1, 100000] with tolerance 1 converges in ~17 iterations. Compute for each SR̂ value in the curve (say 70 points from 0.5 to 4.0) — total: ~1,200 bisection iterations per request. Microseconds.

The PSR heatmap grid: 30 γ₃ values × 37 γ₄ values = 1,110 cells, each requiring one Φ evaluation. Microseconds.

The SR*(N) curve: ~100 N values on log scale, each requiring two Φ⁻¹ evaluations. Microseconds.

Total per-request compute: well under 1ms. No precomputation, no caching needed.

## File Summary

```
cmd/site/
├── handlers/
│   └── dsr.go           # DSR computation + JSON endpoint
└── templates/
    └── dsr.tmpl          # four-panel layout with sliders
```

Add route in `main.go` and link on `index.tmpl`.

## What This Visualization Answers

For a practitioner deciding how to configure a permutation run:

1. **"At my current N, what Sharpe ratio do I need to pass?"** → Panel 2, read the SR* at the vertical dashed line.
2. **"How many permutations can I run before my best strategy can't pass?"** → Panel 2, read the N at the intersection of SR̂ horizontal and SR* curve.
3. **"How much longer would my backtest need to be to pass?"** → Panel 3, read MinTRL at the current SR̂.
4. **"How much does non-normality cost me?"** → Panel 3, gap between the two curves. Panel 1, SE inflation ratio.
5. **"If my returns were less skewed, would this strategy pass?"** → Panel 4, move the crosshair toward (0, 3) and see if it enters the pass region.
6. **"At 2–9 DTE vs. 0–1 DTE, how much harder is it to pass?"** → Panel 4, compare the preset markers' positions relative to the pass/fail contour.
