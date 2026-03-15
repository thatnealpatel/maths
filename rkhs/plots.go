package main

import (
	"bytes"
	"fmt"
	"math"

	"github.com/aclements/go-gg/gg"
	"github.com/aclements/go-gg/table"
)

func plotKernelFunctions(dist Distribution, samples []float64, sigma float64, grid []float64) ([]byte, error) {
	var xs, ys []float64
	var series []string

	for _, t := range grid {
		xs = append(xs, t)
		ys = append(ys, dist.PDF(t))
		series = append(series, "P(x) = "+dist.Name)
	}

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

	var buf bytes.Buffer
	plot.WriteSVG(&buf, 900, 500)
	return buf.Bytes(), nil
}

func plotMeanEmbedding(dist Distribution, samples []float64, sigma float64, grid []float64) ([]byte, error) {
	var xs, ys []float64
	var series []string

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

	var buf bytes.Buffer
	plot.WriteSVG(&buf, 900, 500)
	return buf.Bytes(), nil
}

func plotMMD(distP, distQ Distribution, samplesP, samplesQ []float64, sigma float64, grid []float64) ([]byte, error) {
	var xs, ys []float64
	var series []string

	muPVals := make([]float64, len(grid))
	for i, t := range grid {
		muPVals[i] = MeanEmbedding(samplesP, t, sigma)
		xs = append(xs, t)
		ys = append(ys, muPVals[i])
		series = append(series, "μ̂_P ("+distP.Name+")")
	}

	muQVals := make([]float64, len(grid))
	for i, t := range grid {
		muQVals[i] = MeanEmbedding(samplesQ, t, sigma)
		xs = append(xs, t)
		ys = append(ys, muQVals[i])
		series = append(series, "μ̂_Q ("+distQ.Name+")")
	}

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

	var buf bytes.Buffer
	plot.WriteSVG(&buf, 900, 500)
	return buf.Bytes(), nil
}

func plotGramHeatmap(samples []float64, sigma float64) ([]byte, error) {
	n := len(samples)
	if n > 40 {
		n = 40
	}
	sub := samples[:n]

	var xs, ys, vals []float64
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			xs = append(xs, float64(j))
			ys = append(ys, float64(n-1-i))
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

	var buf bytes.Buffer
	plot.WriteSVG(&buf, 600, 550)
	return buf.Bytes(), nil
}

func plotSigmaSweep(distP, distQ Distribution, samplesP, samplesQ []float64) ([]byte, error) {
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

	var buf bytes.Buffer
	plot.WriteSVG(&buf, 900, 400)
	return buf.Bytes(), nil
}
