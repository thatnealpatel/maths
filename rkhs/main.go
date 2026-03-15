package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
)

//go:embed index.html
var indexHTML string

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

func main() {
	distP := flag.String("p", "bimodal", "Distribution P: gaussian, bimodal, uniform, skewed")
	distQ := flag.String("q", "gaussian", "Distribution Q: gaussian, bimodal, uniform, skewed")
	sigma := flag.Float64("sigma", 0.5, "RBF kernel bandwidth")
	nSamp := flag.Int("n", 12, "Number of samples per distribution")
	seed := flag.Int64("seed", 42, "Random seed (0 for non-deterministic)")
	port := flag.Int("port", 8741, "HTTP server port")
	export := flag.Bool("export", false, "Export SVGs to disk instead of serving")
	outDir := flag.String("out", ".", "Output directory for SVG files (with -export)")
	flag.Parse()

	P, okP := distributions[*distP]
	Q, okQ := distributions[*distQ]
	if !okP {
		fmt.Fprintf(os.Stderr, "Unknown distribution P: %s (options: gaussian, bimodal, uniform, skewed)\n", *distP)
		os.Exit(1)
	}
	if !okQ {
		fmt.Fprintf(os.Stderr, "Unknown distribution Q: %s (options: gaussian, bimodal, uniform, skewed)\n", *distQ)
		os.Exit(1)
	}

	if *export {
		exportSVGs(P, Q, *sigma, *nSamp, *seed, *outDir)
		return
	}

	tmpl := template.Must(template.New("index").Parse(indexHTML))
	srv := NewServer(Params{
		DistP: *distP,
		DistQ: *distQ,
		Sigma: *sigma,
		N:     *nSamp,
		Seed:  *seed,
	}, tmpl)

	http.HandleFunc("/", srv.HandleIndex)
	http.HandleFunc("/plot/", srv.HandlePlot)
	http.HandleFunc("/regenerate", srv.HandleRegenerate)
	http.HandleFunc("/update", srv.HandleUpdate)

	addr := fmt.Sprintf("localhost:%d", *port)
	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("Serving RKHS Explorer at %s\n", url)
	openBrowser(url)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	fmt.Println("\nShutting down.")
}

func exportSVGs(P, Q Distribution, sigma float64, nSamp int, seed int64, outDir string) {
	rng := rand.New(rand.NewSource(seed))
	samplesP := make([]float64, nSamp)
	samplesQ := make([]float64, nSamp)
	for i := 0; i < nSamp; i++ {
		samplesP[i] = P.Sample(rng)
		samplesQ[i] = Q.Sample(rng)
	}
	grid := linspace(-4.5, 5.5, 400)

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  RKHS Explorer  (export mode)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  P = %-10s  Q = %-10s  σ = %.3f  n = %d\n", P.Name, Q.Name, sigma, nSamp)
	fmt.Printf("  output → %s/\n", outDir)
	fmt.Println("───────────────────────────────────────────────────────────")

	type plotJob struct {
		name string
		gen  func() ([]byte, error)
	}
	jobs := []plotJob{
		{"01_kernel_functions", func() ([]byte, error) { return plotKernelFunctions(P, samplesP, sigma, grid) }},
		{"02_mean_embedding", func() ([]byte, error) { return plotMeanEmbedding(P, samplesP, sigma, grid) }},
		{"03_mmd_comparison", func() ([]byte, error) { return plotMMD(P, Q, samplesP, samplesQ, sigma, grid) }},
		{"04_gram_matrix", func() ([]byte, error) { return plotGramHeatmap(samplesP, sigma) }},
		{"05_sigma_sweep", func() ([]byte, error) { return plotSigmaSweep(P, Q, samplesP, samplesQ) }},
	}

	for _, j := range jobs {
		svg, err := j.gen()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", j.name, err)
			continue
		}
		path := outDir + "/" + j.name + ".svg"
		if err := os.WriteFile(path, svg, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", path, err)
			continue
		}
		fmt.Printf("  Wrote %s\n", path)
	}

	mmd2 := MMDSquared(samplesP, samplesQ, sigma)
	fmt.Println("\n───────────────────────────────────────────────────────────")
	fmt.Printf("  MMD²(P, Q) = %.6f\n", mmd2)
	fmt.Printf("  MMD(P, Q)  = %.6f\n", math.Sqrt(math.Max(0, mmd2)))
	if mmd2 < 0.005 {
		fmt.Println("  → Distributions appear SIMILAR in this RKHS")
	} else {
		fmt.Println("  → Distributions appear DIFFERENT in this RKHS")
	}
	fmt.Println("═══════════════════════════════════════════════════════════")
}
