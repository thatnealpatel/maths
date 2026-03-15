package main

import (
	"math"
	"math/rand"
)

const MMDSimilarityThreshold = 0.005

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

func MMD(samplesP, samplesQ []float64, sigma float64) (mmd, mmd2 float64) {
	mmd2 = MMDSquared(samplesP, samplesQ, sigma)
	mmd = math.Sqrt(math.Max(0, mmd2))
	return
}

func linspace(lo, hi float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = lo + float64(i)/float64(n-1)*(hi-lo)
	}
	return out
}
