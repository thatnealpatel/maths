# PREPLAN.md — Math Visualizer Shared Stylesheet & Sampler

## Context

This is a **pre-planning** document for a shared stylesheet that will eventually power a larger tool: a CLI that takes Wikipedia math pages and produces self-contained Go binaries serving interactive mathematical visualizations via Go HTML templates + HTMX.

Right now, we need to **nail down the stylesheet** by building a standalone HTML sampler page that demonstrates every design element. The sampler must be visually iterable — open it in a browser, tweak, reload.

## Design Direction

**Typewriter-monospace on off-white paper.** The aesthetic is a mathematics manuscript typed on good paper. Flat, monochromatic for all structural/UI elements. Visualizations (plots, diagrams, canvases) are explicitly unconstrained — they can use any colors.

### Palette (CSS custom properties)

```
--paper:     #F5F2EB   /* page background, warm off-white */
--ink:       #2C2B28   /* primary text, darkest */
--ink-light: #6B6963   /* body text, secondary */
--ink-faint: #A8A49C   /* labels, hints, disabled */
--rule:      #D6D2C9   /* borders, dividers, grid lines */
--input-bg:  #EDEAE2   /* input fields, math blocks, code backgrounds */
```

No other colors in the UI layer. Visualizations are exempt.

### Typography

Provide a **font toggle** at the top of the sampler so the user can switch the entire page between these three monospace families and visually compare:

1. **IBM Plex Mono** — clean, modern monospace. Weights: 300, 400, 500, 600.
2. **JetBrains Mono** — programming-oriented, slightly wider. Weights: 300, 400, 500, 600.
3. **Courier Prime** — true typewriter feel, narrower. Weights: 400, 700.

Load all three via Google Fonts (`fonts.googleapis.com`). Each toggle button should render in its own font so you can compare at a glance.

**Type scale** (all monospace, inheriting the active font):
- `h1`: 1.5rem, weight 500, letter-spacing -0.01em
- `h2`: 1.125rem, weight 500
- `h3`: 0.875rem, weight 500, color `--ink-light`
- Body `p`: 0.8125rem, weight 400, line-height 1.7, color `--ink-light`
- Labels/section headers: 0.6875rem, uppercase, letter-spacing 0.15em, color `--ink-faint`
- Inline math: 0.8125rem, background `--input-bg`, padding 0.1em 0.35em

### Design Rules

- **Zero border-radius everywhere.** All corners are sharp — inputs, buttons, containers, tags.
- **Square slider thumbs.** Not round. 12×12px, background `--ink`, no border-radius.
- **Slider track:** 2px height, color `--rule`.
- **Borders:** Always 1px solid `--rule` (default) or `--ink` (focus/emphasis). Never thicker except the math-block left border (2px solid `--ink`).
- **No shadows, no gradients, no glow, no blur.** Completely flat.
- **No images or icons.** Pure typographic/structural UI.
- **Button hover:** invert — background becomes `--ink`, text becomes `--paper`.

## Sampler Page Structure

Build a single `index.html` that demonstrates every element below, organized in labeled sections. Each section has a small uppercase label divider (like `TYPOGRAPHY`, `INPUTS & CONTROLS`, etc.) with a 1px bottom border.

### 1. Font Toggle
Row of three buttons at the top. Active button gets inverted (dark bg, light text). Clicking switches `font-family` on the entire page, including canvas text and labels.

### 2. Typography Section
Demonstrate the full type hierarchy:
- h1: "Fourier transform"
- h2: "Continuous Fourier transform"  
- h3: "Definition and properties"
- Body paragraph with inline math spans
- Display math block (border-left style) showing the Fourier transform integral:
  `f̂(ξ) = ∫_{-∞}^{∞} f(x) e^{-2πixξ} dx`
  Use HTML entities / Unicode for math symbols. This is a stylesheet sampler, not a MathJax demo.

### 3. Inputs & Controls Section
- Text input (expression field, e.g., `sin(x) / x`)
- Number inputs (range min/max)
- Select dropdown (sample count: 256, 512, 1024)
- Range slider with live numeric readout
- Primary button ("Evaluate") — 1px `--ink` border, uppercase, letter-spaced
- Secondary button ("Reset") — 1px `--rule` border, muted text
- Tags/badges — inline, 1px `--rule` border, small text (e.g., "L¹ integrable", "continuous")

### 4. Visualization Area
A bordered container (`1px solid --rule`) with:
- A small uppercase label inside the top ("f(x) = sin(x) / x")
- A `<canvas>` element with a live plot of `sin(x)/x`
- Grid lines in `--rule`, axis lines in `--ink-faint`
- The plot curve itself in a warm color like `#C0522A` (terra cotta)
- A second dashed curve in `#2E7D8C` (teal) representing `cos(2πξx)` controlled by the frequency slider from section 3
- Axis labels rendered on the canvas in the active monospace font
- A small legend in the top-left corner of the canvas

**The frequency slider must redraw the canvas live.** This demonstrates that the stylesheet supports interactive visualizations.

### 5. Proof Steps
An ordered list styled with a custom counter:
- Step numbers as `decimal-leading-zero` (01, 02, 03...) in `--ink-faint`
- Step text in `--ink-light`
- 1px `--rule` bottom border between steps
- Inline math spans within step text

### 6. Dividers
Use `<hr>` styled as 1px `--rule` between major sections. 2rem vertical margin.

## File Structure

```
sampler/
├── index.html      # the sampler page, self-contained
├── style.css       # the shared stylesheet (extracted, reusable)
└── README.md       # brief: "open index.html in a browser, use font toggle to compare"
```

`style.css` should be the **canonical shared stylesheet** that the larger tool will eventually import. `index.html` links to it and adds only sampler-specific layout (the section organization, font toggle JS, canvas drawing JS).

## What This Is NOT

- Not the full math visualizer tool. That comes later.
- Not a Go project. This is pure HTML/CSS/JS for stylesheet iteration.
- Not a design system or component library. It's a stylesheet with a sampler.
- Not production code. It's a reference and iteration artifact.

## Success Criteria

Open `index.html` in a browser. You should be able to:
1. Toggle between three fonts and see the entire page reflow
2. See clean typographic hierarchy that feels like typed mathematics
3. Interact with form controls that are visually cohesive
4. See the canvas plot redraw when moving the frequency slider
5. Feel like you're reading a well-typeset math manuscript, not a web app
