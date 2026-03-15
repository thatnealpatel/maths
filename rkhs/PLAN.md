# PLAN.md — RKHS Explorer: Local Web Server Refactor

## Objective

Refactor the RKHS explorer from a CLI tool that writes individual SVG files to a self-contained Go program that generates all plots in memory and serves them on a local HTTP site with single-page scrolling navigation. The user runs one command and gets a URL they can open in their browser.

## Current State

The program in `main.go` generates five standalone SVG files to disk using `go-gg`, then prints `open` commands the user must copy-paste. This is a poor developer experience.

## Target State

Running `go run .` (or `go run . -p skewed -q bimodal -sigma 0.3 -n 50`) should:

1. Generate all five SVG plots in memory (no files written to disk unless `-export` is passed).
2. Start an HTTP server on a free local port (default `localhost:8741`, configurable via `-port`).
3. Print a single line: `Serving RKHS Explorer at http://localhost:8741` and open the browser automatically (best-effort via `xdg-open` / `open` / `start`).
4. Serve a single HTML page at `/` that displays all five plots in a vertically scrolling layout with anchor navigation.
5. Shut down cleanly on `Ctrl+C`.

## Architecture

### File Structure

```
rkhs-gg/
├── main.go          # CLI flags, sampling, server startup, browser open
├── rkhs.go          # Kernel, MMD, mean embedding, distribution definitions (extract from current main.go)
├── plots.go         # Five plot-builder functions, each returning []byte (SVG) instead of writing to disk
├── server.go        # HTTP handler: serves index page and individual SVG endpoints
├── index.html       # Embedded via go:embed — the single-page HTML shell
├── go.mod
├── go.sum
├── README.md
└── PLAN.md
```

### Key Design Decisions

**Embed the HTML template.** Use `//go:embed index.html` so the binary is fully self-contained. No external asset dependencies.

**SVGs served at individual endpoints.** The index page references each plot via `<img src="/plot/01_kernel_functions.svg">`. The server generates them on first request and caches in memory. This keeps the HTML clean and allows the user to also open any individual SVG directly.

**Regeneration endpoint.** Expose `POST /regenerate` which resamples from the distributions (new random draw) and invalidates the cache. The page includes a "Resample" button that hits this endpoint and reloads.

**Query parameter overrides.** The HTML page should include a small control panel at the top that lets the user adjust `-p`, `-q`, `-sigma`, `-n` via form inputs and hit "Update". This submits as query parameters to `/` which re-renders the page with new parameters. Server-side rendering — no JavaScript framework needed.

## Implementation Steps

Execute these in order. Each step should compile and run before moving to the next.

### Step 1: Extract `rkhs.go`

Move all non-main, non-plotting code from `main.go` into `rkhs.go`:

- Distribution type and instances (Gaussian, Bimodal, Uniform, Skewed, the `distributions` map).
- `RBF`, `MeanEmbedding`, `MMDSquared`, `linspace` functions.

Verify: `go build .` succeeds with no changes to behavior.

### Step 2: Refactor plot functions into `plots.go`

Modify the five plot-builder functions so that instead of taking a file path and writing to disk, they each return `([]byte, error)` — the SVG content as bytes. Internally, write to a `bytes.Buffer` instead of `os.Create`.

Function signatures become:

```go
func plotKernelFunctions(dist Distribution, samples []float64, sigma float64, grid []float64) ([]byte, error)
func plotMeanEmbedding(dist Distribution, samples []float64, sigma float64, grid []float64) ([]byte, error)
func plotMMD(distP, distQ Distribution, samplesP, samplesQ []float64, sigma float64, grid []float64) ([]byte, error)
func plotGramHeatmap(samples []float64, sigma float64) ([]byte, error)
func plotSigmaSweep(distP, distQ Distribution, samplesP, samplesQ []float64) ([]byte, error)
```

Verify: temporarily call these from `main()` and write the returned bytes to files to confirm output is identical.

### Step 3: Create `index.html`

A single HTML file with:

- A fixed-position top nav bar with anchor links: "Kernel Functions", "Mean Embedding", "MMD", "Gram Matrix", "σ Sweep".
- A control panel (just below nav) with: two `<select>` dropdowns for P and Q distributions, a number input for σ, a number input for n, and an "Update" button that reloads with query params.
- Five sections, each containing: a heading, a one-paragraph explanation of what the plot shows, and an `<img>` tag loading from `/plot/{name}.svg`.
- A "Resample" button that POSTs to `/regenerate` then reloads.
- Clean, minimal CSS. No JavaScript frameworks. System fonts. Light/dark mode via `prefers-color-scheme`. Max-width 960px centered.

Template variables (Go `html/template`):
- `{{.DistP}}`, `{{.DistQ}}`, `{{.Sigma}}`, `{{.N}}`, `{{.Seed}}`
- `{{.MMD2}}`, `{{.MMD}}` — precomputed numerical results shown inline.

### Step 4: Create `server.go`

Implement the HTTP server:

```go
type Server struct {
    mu       sync.Mutex
    params   Params        // current P, Q, sigma, n, seed
    samples  SampleCache   // samplesP, samplesQ []float64
    plots    map[string][]byte  // cached SVG bytes keyed by plot name
    grid     []float64
    tmpl     *template.Template
}
```

Routes:

| Method | Path                          | Handler                                                      |
|--------|-------------------------------|--------------------------------------------------------------|
| GET    | `/`                           | Render `index.html` template with current params and MMD stats |
| GET    | `/plot/{name}.svg`            | Return cached SVG bytes (generate on first access), `Content-Type: image/svg+xml` |
| POST   | `/regenerate`                 | Resample, clear cache, redirect to `/`                       |
| GET    | `/update?p=...&q=...&sigma=...&n=...` | Parse params, resample, clear cache, redirect to `/`  |

Use `net/http` from the standard library. No third-party router needed — `http.HandleFunc` with path matching is sufficient.

Cache invalidation: any change to params or resample clears `s.plots` so the next GET to `/plot/` regenerates.

### Step 5: Rewrite `main.go`

New `main()`:

1. Parse flags: `-p`, `-q`, `-sigma`, `-n`, `-seed`, `-port` (default 8741), `-export` (bool, writes SVGs to disk instead of serving).
2. If `-export`: generate and write files as before (backward compat), then exit.
3. Otherwise: construct `Server`, start `http.ListenAndServe` in a goroutine, print the URL, attempt browser open, block on signal.

Browser open (best-effort, don't fail if it doesn't work):

```go
func openBrowser(url string) {
    switch runtime.GOOS {
    case "darwin":
        exec.Command("open", url).Start()
    case "linux":
        exec.Command("xdg-open", url).Start()
    case "windows":
        exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    }
}
```

### Step 6: Update `README.md`

Replace the current "open SVG files" instructions with:

```bash
cd rkhs-gg
go mod tidy
go run .
# → Serving RKHS Explorer at http://localhost:8741
```

Document:
- The browser-based parameter controls.
- The `-export` flag for offline SVG generation.
- The `-port` flag.
- That `Ctrl+C` shuts down the server.

### Step 7: Verify and clean up

- `go vet ./...` passes.
- `go build .` produces a single binary.
- Run with no flags: browser opens, page loads, all five plots render, nav links scroll correctly, "Update" changes params, "Resample" draws fresh samples.
- Run with `-export`: SVG files written to disk as before.
- Run with `-p skewed -q uniform -sigma 0.3 -n 50`: verify params propagate to plots and numerical summary.

## Constraints

- **Zero JavaScript dependencies.** The page is server-rendered HTML with `<img>` tags. The only JS is the Resample button handler (a one-liner `fetch` + `location.reload`).
- **No external CSS frameworks.** Style inline or in a `<style>` block within `index.html`.
- **No file I/O in server mode.** All SVGs generated in memory.
- **Single binary.** The `go:embed` directive ensures `index.html` is compiled into the binary.
- **Backward compatible.** The `-export` flag preserves the original file-writing behavior.

## Out of Scope

- WebSocket-based live updates (unnecessary complexity for a local tool).
- Client-side interactivity like sliders (the server round-trip is fast enough for parameter changes).
- Docker or container packaging.
- Tests (unless the implementer wants to add them — encouraged but not blocking).
