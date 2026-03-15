# PLAN.md — mathvis `cmd/site`

## Overview

A single Go binary that serves interactive mathematical visualizations as HTML pages. Each visualization is a bespoke Go HTML template built from a specific Wikipedia math article. Templates accumulate over time in a shared directory. There is no generic component API, no JSON schema, no runtime LLM — just a growing collection of hand-edited templates served by route.

A future refactor may extract a JSON API from patterns observed across accumulated templates. That refactor is explicitly **not in scope** here.

## Architecture

### Project Layout

This lives inside an existing Go module that already contains `cmd/design/` (the stylesheet sampler). The shared stylesheet is the canonical source of truth from `cmd/design/static/css/style.css`.

```
cmd/site/
├── main.go                        # HTTP server, route registration, shared helpers
├── handlers/                      # per-topic handler files (Go compute endpoints)
│   └── fourier.go                 # example: handlers for Fourier transform HTMX endpoints
├── templates/
│   ├── layout.tmpl                # base layout: html head, stylesheet link, nav shell
│   ├── index.tmpl                 # landing page: list of available visualizations
│   └── fourier-transform.tmpl     # example: bespoke template for Fourier transform
└── static/
    ├── css/
    │   └── style.css -> ../../../design/static/css/style.css   # symlink to canonical stylesheet
    └── js/
        └── htmx.min.js           # vendored HTMX (no CDN dependency at runtime)
```

### Key Decisions

**Symlinked stylesheet.** `cmd/site/static/css/style.css` is a symlink to `cmd/design/static/css/style.css`. One source of truth. Changes to the design sampler propagate to the site automatically. If symlinks cause problems on your OS or with `go:embed`, copy the file and document the source — but prefer the symlink.

**Vendored HTMX.** Download `htmx.min.js` into `static/js/`. No CDN, no NPM, no external runtime dependencies. The binary + its embedded assets are fully self-contained.

**`go:embed` for all static assets and templates.** The binary embeds everything. `go run ./cmd/site` and open a browser — nothing else required.

**Route-based serving.** Each template is served at a route matching its filename: `fourier-transform.tmpl` → `GET /fourier-transform`. No dynamic dispatch, no registry pattern. `main.go` registers routes explicitly. When you add a template, you add a route. This is intentionally manual.

**Per-topic handler files.** Each visualization that needs server-side computation gets a file in `handlers/`. The Fourier transform page might need endpoints like `POST /fourier-transform/evaluate` that accept expression + parameters and return computed data as an HTMX partial or JSON for canvas rendering. Handlers are registered in `main.go` alongside the page route.

**No client-side expression parsing.** All math evaluation happens in Go. The client sends expressions as strings; Go parses and evaluates them. This is safer and keeps JS minimal. Use a library or hand-roll as needed per topic — no requirement for a universal evaluator yet.

## Base Layout Template

`templates/layout.tmpl` defines the HTML shell:

```
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} — mathvis</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@300;400;500;600&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="/static/js/htmx.min.js"></script>
</head>
<body>
    <main class="Page">
        {{template "content" .}}
    </main>
    {{template "scripts" .}}
</body>
</html>
```

Each topic template defines the `"content"` and `"scripts"` blocks. The `"scripts"` block is for page-specific JS (canvas rendering, animation). It goes at the bottom of the body.

## Index Page

`templates/index.tmpl` lists available visualizations. This is manually maintained — a simple list of links. No auto-discovery. When you add a template, you add a link to the index.

## How a New Topic Gets Added

This is the growth loop. It's manual and intentional.

1. **Pick a Wikipedia math article.** Read it. Understand what the useful interactive visualization would be for someone with aphantasia — what needs to be manipulable, not just rendered.

2. **Write a topic-specific PLAN.** Hand this to Claude Code. It should specify:
   - Which concepts from the article to visualize
   - What the interactive controls are (sliders, inputs, toggles)
   - What computation happens server-side (Go handler endpoints)
   - What computation happens client-side (canvas drawing, animation)
   - The HTMX interaction pattern (what triggers what, what gets swapped)

3. **Claude Code generates:**
   - `templates/<topic-name>.tmpl` — the bespoke template using the shared stylesheet classes
   - `handlers/<topic>.go` — the Go compute endpoints
   - Any additions to `main.go` for route registration

4. **You edit for idioms and style.** The generated Go gets reviewed and adjusted. The template gets reviewed for design language compliance with DESIGN.md.

5. **Add to index.** Add a link in `index.tmpl`.

6. **Run and verify.** `go run ./cmd/site`, open browser, test.

## HTMX Patterns

Keep these simple and consistent across topics:

**Compute-and-swap:** A control (slider, input) triggers an HTMX request to a Go handler. The handler computes and returns an HTML fragment that replaces a target div.

```html
<input type="range" min="0" max="100" value="50"
       hx-get="/fourier-transform/evaluate?xi={{.}}"
       hx-target="#result"
       hx-trigger="input changed delay:100ms">
<div id="result">
    <!-- swapped by HTMX -->
</div>
```

**Canvas data:** For canvas-based plots where you can't swap HTML, the HTMX endpoint returns JSON. A small inline script reads the response and redraws. This is the exception to "minimal JS" — canvas requires it.

```html
<div hx-get="/fourier-transform/plot-data?xi=3.5"
     hx-trigger="input changed from:#freq-slider delay:100ms"
     hx-swap="none"
     hx-on::after-request="drawPlot(event.detail.xhr.response)">
</div>
```

**No nested HTMX.** Keep it flat. One trigger, one target, one swap. If the interaction feels like it needs nested or chained swaps, rethink the page structure.

**Two interaction modes, two debugging stories.** These patterns look similar but have different failure modes. Treat them as distinct when something breaks:

- **Fragment swap** (compute-and-swap): HTMX transports the response *and* renders it. The server sends a representation the browser displays without interpretation. When it breaks, debug the Go template — the HTML it returns is wrong, or the swap target is mismatched. JS is not involved.
- **Data transport** (canvas data): HTMX transports the response, but JS renders it. The server sends data that requires interpretation — coordinate mapping, line drawing, axis labeling. When it breaks, the bug is either in the Go handler (wrong data) or the JS rendering function (wrong drawing). The failure is in application code on one end or the other, not in the HTMX layer.

Do not treat these as interchangeable. The transport pattern is identical; the rendering and debugging are not.

## CSS Classes Available

From the canonical stylesheet (see DESIGN.md for full reference):

```
.Page                    /* max-width container */
.Section--Label          /* uppercase divider labels */
.Math--Inline            /* inline math spans */
.Math--Block             /* display math with left border */
.Btn, .Btn--Primary, .Btn--Secondary
.Tag                     /* small bordered labels */
.Controls--Row, .Controls--Group, .Controls--Expression,
.Controls--Number, .Controls--Slider, .Controls--Readout,
.Controls--Actions, .Controls--Tags
.Vis, .Vis--Label, .Vis--Canvas
.Proof, .Proof--Step
```

All templates must use these classes. No inline styles. Page-specific styles go in a `<style>` block inside the template if absolutely necessary, but prefer extending the shared stylesheet.

## Expression Evaluation Strategy

There is no universal expression evaluator. Each topic handler evaluates what it needs.

For the first topic (Fourier transform), this means:
- Accept a mathematical expression as a string
- Parse and evaluate it over a range of x values
- Return the computed y values

Use an existing Go library if one fits cleanly (e.g., `github.com/Knetic/govaluate` or similar). If the library doesn't fit, write a minimal evaluator scoped to the topic's needs. Do not build a general-purpose expression parser unless multiple topics demand it.

## Non-Goals

- No runtime LLM integration
- No generic JSON API for component composition
- No voice input
- No auto-discovery of templates
- No build system beyond `go run` / `go build`
- No external dependencies at runtime (all assets embedded)
- No React, no NPM, no Node
- No attempt to support arbitrary Wikipedia pages — one topic at a time, manually

## Success Criteria

After following this plan, `go run ./cmd/site` should:

1. Serve the index page at `/` listing available visualizations
2. Serve each topic at its named route (e.g., `/fourier-transform`)
3. Apply the shared stylesheet from `cmd/design/` consistently
4. Handle HTMX interactions for server-side compute without page reloads
5. Render interactive canvas visualizations with minimal client-side JS
6. Feel like a typed mathematics manuscript, per DESIGN.md
