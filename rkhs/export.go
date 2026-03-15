package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
)

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
			log.Printf("error generating %s: %v", j.name, err)
			continue
		}
		path := outDir + "/" + j.name + ".svg"
		if err := os.WriteFile(path, svg, 0644); err != nil {
			log.Printf("error writing %s: %v", path, err)
			continue
		}
		fmt.Printf("  Wrote %s\n", path)
	}

	mmd, mmd2 := MMD(samplesP, samplesQ, sigma)
	fmt.Println("\n───────────────────────────────────────────────────────────")
	fmt.Printf("  MMD²(P, Q) = %.6f\n", mmd2)
	fmt.Printf("  MMD(P, Q)  = %.6f\n", mmd)
	if mmd2 < MMDSimilarityThreshold {
		fmt.Println("  → Distributions appear SIMILAR in this RKHS")
	} else {
		fmt.Println("  → Distributions appear DIFFERENT in this RKHS")
	}
	fmt.Println("═══════════════════════════════════════════════════════════")
}
