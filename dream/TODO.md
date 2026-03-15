# TODO: Loading Page for Precomputing Visualizations

## Context

C2ST precomputation takes ~30-60s. Currently `NewRobustness()` blocks server start — no pages served until everything finishes. We want the server to start immediately and show a themed loading page when a visualization's data isn't ready yet.

**Chosen approach:** Dedicated loading page template. Server starts immediately, background precomputation, page routes serve `loading.tmpl` until ready.

## Implementation

### Step 1: Add readiness tracking to `Robustness` struct

**File:** `cmd/site/handlers/robustness.go`

Add a `sync.RWMutex` and per-visualization readiness flags to the struct:

```go
type Robustness struct {
    mu          sync.RWMutex
    ddDefaults  [3]ddPanelResult
    ddReady     bool
    concDefault concResult
    concReady   bool
    c2stDefault c2stGridResult
    c2stReady   bool
}
```

Add a public method:
```go
func (h *Robustness) Ready(name string) bool {
    h.mu.RLock()
    defer h.mu.RUnlock()
    switch name {
    case "dd-plots":    return h.ddReady
    case "concentration": return h.concReady
    case "c2st":        return h.c2stReady
    default:            return false
    }
}
```

### Step 2: Make `NewRobustness()` non-blocking

**File:** `cmd/site/handlers/robustness.go`

`NewRobustness()` returns immediately. Precomputation runs in a background goroutine, setting flags as each visualization completes:

```go
func NewRobustness() *Robustness {
    h := &Robustness{}
    go h.precompute()
    return h
}

func (h *Robustness) precompute() {
    log.Println("  dd-plots: computing default state...")
    dd0 := ddPanelResult{Points: ddCompute("a", 1.5, ...), Panel: "a"}
    dd1 := ddPanelResult{Points: ddCompute("b", 2.0, ...), Panel: "b"}
    dd2 := ddPanelResult{Points: ddCompute("c", 0.8, ...), Panel: "c"}
    h.mu.Lock()
    h.ddDefaults = [3]ddPanelResult{dd0, dd1, dd2}
    h.ddReady = true
    h.mu.Unlock()
    log.Println("  dd-plots: done")

    // same pattern for concentration, then c2st
    // ...
}
```

Data handlers acquire a read lock and check readiness. If not ready, return **HTTP 503**.

### Step 3: Create `templates/loading.tmpl`

**File:** `cmd/site/templates/loading.tmpl` (new)

Extends `layout.tmpl`. Content block: centered vertically and horizontally, shows:
- The page title (passed as template data)
- The pulsing three-dot SVG
- A small "computing..." message in `--ink-faint`
- `<meta http-equiv="refresh" content="2">` for auto-reload every 2 seconds

SVG (inline, themed):
```svg
<svg width="48" height="12" viewBox="0 0 48 12">
  <circle cx="6" cy="6" r="4" fill="#A8A49C">
    <animate attributeName="opacity" values="1;0.3;1" dur="1.2s" repeatCount="indefinite" begin="0s"/>
  </circle>
  <circle cx="24" cy="6" r="4" fill="#A8A49C">
    <animate attributeName="opacity" values="1;0.3;1" dur="1.2s" repeatCount="indefinite" begin="0.4s"/>
  </circle>
  <circle cx="42" cy="6" r="4" fill="#A8A49C">
    <animate attributeName="opacity" values="1;0.3;1" dur="1.2s" repeatCount="indefinite" begin="0.8s"/>
  </circle>
</svg>
```

Page-specific `<style>` in the template for centering (`.Loading--Container` with flexbox center, min-height). Uses `#A8A49C` (the `--ink-faint` value) since SVG `fill` can't reliably read CSS vars in all browsers.

### Step 4: Wire up page routes with readiness check

**File:** `cmd/site/main.go`

Parse `loading.tmpl` alongside other templates. Each robustness page route checks readiness before deciding which template to serve:

```go
loadingTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/loading.tmpl"))

mux.HandleFunc("GET /robustness/c2st", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if !rob.Ready("c2st") {
        loadingTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{
            "Title": "C2ST Power Surface",
        })
        return
    }
    c2stTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "C2ST Power Surface"})
})
```

Same pattern for `/robustness/dd-plots` and `/robustness/concentration`.

Move server start before precomputation — `NewRobustness()` now returns instantly.

### Step 5: Guard data endpoints

**File:** `cmd/site/handlers/robustness.go`

Each data handler checks readiness under read lock. Example for C2ST:

```go
func (h *Robustness) handleC2STData(w http.ResponseWriter, r *http.Request) {
    h.mu.RLock()
    ready := h.c2stReady
    result := h.c2stDefault
    h.mu.RUnlock()
    if !ready {
        http.Error(w, "computing", http.StatusServiceUnavailable)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

## Files Changed

| File | Change |
|------|--------|
| `cmd/site/handlers/robustness.go` | Add mutex + readiness flags, non-blocking `NewRobustness()`, `Ready()` method, guard data handlers |
| `cmd/site/templates/loading.tmpl` | New template with SVG animation + auto-refresh |
| `cmd/site/main.go` | Parse loading template, readiness checks in page routes, reorder startup |

## Verification

1. `go build -o /dev/null ./cmd/site/`
2. `go run ./cmd/site/` — server starts immediately (listen message before precomputation logs)
3. Visit `/robustness/c2st` before precomputation finishes → loading page with pulsing dots
4. Page auto-refreshes every 2s; once C2ST finishes, refresh shows real heatmap
5. Visit `/robustness/dd-plots` — ready almost immediately
6. Existing Fourier transform page unaffected
