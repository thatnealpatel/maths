package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"gonum.org/v1/gonum/stat/distuv"
)

var stdNormal = distuv.Normal{Mu: 0, Sigma: 1}

const euler = 0.5772156649015329

type dsrResult struct {
	Verdict    dsrVerdict    `json:"verdict"`
	SRStarCurve dsrSRStarCurve `json:"sr_star_curve"`
	MinTRL     dsrMinTRL     `json:"mintrl_curves"`
	Heatmap    dsrHeatmap    `json:"psr_heatmap"`
}

type dsrVerdict struct {
	SRStar      float64 `json:"sr_star"`
	PSR         float64 `json:"psr"`
	Pass        bool    `json:"pass"`
	V           float64 `json:"v"`
	SEInflation float64 `json:"se_inflation"`
}

type dsrSRStarCurve struct {
	NValues      []float64 `json:"n_values"`
	SRStarValues []float64 `json:"sr_star_values"`
}

type dsrMinTRL struct {
	SRValues     []float64 `json:"sr_values"`
	MinTRLActual []float64 `json:"mintrl_actual"`
	MinTRLNormal []float64 `json:"mintrl_normal"`
}

type dsrHeatmap struct {
	SkewValues []float64   `json:"skew_values"`
	KurtValues []float64   `json:"kurt_values"`
	PSRGrid    [][]float64 `json:"psr_grid"`
}

func RegisterDSR(mux *http.ServeMux) {
	mux.HandleFunc("GET /dsr/data", handleDSRData)
}

func handleDSRData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	sr, err := strconv.ParseFloat(q.Get("sr"), 64)
	if err != nil {
		http.Error(w, "invalid sr", http.StatusBadRequest)
		return
	}
	logN, err := strconv.ParseFloat(q.Get("n"), 64)
	if err != nil {
		http.Error(w, "invalid n", http.StatusBadRequest)
		return
	}
	n := math.Round(math.Pow(10, logN))
	if n < 1 {
		n = 1
	}
	t, err := strconv.ParseFloat(q.Get("t"), 64)
	if err != nil || t < 1 {
		http.Error(w, "invalid t", http.StatusBadRequest)
		return
	}
	skew, err := strconv.ParseFloat(q.Get("skew"), 64)
	if err != nil {
		http.Error(w, "invalid skew", http.StatusBadRequest)
		return
	}
	kurt, err := strconv.ParseFloat(q.Get("kurt"), 64)
	if err != nil {
		http.Error(w, "invalid kurt", http.StatusBadRequest)
		return
	}
	alpha, err := strconv.ParseFloat(q.Get("alpha"), 64)
	if err != nil || alpha <= 0 || alpha >= 1 {
		http.Error(w, "invalid alpha", http.StatusBadRequest)
		return
	}

	result := dsrCompute(sr, n, t, skew, kurt, alpha)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func dsrCompute(sr, n, t, skew, kurt, alpha float64) dsrResult {
	v := dsrVariance(sr, skew, kurt)
	srStar := dsrSRStar(v, t, n)
	psr := dsrPSR(sr, srStar, v, t)
	vNormal := dsrVariance(sr, 0, 3)
	seInflation := 1.0
	if vNormal > 0 && v > 0 {
		seInflation = math.Sqrt(v / vNormal)
	}

	verdict := dsrVerdict{
		SRStar:      srStar,
		PSR:         psr,
		Pass:        psr >= 1-alpha,
		V:           v,
		SEInflation: seInflation,
	}

	srStarCurve := dsrComputeSRStarCurve(v, t)
	mintrl := dsrComputeMinTRL(n, skew, kurt, alpha)
	heatmap := dsrComputeHeatmap(sr, n, t, alpha)

	return dsrResult{
		Verdict:     verdict,
		SRStarCurve: srStarCurve,
		MinTRL:      mintrl,
		Heatmap:     heatmap,
	}
}

func dsrVariance(sr, skew, kurt float64) float64 {
	return 1 - skew*sr + (kurt-1)/4*sr*sr
}

func dsrSRStar(v, t, n float64) float64 {
	if n <= 1 {
		return 0
	}
	sqrtVT := math.Sqrt(v / t)
	return sqrtVT * ((1-euler)*stdNormal.Quantile(1-1/n) + euler*stdNormal.Quantile(1-1/(n*math.E)))
}

func dsrPSR(sr, srStar, v, t float64) float64 {
	if v <= 0 {
		if sr > srStar {
			return 1
		}
		return 0
	}
	z := (sr - srStar) / math.Sqrt(v/t)
	return stdNormal.CDF(z)
}

func dsrComputeSRStarCurve(v, t float64) dsrSRStarCurve {
	nValues := make([]float64, 0, 100)
	srStarValues := make([]float64, 0, 100)

	for i := 0; i <= 100; i++ {
		logN := float64(i) / 100 * 4 // 10^0 to 10^4
		n := math.Pow(10, logN)
		if n < 1 {
			n = 1
		}
		nValues = append(nValues, n)
		srStarValues = append(srStarValues, dsrSRStar(v, t, n))
	}

	return dsrSRStarCurve{NValues: nValues, SRStarValues: srStarValues}
}

func dsrComputeMinTRL(n, skew, kurt, alpha float64) dsrMinTRL {
	zAlpha := stdNormal.Quantile(1 - alpha)
	srValues := make([]float64, 0, 71)
	mintrlActual := make([]float64, 0, 71)
	mintrlNormal := make([]float64, 0, 71)

	for i := 0; i <= 70; i++ {
		sr := 0.5 + float64(i)*0.05
		srValues = append(srValues, sr)
		mintrlActual = append(mintrlActual, dsrBisectMinTRL(sr, n, skew, kurt, zAlpha))
		mintrlNormal = append(mintrlNormal, dsrBisectMinTRL(sr, n, 0, 3, zAlpha))
	}

	return dsrMinTRL{SRValues: srValues, MinTRLActual: mintrlActual, MinTRLNormal: mintrlNormal}
}

func dsrBisectMinTRL(sr, n, skew, kurt, zAlpha float64) float64 {
	v := dsrVariance(sr, skew, kurt)

	lo, hi := 1.0, 100000.0
	for range 50 {
		mid := (lo + hi) / 2
		srStar := dsrSRStar(v, mid, n)
		z := (sr - srStar) / math.Sqrt(v/mid)
		if z >= zAlpha {
			hi = mid
		} else {
			lo = mid
		}
	}

	if hi >= 99999 {
		return -1
	}
	return math.Ceil(hi)
}

func dsrComputeHeatmap(sr, n, t, alpha float64) dsrHeatmap {
	skewValues := make([]float64, 31)
	for i := range skewValues {
		skewValues[i] = -6.0 + float64(i)*0.2
	}

	kurtValues := make([]float64, 38)
	for i := range kurtValues {
		kurtValues[i] = 3.0 + float64(i)*1.0
	}

	grid := make([][]float64, len(skewValues))
	for si, skew := range skewValues {
		row := make([]float64, len(kurtValues))
		for ki, kurt := range kurtValues {
			v := dsrVariance(sr, skew, kurt)
			if v <= 0 {
				row[ki] = -1
				continue
			}
			srStar := dsrSRStar(v, t, n)
			row[ki] = dsrPSR(sr, srStar, v, t)
		}
		grid[si] = row
	}

	return dsrHeatmap{
		SkewValues: skewValues,
		KurtValues: kurtValues,
		PSRGrid:    grid,
	}
}
