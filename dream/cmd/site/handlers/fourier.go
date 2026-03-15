package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
)

type plotData struct {
	Time timeData `json:"time"`
	Freq freqData `json:"freq"`
}

type timeData struct {
	X []float64 `json:"x"`
	Y []float64 `json:"y"`
}

type freqData struct {
	Xi        []float64 `json:"xi"`
	Magnitude []float64 `json:"magnitude"`
}

func RegisterFourier(mux *http.ServeMux) {
	mux.HandleFunc("POST /fourier-transform/compute", handleFourierCompute)
}

func handleFourierCompute(w http.ResponseWriter, r *http.Request) {
	expr, err := Parse(r.FormValue("expr"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	xmin, err := strconv.ParseFloat(r.FormValue("xmin"), 64)
	if err != nil {
		http.Error(w, "invalid xmin", http.StatusBadRequest)
		return
	}

	xmax, err := strconv.ParseFloat(r.FormValue("xmax"), 64)
	if err != nil {
		http.Error(w, "invalid xmax", http.StatusBadRequest)
		return
	}

	n, err := strconv.Atoi(r.FormValue("samples"))
	if err != nil || n < 1 || n > 4096 {
		http.Error(w, "invalid samples", http.StatusBadRequest)
		return
	}

	dx := (xmax - xmin) / float64(n)
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i := range n {
		xs[i] = xmin + float64(i)*dx
		ys[i] = expr(xs[i])
		if math.IsNaN(ys[i]) || math.IsInf(ys[i], 0) {
			ys[i] = 0
		}
	}

	nFreq := n / 2
	xis := make([]float64, nFreq)
	mags := make([]float64, nFreq)
	freqRes := 1.0 / (xmax - xmin)

	for k := range nFreq {
		xi := float64(k) * freqRes
		xis[k] = xi
		var re, im float64
		for j := range n {
			angle := -2 * math.Pi * xs[j] * xi
			re += ys[j] * math.Cos(angle)
			im += ys[j] * math.Sin(angle)
		}
		re *= dx
		im *= dx
		mags[k] = math.Sqrt(re*re + im*im)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plotData{
		Time: timeData{X: xs, Y: ys},
		Freq: freqData{Xi: xis, Magnitude: mags},
	})
}
