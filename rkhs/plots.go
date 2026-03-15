package main

import (
	"bytes"
	"fmt"
	"math"

	"github.com/aclements/go-gg/gg"
	"github.com/aclements/go-gg/table"
)

const (
	maxKernelCurves    = 12
	maxEmbeddingCurves = 20
	maxGramSize        = 40
)

func renderSVG(plot *gg.Plot, w, h int) []byte {
	var buf bytes.Buffer
	plot.WriteSVG(&buf, w, h)
	return buf.Bytes()
}

func plotKernelFunctions(dist Distribution, samples []float64, sigma float64, grid []float64) ([]byte, error) {
	var xs, ys []float64
	var series []string

	for _, t := range grid {
		xs = append(xs, t)
		ys = append(ys, dist.PDF(t))
		series = append(series, "P(x) = "+dist.Name)
	}

	show := min(len(samples), maxKernelCurves)
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

	return renderSVG(plot, 900, 500), nil
}

func plotMeanEmbedding(dist Distribution, samples []float64, sigma float64, grid []float64) ([]byte, error) {
	var xs, ys []float64
	var series []string

	show := min(len(samples), maxEmbeddingCurves)
	for i := 0; i < show; i++ {
		xi := samples[i]
		for _, t := range grid {
			xs = append(xs, t)
			ys = append(ys, RBF(xi, t, sigma))
			series = append(series, "individual K(xi, ·)")
		}
	}

	for _, t := range grid {
		xs = append(xs, t)
		ys = append(ys, MeanEmbedding(samples, t, sigma))
		series = append(series, "μ̂_P(t) = (1/n) Σ K(xi, t)")
	}

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

	return renderSVG(plot, 900, 500), nil
}

func plotMMD(distP, distQ Distribution, samplesP, samplesQ []float64, sigma float64, grid []float64) ([]byte, error) {
	var xs, ys []float64
	var series []string

	muP := make([]float64, len(grid))
	muQ := make([]float64, len(grid))
	for i, t := range grid {
		muP[i] = MeanEmbedding(samplesP, t, sigma)
		muQ[i] = MeanEmbedding(samplesQ, t, sigma)
	}

	for i, t := range grid {
		xs = append(xs, t, t, t)
		ys = append(ys, muP[i], muQ[i], math.Abs(muP[i]-muQ[i]))
		series = append(series,
			"μ̂_P ("+distP.Name+")",
			"μ̂_Q ("+distQ.Name+")",
			"|μ̂_P − μ̂_Q| pointwise",
		)
	}

	tb := table.NewBuilder(nil)
	tb.Add("x", xs)
	tb.Add("y", ys)
	tb.Add("series", series)

	mmd, mmd2 := MMD(samplesP, samplesQ, sigma)

	plot := gg.NewPlot(tb.Done())
	plot.GroupBy("series")
	plot.SortBy("x")
	plot.Add(
		gg.LayerLines{X: "x", Y: "y", Color: "series"},
		gg.Title(fmt.Sprintf("Step 3: MMD(P,Q) = %.4f  |  MMD² = %.6f  [σ=%.2f]", mmd, mmd2, sigma)),
		gg.AxisLabel("x", "t"),
		gg.AxisLabel("y", "value"),
	)

	return renderSVG(plot, 900, 500), nil
}

func plotGramHeatmap(samples []float64, sigma float64) ([]byte, error) {
	n := min(len(samples), maxGramSize)
	samples = samples[:n]

	var xs, ys, vals []float64
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			xs = append(xs, float64(j))
			ys = append(ys, float64(n-1-i))
			vals = append(vals, RBF(samples[i], samples[j], sigma))
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

	return renderSVG(plot, 600, 550), nil
}

func plotSigmaSweep(distP, distQ Distribution, samplesP, samplesQ []float64) ([]byte, error) {
	sigmas := linspace(0.1, 3.0, 60)
	var xs, ys []float64
	var series []string

	for _, s := range sigmas {
		mmd, mmd2 := MMD(samplesP, samplesQ, s)
		xs = append(xs, s, s)
		ys = append(ys, mmd2, mmd)
		series = append(series, "MMD²", "MMD")
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

	return renderSVG(plot, 900, 400), nil
}
