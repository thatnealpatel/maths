package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"sync"
)

type Params struct {
	DistP string
	DistQ string
	Sigma float64
	N     int
	Seed  int64
}

type Server struct {
	mu       sync.Mutex
	params   Params
	samplesP []float64
	samplesQ []float64
	plots    map[string][]byte
	grid     []float64
	tmpl     *template.Template
}

func NewServer(params Params, tmpl *template.Template) *Server {
	s := &Server{
		params: params,
		plots:  make(map[string][]byte),
		grid:   linspace(-4.5, 5.5, 400),
		tmpl:   tmpl,
	}
	s.resample()
	return s
}

func (s *Server) resample() {
	rng := rand.New(rand.NewSource(s.params.Seed))
	s.samplesP = make([]float64, s.params.N)
	s.samplesQ = make([]float64, s.params.N)
	P := distributions[s.params.DistP]
	Q := distributions[s.params.DistQ]
	for i := 0; i < s.params.N; i++ {
		s.samplesP[i] = P.Sample(rng)
		s.samplesQ[i] = Q.Sample(rng)
	}
	s.plots = make(map[string][]byte)
}

var plotNames = []string{
	"01_kernel_functions",
	"02_mean_embedding",
	"03_mmd_comparison",
	"04_gram_matrix",
	"05_sigma_sweep",
}

func (s *Server) generatePlot(name string) ([]byte, error) {
	P := distributions[s.params.DistP]
	Q := distributions[s.params.DistQ]

	switch name {
	case "01_kernel_functions":
		return plotKernelFunctions(P, s.samplesP, s.params.Sigma, s.grid)
	case "02_mean_embedding":
		return plotMeanEmbedding(P, s.samplesP, s.params.Sigma, s.grid)
	case "03_mmd_comparison":
		return plotMMD(P, Q, s.samplesP, s.samplesQ, s.params.Sigma, s.grid)
	case "04_gram_matrix":
		return plotGramHeatmap(s.samplesP, s.params.Sigma)
	case "05_sigma_sweep":
		return plotSigmaSweep(P, Q, s.samplesP, s.samplesQ)
	default:
		return nil, fmt.Errorf("unknown plot: %s", name)
	}
}

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	mmd2 := MMDSquared(s.samplesP, s.samplesQ, s.params.Sigma)
	mmd := math.Sqrt(math.Max(0, mmd2))

	names := make([]string, 0, len(distributions))
	for k := range distributions {
		names = append(names, k)
	}
	sort.Strings(names)

	data := struct {
		DistP     string
		DistQ     string
		Sigma     float64
		N         int
		Seed      int64
		MMD2      float64
		MMD       float64
		DistNames []string
	}{
		DistP:     s.params.DistP,
		DistQ:     s.params.DistQ,
		Sigma:     s.params.Sigma,
		N:         s.params.N,
		Seed:      s.params.Seed,
		MMD2:      mmd2,
		MMD:       mmd,
		DistNames: names,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.tmpl.Execute(w, data)
}

func (s *Server) HandlePlot(w http.ResponseWriter, r *http.Request) {
	// Path is /plot/{name}.svg — extract name.
	path := r.URL.Path
	// Strip "/plot/" prefix and ".svg" suffix.
	if len(path) <= len("/plot/")+len(".svg") {
		http.NotFound(w, r)
		return
	}
	name := path[len("/plot/") : len(path)-len(".svg")]

	s.mu.Lock()
	defer s.mu.Unlock()

	svg, ok := s.plots[name]
	if !ok {
		var err error
		svg, err = s.generatePlot(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.plots[name] = svg
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(svg)
}

func (s *Server) statsJSON() []byte {
	mmd2 := MMDSquared(s.samplesP, s.samplesQ, s.params.Sigma)
	mmd := math.Sqrt(math.Max(0, mmd2))
	data := map[string]any{
		"distP": s.params.DistP,
		"distQ": s.params.DistQ,
		"sigma": s.params.Sigma,
		"n":     s.params.N,
		"seed":  s.params.Seed,
		"mmd2":  mmd2,
		"mmd":   mmd,
	}
	b, _ := json.Marshal(data)
	return b
}

func (s *Server) HandleRegenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	s.params.Seed = rand.Int63()
	s.resample()
	b := s.statsJSON()
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *Server) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	s.mu.Lock()
	if v := q.Get("p"); v != "" {
		if _, ok := distributions[v]; ok {
			s.params.DistP = v
		}
	}
	if v := q.Get("q"); v != "" {
		if _, ok := distributions[v]; ok {
			s.params.DistQ = v
		}
	}
	if v := q.Get("sigma"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			s.params.Sigma = f
		}
	}
	if v := q.Get("n"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 30 {
			s.params.N = n
		}
	}
	s.resample()
	b := s.statsJSON()
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
