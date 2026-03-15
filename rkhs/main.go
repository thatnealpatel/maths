package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"

	"github.com/aclements/go-gg/gg"
	"github.com/aclements/go-gg/table"
)

// --- Distributions ---

type Distribution struct {
	Name   string
	Sample func(rng *rand.Rand) float64
	PDF    func(x float64) float64
}

var Gaussian = Distribution{
	Name:   "gaussian",
	Sample: func(rng *rand.Rand) float64 { return rng.NormFloat64() },
	PDF:    func(x float64) float64 { return math.Exp(-x*x/2) / math.Sqrt(2*math.Pi) },
}

var Bimodal = Distribution{
	Name: "bimodal",
	Sample: func(rng *rand.Rand) float64 {
		if rng.Float64() < 0.5 {
			return -1.5 + rng.NormFloat64()*0.5
		}
		return 1.5 + rng.NormFloat64()*0.5
	},
	PDF: func(x float64) float64 {
		a := math.Exp(-(x+1.5)*(x+1.5)/0.5) / math.Sqrt(math.Pi*0.5)
		b := math.Exp(-(x-1.5)*(x-1.5)/0.5) / math.Sqrt(math.Pi*0.5)
		return 0.5*a + 0.5*b
	},
}

var Uniform = Distribution{
	Name:   "uniform",
	Sample: func(rng *rand.Rand) float64 { return (rng.Float64() - 0.5) * 5 },
	PDF: func(x float64) float64 {
		if x >= -2.5 && x <= 2.5 {
			return 0.2
		}
		return 0
	},
}

var Skewed = Distribution{
	Name:   "skewed",
	Sample: func(rng *rand.Rand) float64 { return -math.Log(1-rng.Float64()) / 1.5 },
	PDF: func(x float64) float64 {
		if x < 0 {
			return 0
		}
		return 1.5 * math.Exp(-1.5*x)
	},
}

var distributions = map[string]Distribution{
	"gaussian": Gaussian,
	"bimodal":  Bimodal,
	"uniform":  Uniform,
	"skewed":   Skewed,
}

// --- Kernel and RKHS operations ---

func RBF(a, b, sigma float64) float64 {
	d := a - b
	return math.Exp(-d * d / (2 * sigma * sigma))
}

func MeanEmbedding(samples []float64, t, sigma float64) float64 {
	sum := 0.0
	for _, xi := range samples {
		sum += RBF(xi, t, sigma)
	}
	return sum / float64(len(samples))
}

func MMDSquared(samplesP, samplesQ []float64, sigma float64) float64 {
	n, m := len(samplesP), len(samplesQ)
	var ePP, eQQ, ePQ float64
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			ePP += RBF(samplesP[i], samplesP[j], sigma)
		}
	}
	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			eQQ += RBF(samplesQ[i], samplesQ[j], sigma)
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			ePQ += RBF(samplesP[i], samplesQ[j], sigma)
		}
	}
	ePP /= float64(n * n)
	eQQ /= float64(m * m)
	ePQ /= float64(n * m)
	return ePP + eQQ - 2*ePQ
}

func linspace(lo, hi float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = lo + float64(i)/float64(n-1)*(hi-lo)
	}
	return out
}

// --- Plot builders ---

// plotKernelFunctions creates an SVG showing P(x) and individual K(xi, ·) curves.
func plotKernelFunctions(path string, dist Distribution, samples []float64, sigma float64, grid []float64) {
	var xs, ys []float64
	var series []string

	// PDF curve.
	for _, t := range grid {
		xs = append(xs, t)
		ys = append(ys, dist.PDF(t))
		series = append(series, "P(x) = "+dist.Name)
	}

	// Kernel functions for each sample (cap at 12 for readability).
	show := len(samples)
	if show > 12 {
		show = 12
	}
	for i := 0; i < show; i++ {
		xi := samples[i]
		label := fmt.Sprintf("K(x%d=%.2f, ·)", i, xi)
		for _, t := range grid {
			xs = append(xs, t)
			ys = append(ys, RBF(xi, t, sigma))
			series = append(series, label)
		}
	}

	tb := table.NewBuilder(nil)
	tb.Add("x", xs)
	tb.Add("y", ys)
	tb.Add("series", series)

	plot := gg.NewPlot(tb.Done())
	plot.GroupBy("series")
	plot.SortBy("x")
	plot.Add(
		gg.LayerLines{X: "x", Y: "y", Color: "series"},
		gg.Title(fmt.Sprintf("Step 1: Each sample xi spawns K(xi, ·) in the RKHS  [σ=%.2f]", sigma)),
		gg.AxisLabel("x", "t"),
		gg.AxisLabel("y", "value"),
	)

	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, err)
		return
	}
	defer f.Close()
	plot.WriteSVG(f, 900, 500)
	fmt.Printf("  Wrote %s\n", path)
}

// plotMeanEmbedding creates an SVG showing individual K(xi, ·), the mean embedding, and P(x).
func plotMeanEmbedding(path string, dist Distribution, samples []float64, sigma float64, grid []float64) {
	var xs, ys []float64
	var series []string

	// Individual kernel functions (faded).
	show := len(samples)
	if show > 20 {
		show = 20
	}
	for i := 0; i < show; i++ {
		xi := samples[i]
		for _, t := range grid {
			xs = append(xs, t)
			ys = append(ys, RBF(xi, t, sigma))
			series = append(series, "individual K(xi, ·)")
		}
	}

	// Mean embedding.
	for _, t := range grid {
		xs = append(xs, t)
		ys = append(ys, MeanEmbedding(samples, t, sigma))
		series = append(series, "μ̂_P(t) = (1/n) Σ K(xi, t)")
	}

	// PDF (reference).
	for _, t := range grid {
		xs = append(xs, t)
		ys = append(ys, dist.PDF(t))
		series = append(series, "P(x) reference")
	}

	tb := table.NewBuilder(nil)
	tb.Add("x", xs)
	tb.Add("y", ys)
	tb.Add("series", series)

	plot := gg.NewPlot(tb.Done())
	plot.GroupBy("series")
	plot.SortBy("x")
	plot.Add(
		gg.LayerLines{X: "x", Y: "y", Color: "series"},
		gg.Title(fmt.Sprintf("Step 2: Mean embedding μ̂_P  [n=%d, σ=%.2f]", len(samples), sigma)),
		gg.AxisLabel("x", "t"),
		gg.AxisLabel("y", "value"),
	)

	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, err)
		return
	}
	defer f.Close()
	plot.WriteSVG(f, 900, 500)
	fmt.Printf("  Wrote %s\n", path)
}

// plotMMD creates an SVG comparing two mean embeddings with a shaded difference region.
func plotMMD(path string, distP, distQ Distribution, samplesP, samplesQ []float64, sigma float64, grid []float64) {
	var xs, ys []float64
	var series []string

	// Mean embedding P.
	muPVals := make([]float64, len(grid))
	for i, t := range grid {
		muPVals[i] = MeanEmbedding(samplesP, t, sigma)
		xs = append(xs, t)
		ys = append(ys, muPVals[i])
		series = append(series, "μ̂_P ("+distP.Name+")")
	}

	// Mean embedding Q.
	muQVals := make([]float64, len(grid))
	for i, t := range grid {
		muQVals[i] = MeanEmbedding(samplesQ, t, sigma)
		xs = append(xs, t)
		ys = append(ys, muQVals[i])
		series = append(series, "μ̂_Q ("+distQ.Name+")")
	}

	// Pointwise absolute difference.
	for i, t := range grid {
		xs = append(xs, t)
		ys = append(ys, math.Abs(muPVals[i]-muQVals[i]))
		series = append(series, "|μ̂_P − μ̂_Q| pointwise")
	}

	tb := table.NewBuilder(nil)
	tb.Add("x", xs)
	tb.Add("y", ys)
	tb.Add("series", series)

	mmd2 := MMDSquared(samplesP, samplesQ, sigma)
	mmd := math.Sqrt(math.Max(0, mmd2))

	plot := gg.NewPlot(tb.Done())
	plot.GroupBy("series")
	plot.SortBy("x")
	plot.Add(
		gg.LayerLines{X: "x", Y: "y", Color: "series"},
		gg.Title(fmt.Sprintf("Step 3: MMD(P,Q) = %.4f  |  MMD² = %.6f  [σ=%.2f]", mmd, mmd2, sigma)),
		gg.AxisLabel("x", "t"),
		gg.AxisLabel("y", "value"),
	)

	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, err)
		return
	}
	defer f.Close()
	plot.WriteSVG(f, 900, 500)
	fmt.Printf("  Wrote %s\n", path)
}

// plotGramHeatmap creates an SVG heatmap of the Gram matrix using LayerTiles.
func plotGramHeatmap(path string, samples []float64, sigma float64) {
	n := len(samples)
	if n > 40 {
		n = 40
	}
	sub := samples[:n]

	var xs, ys, vals []float64
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			xs = append(xs, float64(j))
			ys = append(ys, float64(n-1-i)) // flip so (0,0) is top-left
			vals = append(vals, RBF(sub[i], sub[j], sigma))
		}
	}

	tb := table.NewBuilder(nil)
	tb.Add("j", xs)
	tb.Add("i", ys)
	tb.Add("K(xi,xj)", vals)

	plot := gg.NewPlot(tb.Done())
	plot.Add(
		gg.LayerTiles{X: "j", Y: "i", Fill: "K(xi,xj)"},
		gg.Title(fmt.Sprintf("Gram matrix K(xi, xj)  [%dx%d, σ=%.2f]", n, n, sigma)),
		gg.AxisLabel("x", "j"),
		gg.AxisLabel("y", "i"),
	)

	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, err)
		return
	}
	defer f.Close()
	plot.WriteSVG(f, 600, 550)
	fmt.Printf("  Wrote %s\n", path)
}

// plotSigmaSweep creates an SVG showing how MMD varies with bandwidth.
func plotSigmaSweep(path string, distP, distQ Distribution, samplesP, samplesQ []float64) {
	sigmas := linspace(0.1, 3.0, 60)
	var xs, ys []float64
	var series []string

	for _, s := range sigmas {
		m2 := MMDSquared(samplesP, samplesQ, s)
		xs = append(xs, s)
		ys = append(ys, math.Max(0, m2))
		series = append(series, "MMD²")

		xs = append(xs, s)
		ys = append(ys, math.Sqrt(math.Max(0, m2)))
		series = append(series, "MMD")
	}

	tb := table.NewBuilder(nil)
	tb.Add("sigma", xs)
	tb.Add("value", ys)
	tb.Add("metric", series)

	plot := gg.NewPlot(tb.Done())
	plot.GroupBy("metric")
	plot.SortBy("sigma")
	plot.Add(
		gg.LayerLines{X: "sigma", Y: "value", Color: "metric"},
		gg.Title(fmt.Sprintf("Sigma sweep: %s vs %s  [n=%d]", distP.Name, distQ.Name, len(samplesP))),
		gg.AxisLabel("x", "σ (bandwidth)"),
		gg.AxisLabel("y", "distance"),
	)

	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, err)
		return
	}
	defer f.Close()
	plot.WriteSVG(f, 900, 400)
	fmt.Printf("  Wrote %s\n", path)
}

// --- Main ---

func main() {
	distP := flag.String("p", "bimodal", "Distribution P: gaussian, bimodal, uniform, skewed")
	distQ := flag.String("q", "gaussian", "Distribution Q: gaussian, bimodal, uniform, skewed")
	sigma := flag.Float64("sigma", 0.5, "RBF kernel bandwidth")
	nSamp := flag.Int("n", 12, "Number of samples per distribution")
	seed := flag.Int64("seed", 42, "Random seed (0 for non-deterministic)")
	mode := flag.String("mode", "all", "Mode: kernels, embedding, mmd, gram, sweep, all")
	outDir := flag.String("out", ".", "Output directory for SVG files")
	gridN := flag.Int("grid", 400, "Number of grid points for function evaluation")
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

	rng := rand.New(rand.NewSource(*seed))
	samplesP := make([]float64, *nSamp)
	samplesQ := make([]float64, *nSamp)
	for i := 0; i < *nSamp; i++ {
		samplesP[i] = P.Sample(rng)
		samplesQ[i] = Q.Sample(rng)
	}

	grid := linspace(-4.5, 5.5, *gridN)

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  RKHS Explorer  (go-gg edition)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  P = %-10s  Q = %-10s  σ = %.3f  n = %d\n", P.Name, Q.Name, *sigma, *nSamp)
	fmt.Printf("  output → %s/\n", *outDir)
	fmt.Println("───────────────────────────────────────────────────────────")

	doAll := *mode == "all"

	if doAll || *mode == "kernels" {
		plotKernelFunctions(
			*outDir+"/01_kernel_functions.svg",
			P, samplesP, *sigma, grid,
		)
	}

	if doAll || *mode == "embedding" {
		plotMeanEmbedding(
			*outDir+"/02_mean_embedding.svg",
			P, samplesP, *sigma, grid,
		)
	}

	if doAll || *mode == "mmd" {
		plotMMD(
			*outDir+"/03_mmd_comparison.svg",
			P, Q, samplesP, samplesQ, *sigma, grid,
		)
	}

	if doAll || *mode == "gram" {
		plotGramHeatmap(
			*outDir+"/04_gram_matrix.svg",
			samplesP, *sigma,
		)
	}

	if doAll || *mode == "sweep" {
		plotSigmaSweep(
			*outDir+"/05_sigma_sweep.svg",
			P, Q, samplesP, samplesQ,
		)
	}

	// Print numerical summary regardless of mode.
	mmd2 := MMDSquared(samplesP, samplesQ, *sigma)
	fmt.Println("\n───────────────────────────────────────────────────────────")
	fmt.Printf("  MMD²(P, Q) = %.6f\n", mmd2)
	fmt.Printf("  MMD(P, Q)  = %.6f\n", math.Sqrt(math.Max(0, mmd2)))
	if mmd2 < 0.005 {
		fmt.Println("  → Distributions appear SIMILAR in this RKHS")
	} else {
		fmt.Println("  → Distributions appear DIFFERENT in this RKHS")
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("  Usage examples:")
	fmt.Println("    go run . -p skewed -q bimodal -sigma 0.3 -n 50")
	fmt.Println("    go run . -mode gram -n 25 -sigma 1.0")
	fmt.Println("    go run . -mode sweep -p uniform -q skewed -n 40")
	fmt.Println("    go run . -out ./plots")
	fmt.Println()
	fmt.Println("  SVG files open in any browser. Try:")
	fmt.Println("    " + strings.Join([]string{
		"open 01_kernel_functions.svg",
		"open 02_mean_embedding.svg",
		"open 03_mmd_comparison.svg",
		"open 04_gram_matrix.svg",
		"open 05_sigma_sweep.svg",
	}, "\n    "))
	fmt.Println("═══════════════════════════════════════════════════════════")
}
