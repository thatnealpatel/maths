# Design Language

A reference for reproducing the visual and structural idioms used in this project.

## CSS Authority

The global stylesheet (`static/css/style.css`) is the single source of truth for the design system. It defines all CSS custom properties, base element styles, and reusable component classes. Every rule below is specified in that file.

**Do not deviate from the stylesheet.** Specifically:

- Never redefine or override `:root` custom properties.
- Never restyle base elements (`h1`, `h2`, `p`, `hr`, `a`, `input`, `select`) — their appearance is set globally and must remain consistent across all pages.
- Never override the global component classes (`.Btn`, `.Btn--Primary`, `.Btn--Secondary`, `.Tag`, `.Controls--Row`, `.Controls--Group`, `.Controls--Slider`, `.Controls--Readout`, `.Controls--Tags`, `.Vis`, `.Vis--Label`, `.Vis--Canvas`, `.Section--Label`, `.Math--Inline`, `.Math--Block`, `.Page`, `.Proof`, `.Proof--Step`). Use them as-is.
- Page-specific `<style>` blocks in templates exist only to add **new** classes scoped to that page (e.g., `.DDPlots--Row`, `.DSR--Grid`). These classes compose the global components into page-specific layouts — they do not redefine appearance.
- If a page-specific class needs a value, reference a CSS custom property (e.g., `color: var(--ink-faint)`) rather than hardcoding a hex value. The exceptions are canvas JS (which cannot read CSS vars) and SVG `fill` attributes (inconsistent browser support for CSS vars) — in these cases, use the exact hex values from the palette table below.
- All `border-radius` is `0`. Do not introduce rounded corners anywhere.
- All `font-family` inherits from `--font-family` (IBM Plex Mono). Do not introduce additional typefaces.

If a new component is needed, define it as a new class following the existing naming convention (`PageName--ComponentName`), using only the existing custom properties for colors, and matching the sizing/spacing conventions documented below.

## Visual Identity

### Palette

All colors derive from six CSS custom properties on `:root`:

| Token | Hex | Usage |
|---|---|---|
| `--paper` | `#F5F2EB` | Page background, light text on dark fills |
| `--ink` | `#2C2B28` | Primary text, slider thumbs, active button fills |
| `--ink-light` | `#6B6963` | Body copy, secondary text |
| `--ink-faint` | `#A8A49C` | Labels, axis text, grid zero-lines, section labels |
| `--rule` | `#D6D2C9` | Borders, grid lines, `<hr>`, input borders |
| `--input-bg` | `#EDEAE2` | Input backgrounds, math blocks, disabled states |

Two accent colors are used directly (not as CSS vars) for data visualization:

| Name | Hex | Role |
|---|---|---|
| Terra cotta | `#C0522A` | Primary data series, "fail" states, warm emphasis |
| Teal | `#2E7D8C` | Secondary data series, "pass" states, cool emphasis |

Additional accent colors appear only in multi-series charts:

| Hex | Context |
|---|---|
| `#6B6963` | Third series (reuses `--ink-light`) |
| `#8B6914` | Fourth series (amber) |
| `#534AB7` | Fifth series (indigo) |

### Typography

Single typeface: **IBM Plex Mono** at weights 300/400/500/600. Loaded via Google Fonts. Everything is monospace — headings, body, labels, axes, tooltips.

| Element | Size | Weight | Color |
|---|---|---|---|
| `h1` | 1.6875rem | 500 | `--ink` |
| `h2` | 1.3125rem | 500 | `--ink` |
| `h3` | 1.0625rem | 500 | `--ink-light` |
| `p` | 1rem | 400 | `--ink-light` |
| Section label | 0.875rem | 500 | `--ink-faint`, uppercase, `letter-spacing: 0.15em` |
| Canvas axis text | 13px | — | `#A8A49C` (hardcoded; matches `--ink-faint`) |
| Canvas annotation | 11px | — | Context-dependent |
| Tooltip | 0.8125rem | — | `--paper` on `--ink` background |
| Tag | 0.875rem | — | `--ink-light`, `letter-spacing: 0.03em` |
| Button | 0.875rem | 500 | Uppercase, `letter-spacing: 0.15em` |

### Layout

- `.Page` container: `max-width: 850px`, centered with `margin: 3rem auto`, `padding: 0 1.5rem`.
- Horizontal rules (`<hr>`): `1px solid var(--rule)`, `margin: 2rem 0`. Used as section separators between controls, visualizations, and description blocks.
- Zero border-radius everywhere. Buttons, inputs, tags, slider thumbs — all sharp corners.
- Line height: `1.7` for body text, `1.5` for tooltips, `1.8` for math blocks.

### Links

Underlined with `text-decoration-color: var(--rule)` and `text-underline-offset: 0.2em`. On hover, underline color transitions to `--ink`.

## Component Vocabulary

### Section Labels (`.Section--Label`)

Uppercase, small, faint dividers that introduce groups of content. Always followed by the content they label (controls, visualizations, tables).

```html
<div class="Section--Label">Controls</div>
```

### Controls

**Sliders** (`.Controls--Slider`): Flex row with `<input type="range">` and a `.Controls--Readout` span. The readout updates via inline `oninput`. Range inputs are styled to 2px height track, 12px square thumb in `--ink`.

```html
<div class="Controls--Slider">
    <input type="range" name="shift" min="0" max="3" step="0.05" value="1.5"
           id="slider-a"
           oninput="document.getElementById('readout-a').textContent = parseFloat(this.value).toFixed(2)">
    <span class="Controls--Readout" id="readout-a">1.50</span>
</div>
```

**Groups** (`.Controls--Group`): Vertical stack of label + input. Labels are uppercase/faint. Groups sit inside a `.Controls--Row` (horizontal flex with gap and wrap).

**Select dropdowns**: Same font, background, border treatment as text inputs. No native appearance (`appearance: none`).

**Tags** (`.Tag`): Read-only parameter badges. Inline-block, 1px border in `--rule`, no border-radius.

```html
<div class="Controls--Tags">
    <span class="Tag">n = 200 per group</span>
    <span class="Tag">B = 500 permutations</span>
</div>
```

**Buttons** (`.Btn`): Uppercase, no border-radius. Two variants:
- `.Btn--Primary`: `--ink` border, inverts on hover (ink background, paper text).
- `.Btn--Secondary`: `--rule` border, `--ink-faint` text, inverts to ink on hover.

**Presets**: Smaller than buttons (0.8125rem, not uppercase), `1px solid var(--rule)`, no background. Invert to `--ink`/`--paper` on hover. Used for parameter presets that programmatically set slider values and trigger HTMX.

### Visualizations

**Container** (`.Vis`): `1px solid var(--rule)`, `padding: 1rem`, `position: relative`. Contains an optional `.Vis--Label` (same style as section labels) and a `.Vis--Canvas`.

**Canvas** (`.Vis--Canvas`): `display: block`, `width: 100%`, default `height: 300px`. Override height per-page via page-scoped `<style>` blocks, commonly with `aspect-ratio` for square/4:3 panels.

**Multi-panel layouts**: Flex rows (`.DDPlots--Row`, `.Spurious--Row`) or CSS grid (`.DSR--Grid`). Each panel is `flex: 1; min-width: 0`. Canvas aspect ratio is set per panel type.

### Tooltips

Fixed-position divs, hidden by default (`display: none`). Dark background (`--ink`), light text (`--paper`). Positioned at `clientX + 12, clientY + 12` on `mousemove`. Hidden on `mouseleave`. Never block pointer events (`pointer-events: none`).

```css
.Tooltip {
    position: fixed;
    background: var(--ink);
    color: var(--paper);
    font-family: var(--font-family);
    font-size: 0.8125rem;
    line-height: 1.5;
    padding: 0.4rem 0.6rem;
    pointer-events: none;
    display: none;
    z-index: 10;
}
```

### Legends

Horizontal flex rows below their visualization. Color swatches are `inline-block` spans (10-16px wide, 3-12px tall depending on series type). Clickable legend entries toggle visibility client-side and get `.disabled` class (`opacity: 0.3`).

### Tables

Full-width, `border-collapse: collapse`. Headers are uppercase/faint (`--ink-faint`). Cells right-aligned except the first column (left-aligned). Bottom border per row in `--rule`.

## Canvas Rendering Conventions

### DPR Handling

Every draw function begins with this preamble:

```javascript
var dpr = window.devicePixelRatio || 1;
var rect = canvas.getBoundingClientRect();
canvas.width = rect.width * dpr;
canvas.height = rect.height * dpr;
var ctx = canvas.getContext('2d');
ctx.scale(dpr, dpr);
var w = rect.width, h = rect.height;
```

All subsequent coordinates use CSS pixels (not physical pixels).

### Padding Object

Every chart defines a `pad` object with `t`, `r`, `b`, `l` fields (top, right, bottom, left in CSS px). Typical values:

| Chart type | Padding |
|---|---|
| Full-width line chart | `{t: 15, r: 15, b: 35, l: 55}` |
| Small panel scatterplot | `{t: 10, r: 10, b: 30, l: 40}` |
| Heatmap | `{t: 10, r: 15, b: 35, l: 40}` |
| Bar chart | `{t: 15, r: 15, b: 40, l: 55}` |

Plot width and height: `pw = w - pad.l - pad.r`, `ph = h - pad.t - pad.b`.

### Coordinate Mapping

Mapping functions `mx(x)` (data → canvas x) and `my(y)` (data → canvas y, inverted):

```javascript
function mx(x) { return pad.l + (x - xmin) / (xmax - xmin) * pw; }
function my(y) { return pad.t + (ymax - y) / (ymax - ymin) * ph; }
```

### Grid and Axes

1. **Grid lines**: `strokeStyle = '#D6D2C9'` (matches `--rule`), `lineWidth = 1`. Drawn first, behind data.
2. **Zero lines / reference lines**: `strokeStyle = '#A8A49C'` (matches `--ink-faint`), `lineWidth = 1`. For dashed references (e.g., α=0.05 threshold, diagonal), use `setLineDash([4, 4])`.
3. **Tick labels**: `fillStyle = '#A8A49C'`, `font = '13px IBM Plex Mono, monospace'`. X-axis labels centered below plot area, Y-axis labels right-aligned to the left of plot area.
4. **Axis titles**: Same font/color, positioned at `pad.t + ph + 20` (x-axis) or rotated 90° at left edge (y-axis).

### Nice Tick Step

Shared across charts. Targets ~6 ticks:

```javascript
function niceStep(span) {
    var rough = span / 6;
    var mag = Math.pow(10, Math.floor(Math.log10(rough)));
    var norm = rough / mag;
    if (norm < 1.5) return mag;
    if (norm < 3.5) return 2 * mag;
    if (norm < 7.5) return 5 * mag;
    return 10 * mag;
}
```

### Number Formatting

```javascript
function fmtNum(n) {
    if (Math.abs(n) < 1e-10) return '0';
    if (Math.abs(n) >= 1000 || (Math.abs(n) < 0.01 && n !== 0)) return n.toExponential(1);
    return parseFloat(n.toPrecision(4)).toString();
}
```

### Data Series Colors

- Primary series: `#C0522A` (terra cotta), `lineWidth = 2`.
- Secondary series: `#2E7D8C` (teal), `lineWidth = 2`.
- Reference/comparison lines: `#A8A49C`, `lineWidth = 1.5`, often dashed.
- Current-value indicator: `#3D3A35`, `lineWidth = 1`, dashed `[4, 4]`.
- Scatterplot points: filled circles, `radius = 2.5`, `globalAlpha = 0.5`.

### Color Interpolation (Heatmaps)

Interpolate from `--paper` (#F5F2EB) to the accent color based on the data value `t ∈ [0, 1]`:

```javascript
// Paper → Terra cotta
var r = Math.round(245 + (192 - 245) * t);
var g = Math.round(242 + (82 - 242) * t);
var b = Math.round(235 + (42 - 235) * t);
```

For diverging scales, use paper → teal for positive and paper → terra cotta for negative. Text inside cells flips between `--paper` and `--ink` based on intensity (`t > 0.5`).

### Shaded Regions

Fill with the accent color at 10% opacity: `rgba(46, 125, 140, 0.1)` (teal) or `rgba(192, 82, 42, 0.1)` (terra cotta). Used to shade pass/fail regions or areas between curves and thresholds.

## Architecture

### Directory Structure

```
cmd/site/
├── main.go              # template parsing, route registration, server lifecycle
├── handlers/            # page endpoints — serve full HTML pages
│   ├── fourier.go       # GET /fourier-transform
│   ├── robustness.go    # GET /robustness/*
│   └── dsr.go           # GET /dsr
├── api/                 # data endpoints — return JSON fragments, handle updates
│   ├── fourier.go       # POST /fourier-transform/compute
│   ├── robustness.go    # GET /robustness/*/data
│   └── dsr.go           # GET /dsr/data
├── templates/           # Go html/template files (.tmpl)
│   ├── layout.tmpl
│   ├── index.tmpl
│   └── ...
└── static/
    └── css/
        └── style.css
```

**`handlers/`** contains page-level route handlers. Each handler renders a full HTML page by executing a template. These are the routes a user navigates to in the browser.

**`api/`** contains data route handlers. These return JSON (or HTML fragments, for HTMX partial swaps) consumed by client-side JS or HTMX. All `hx-get` / `hx-post` targets point here. Precomputed state, parameter parsing, and compute-on-the-fly logic live in this package.

This separation keeps the page-rendering concern (template selection, title, layout) cleanly apart from the data-serving concern (parsing, validation, computation, JSON encoding). Both packages register their routes on the same `*http.ServeMux` in `main.go`.

### Template Structure

Layout is defined in `templates/layout.tmpl`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{.Title}} — mathvis</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
</head>
<body>
    <main class="Page">
        {{block "content" .}}{{end}}
    </main>
    {{block "scripts" .}}{{end}}
</body>
</html>
```

Each page template defines two blocks:
- `{{define "content"}}`: Page-specific `<style>`, HTML structure, controls, visualization containers.
- `{{define "scripts"}}`: A single `<script>` tag with all page JS (handler functions, draw functions, event listeners).

Page-specific CSS goes in a `<style>` tag at the top of the content block, not in the global stylesheet. Class names are BEM-ish, namespaced to the page: `.DDPlots--Row`, `.Conc--Legend`, `.C2ST--Tooltip`, `.DSR--Grid`, `.EMMD--Table`.

Templates are cloned from the layout at startup:

```go
layoutTmpl := template.Must(template.ParseFS(templateFS, "templates/layout.tmpl"))
pageTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/page.tmpl"))
```

### HTMX Data Transport

The pattern for all interactive visualizations:

1. **No DOM swap**: `hx-swap="none"`. The server returns JSON, not HTML.
2. **JS handler on response**: `hx-on::after-request="handlePage(event)"`. The handler parses JSON and calls canvas draw functions.
3. **Include all params**: Each slider uses `hx-include` to list all other slider IDs, so the server always receives the full parameter set.
4. **Abort stale requests**: `hx-sync="this:replace"` on each control. If the user moves a slider faster than the server responds, in-flight requests are cancelled.
5. **Initial load**: One control (usually the first slider) has `hx-trigger="load, input"`. All others have `hx-trigger="input"` (or `"change"` for selects/numbers). This fires exactly one initial request on page load.

```html
<input type="range" name="shift" min="0" max="3" step="0.05" value="1.5"
       id="slider-a"
       hx-get="/page/data"
       hx-include="#slider-b, #slider-c"
       hx-swap="none"
       hx-sync="this:replace"
       hx-trigger="load, input"
       hx-on::after-request="handlePage(event)">
```

The JS handler always checks for success before parsing:

```javascript
function handlePage(event) {
    if (!event.detail.successful) return;
    var data = JSON.parse(event.detail.xhr.responseText);
    drawChart(data);
}
```

### Server-Side Handler Pattern

Each handler file registers its routes via a public function:

```go
func RegisterFoo(mux *http.ServeMux) {
    mux.HandleFunc("GET /foo/data", handleFooData)
}
```

Or, for handlers with precomputed state, via a struct:

```go
type Foo struct {
    precomputed fooResult
}

func NewFoo(ctx context.Context) *Foo { ... }

func (h *Foo) Register(mux *http.ServeMux) {
    mux.HandleFunc("GET /foo/data", h.handleFooData)
}
```

Data handlers:
1. Parse query params with `strconv.ParseFloat` / `strconv.Atoi`.
2. Validate ranges. Return `http.StatusBadRequest` with a short message on invalid input.
3. Check for default values to serve precomputed results (avoids redundant computation on initial load).
4. Compute on-the-fly for non-default values.
5. Set `Content-Type: application/json` and encode with `json.NewEncoder(w).Encode(result)`.

Page routes are registered in `main.go` as closures:

```go
mux.HandleFunc("GET /page", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := pageTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "Page Title"}); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
})
```

### Computation Models

Two patterns:

**Per-request** (Fourier, DD-plots, concentration, spurious correlation, DSR): Computation is fast (< 100ms). Handler computes on every request. Default values may be precomputed and cached for instant initial load.

**Precomputed** (C2ST, energy-vs-mmd): Computation takes seconds to minutes. Results are computed once at startup and stored in the handler struct. The data endpoint serves the cached result directly. There are no sliders — the page loads a static visualization.

For precomputed visualizations, the server blocks during startup while computing. Seeded RNGs (`rand.New(rand.NewSource(N))`) ensure deterministic results across restarts.

### Math in HTML

Mathematical symbols are rendered as HTML entities or Unicode code points, not LaTeX:

```html
&#x03C1;   <!-- ρ -->
&#x03B1;   <!-- α -->
&#x03BC;   <!-- μ -->
&#x03C3;   <!-- σ -->
&#x03B3;   <!-- γ -->
&#x03BE;   <!-- ξ -->
&#x2081;   <!-- ₁ (subscript 1) -->
&#x0394;   <!-- Δ -->
&#x221A;   <!-- √ -->
&#x222B;   <!-- ∫ -->
&#x00D7;   <!-- × -->
&#x00B1;   <!-- ± -->
SR&#x0302; <!-- SR̂ (combining circumflex) -->
p&#x0304;  <!-- p̄ (combining macron) -->
```

Inline math uses `.Math--Inline` (monospace, light background, slight padding). Block math uses `.Math--Block` (background, left border in `--ink`, generous padding).

## Page Anatomy

Every visualization page follows this vertical structure:

```
h1          — Page title
p           — 2-3 sentence description of the visualization
[Tags]      — Optional parameter badges for precomputed visualizations
<hr>
Section--Label "Controls"
Controls--Row  — Sliders, selects, number inputs
[Presets]      — Optional preset buttons
<hr>
Section--Label — Describes what the visualization shows
Vis            — Canvas(es) in Vis containers
[Legend]        — Optional legend below the visualization
[Table]         — Optional summary table
[Tooltip]       — Hidden tooltip div, positioned via JS
```
