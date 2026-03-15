# DESIGN.md — mathvis Design Language

Finalized design language for the math visualizer tool. Derived from the interactive sampler at `cmd/design/`. The canonical stylesheet lives at `cmd/design/static/css/style.css`.

## Aesthetic

Typewriter-monospace on off-white paper. A mathematics manuscript typed on good stock. Flat, monochromatic for all structural/UI elements. Visualizations (plots, diagrams, canvases) are explicitly unconstrained and may use any colors.

## Font

**IBM Plex Mono** via Google Fonts. Weights: 300, 400, 500, 600. Fallback: `monospace`.

```
--font-family: 'IBM Plex Mono', monospace;
```

## Palette

Six CSS custom properties. No other colors in the UI layer. Visualizations are exempt.

```
--paper:     #F5F2EB   /* page background, warm off-white */
--ink:       #2C2B28   /* primary text, headings */
--ink-light: #6B6963   /* body text, secondary content */
--ink-faint: #A8A49C   /* labels, hints, disabled, counters */
--rule:      #D6D2C9   /* borders, dividers, grid lines */
--input-bg:  #EDEAE2   /* input fields, math blocks, code backgrounds */
```

## Type Scale

All sizes in rem at a 16px base.

| Role                  | Size       | Weight | Extra                              |
|-----------------------|------------|--------|------------------------------------|
| h1                    | 1.6875rem  | 500    | letter-spacing: -0.01em            |
| h2                    | 1.3125rem  | 500    |                                    |
| h3                    | 1.0625rem  | 500    | color: --ink-light                 |
| Body (p)              | 1rem       | 400    | line-height: 1.7, color: --ink-light |
| Labels / section hdrs | 0.875rem   | 500    | uppercase, letter-spacing: 0.15em, color: --ink-faint |
| Buttons               | 0.875rem   | 500    | uppercase, letter-spacing: 0.15em  |
| Proof step counters   | 0.9375rem  | 400    | color: --ink-faint                 |
| Math inline           | 1rem       | —      | background: --input-bg             |
| Math block            | 1rem       | —      | line-height: 1.8, border-left: 2px solid --ink |
| Canvas labels         | 13px       | —      | rendered on canvas in active font  |

## Design Rules

- Zero border-radius everywhere. All corners sharp.
- Square slider thumbs: 12×12px, background `--ink`, no border-radius.
- Slider track: 2px height, color `--rule`.
- Borders: always 1px solid `--rule` (default) or `--ink` (focus/emphasis). Never thicker except Math--Block left border (2px solid `--ink`).
- No shadows, no gradients, no glow, no blur. Completely flat.
- No images or icons. Pure typographic/structural UI.
- Button hover: invert — background becomes `--ink`, text becomes `--paper`.
- Dividers (`<hr>`): 1px solid `--rule`, 2rem vertical margin.

## CSS Naming Convention

`Component--Thing` using PascalCase components, mimicking Go naming:

```
.Section--Label
.Math--Inline
.Math--Block
.Btn
.Btn--Primary
.Btn--Secondary
.Tag
.Controls--Row
.Controls--Group
.Controls--Expression
.Controls--Number
.Controls--Slider
.Controls--Readout
.Controls--Actions
.Controls--Tags
.Vis
.Vis--Label
.Vis--Canvas
.Proof
.Proof--Step
```

No CSS comments. Class names are self-documenting.

## Visualization Colors

Exempt from the monochromatic UI palette. Current sampler uses:

- `#C0522A` (terra cotta) — primary plot curve
- `#2E7D8C` (teal) — secondary/overlay curve, dashed

Grid lines use `--rule`. Axis lines use `--ink-faint`. Axis labels use `--ink-faint` at 13px in the active font.

## Layout

- Max content width: 850px, centered, 1.5rem horizontal padding.
- Sections separated by `<hr>` with uppercase label dividers (`.Section--Label`).

## File Structure

```
cmd/design/
├── main.go                  # Go server, embeds static/ and index.tmpl
├── index.tmpl               # sampler page template
└── static/
    ├── css/
    │   └── style.css        # canonical shared stylesheet
    └── js/
        └── sampler.js       # font toggle + canvas drawing (sampler-only)
```

`style.css` is the reusable artifact. The sampler template and JS are iteration tools.

## What This Decides

- Font: IBM Plex Mono
- Palette: the six custom properties above
- Type scale: the sizes above
- Design rules: flat, sharp, monochromatic, typographic
- Naming: Component--Thing PascalCase
- No inline styles, no inline scripts

## What This Does Not Decide

- Expression parsing or evaluation
- HTMX integration patterns
- Go template structure for generated visualizations
- CLI interface or Wikipedia page ingestion
- Which math topics to support
