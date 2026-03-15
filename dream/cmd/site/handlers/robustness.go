package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"gonum.org/v1/gonum/mat"

	"github.com/thatnealpatel/maths/dream/internal/portable"
)

const (
	ddSamples    = 200
	ddDirections = 1000
)

type Robustness struct {
	ddDefaults  [3]ddPanelResult
	concDefault concResult
	c2stDefault c2stGridResult
	emmdDefault emmdResult
}

type ddPanelResult struct {
	Points []ddPoint `json:"points"`
	Panel  string    `json:"panel"`
}

type ddPoint struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Group int     `json:"group"`
}

func NewRobustness(ctx context.Context) *Robustness {
	h := &Robustness{}

	log.Println("  dd-plots: computing default state...")
	h.ddDefaults[0] = ddPanelResult{
		Points: ddCompute("a", 1.5, rand.New(rand.NewSource(42))),
		Panel:  "a",
	}
	h.ddDefaults[1] = ddPanelResult{
		Points: ddCompute("b", 2.0, rand.New(rand.NewSource(43))),
		Panel:  "b",
	}
	h.ddDefaults[2] = ddPanelResult{
		Points: ddCompute("c", 0.8, rand.New(rand.NewSource(44))),
		Panel:  "c",
	}

	log.Println("  concentration: computing pairwise distances...")
	h.concDefault = concCompute(200, rand.New(rand.NewSource(50)))

	log.Println("  c2st: computing power surface (this takes a while)...")
	h.c2stDefault = c2stCompute(ctx)

	log.Println("  energy-vs-mmd: computing permutation tests...")
	h.emmdDefault = emmdCompute(ctx)

	return h
}

func (h *Robustness) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /robustness/dd-plots/data", h.handleDDData)
	mux.HandleFunc("GET /robustness/concentration/data", h.handleConcData)
	mux.HandleFunc("GET /robustness/c2st/data", h.handleC2STData)
	mux.HandleFunc("GET /robustness/spurious-correlation/data", h.handleSpuriousData)
	mux.HandleFunc("GET /robustness/energy-vs-mmd/data", h.handleEMMDData)
}

func (h *Robustness) handleDDData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	panel := q.Get("panel")

	var result ddPanelResult
	switch panel {
	case "a":
		s := q.Get("shift")
		if s == "" || s == "1.5" {
			result = h.ddDefaults[0]
		} else {
			shift, err := strconv.ParseFloat(s, 64)
			if err != nil {
				http.Error(w, "invalid shift", http.StatusBadRequest)
				return
			}
			result = ddPanelResult{Points: ddCompute("a", shift, rand.New(rand.NewSource(42))), Panel: "a"}
		}
	case "b":
		s := q.Get("scale")
		if s == "" || s == "2" {
			result = h.ddDefaults[1]
		} else {
			scale, err := strconv.ParseFloat(s, 64)
			if err != nil {
				http.Error(w, "invalid scale", http.StatusBadRequest)
				return
			}
			result = ddPanelResult{Points: ddCompute("b", scale, rand.New(rand.NewSource(43))), Panel: "b"}
		}
	case "c":
		s := q.Get("rho")
		if s == "" || s == "0.8" {
			result = h.ddDefaults[2]
		} else {
			rho, err := strconv.ParseFloat(s, 64)
			if err != nil {
				http.Error(w, "invalid rho", http.StatusBadRequest)
				return
			}
			result = ddPanelResult{Points: ddCompute("c", rho, rand.New(rand.NewSource(44))), Panel: "c"}
		}
	default:
		http.Error(w, "invalid panel", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func ddCompute(panel string, param float64, rng *rand.Rand) []ddPoint {
	s1 := make([][2]float64, ddSamples)
	for i := range s1 {
		s1[i] = [2]float64{rng.NormFloat64(), rng.NormFloat64()}
	}

	raw := make([][2]float64, ddSamples)
	for i := range raw {
		raw[i] = [2]float64{rng.NormFloat64(), rng.NormFloat64()}
	}

	s2 := make([][2]float64, ddSamples)
	switch panel {
	case "a":
		for i, z := range raw {
			s2[i] = [2]float64{z[0] + param, z[1]}
		}
	case "b":
		for i, z := range raw {
			s2[i] = [2]float64{z[0] * param, z[1]}
		}
	case "c":
		sq := math.Sqrt(1 - param*param)
		for i, z := range raw {
			s2[i] = [2]float64{z[0], param*z[0] + sq*z[1]}
		}
	}

	dirs := ddDirectionVectors()

	pool := make([][2]float64, 0, 2*ddSamples)
	pool = append(pool, s1...)
	pool = append(pool, s2...)

	d1 := ddDepths(pool, s1, dirs)
	d2 := ddDepths(pool, s2, dirs)

	points := make([]ddPoint, len(pool))
	for i := range pool {
		group := 0
		if i >= ddSamples {
			group = 1
		}
		points[i] = ddPoint{X: d1[i], Y: d2[i], Group: group}
	}

	return points
}

// --- Concentration of Measure ---

type concResult struct {
	Edges  []float64   `json:"edges"`
	Curves []concCurve `json:"curves"`
}

type concCurve struct {
	P       int       `json:"p"`
	Density []float64 `json:"density"`
}

func (h *Robustness) handleConcData(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("n")
	var result concResult
	if s == "" || s == "200" {
		result = h.concDefault
	} else {
		n, err := strconv.Atoi(s)
		if err != nil {
			http.Error(w, "invalid n", http.StatusBadRequest)
			return
		}
		switch n {
		case 50, 100, 200, 500:
			result = concCompute(n, rand.New(rand.NewSource(50)))
		default:
			http.Error(w, "invalid n", http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func concCompute(n int, rng *rand.Rand) concResult {
	dims := [5]int{2, 5, 10, 18, 50}
	nPairs := n * (n - 1) / 2
	const nBins = 50

	allDists := make([][]float64, len(dims))
	globalMin := math.Inf(1)
	globalMax := math.Inf(-1)

	for di, p := range dims {
		points := make([][]float64, n)
		for i := range points {
			pt := make([]float64, p)
			for k := range pt {
				pt[k] = rng.NormFloat64()
			}
			points[i] = pt
		}

		sqrtP := math.Sqrt(float64(p))
		dists := make([]float64, 0, nPairs)
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				var sumSq float64
				for k := 0; k < p; k++ {
					d := points[i][k] - points[j][k]
					sumSq += d * d
				}
				dist := math.Sqrt(sumSq) / sqrtP
				dists = append(dists, dist)
				if dist < globalMin {
					globalMin = dist
				}
				if dist > globalMax {
					globalMax = dist
				}
			}
		}
		allDists[di] = dists
	}

	margin := (globalMax - globalMin) * 0.02
	lo := globalMin - margin
	hi := globalMax + margin
	binWidth := (hi - lo) / nBins

	edges := make([]float64, nBins+1)
	for i := range edges {
		edges[i] = lo + float64(i)*binWidth
	}

	curves := make([]concCurve, len(dims))
	for di, p := range dims {
		density := make([]float64, nBins)
		for _, d := range allDists[di] {
			bin := int((d - lo) / binWidth)
			if bin < 0 {
				bin = 0
			}
			if bin >= nBins {
				bin = nBins - 1
			}
			density[bin]++
		}
		nf := float64(len(allDists[di]))
		for i := range density {
			density[i] /= nf * binWidth
		}
		curves[di] = concCurve{P: p, Density: density}
	}

	return concResult{Edges: edges, Curves: curves}
}

// --- Shared-Denominator Spurious Correlation ---

type spuriousResult struct {
	Before [][]float64 `json:"before"`
	After  [][]float64 `json:"after"`
	Labels []string    `json:"labels"`
	Sigma  float64     `json:"sigma"`
}

func (h *Robustness) handleSpuriousData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	sigma := 0.3
	if s := q.Get("sigma"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0.01 || v > 1.0 {
			http.Error(w, "invalid sigma", http.StatusBadRequest)
			return
		}
		sigma = v
	}

	components := 5
	if s := q.Get("components"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 2 || v > 10 {
			http.Error(w, "invalid components", http.StatusBadRequest)
			return
		}
		components = v
	}

	mu := 0.5
	if s := q.Get("mu"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0 || v > 2.0 {
			http.Error(w, "invalid mu", http.StatusBadRequest)
			return
		}
		mu = v
	}

	result := spuriousCompute(sigma, components, mu, rand.New(rand.NewSource(60)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func spuriousCompute(sigma float64, components int, mu float64, rng *rand.Rand) spuriousResult {
	const nObs = 1000

	x := make([][]float64, components)
	for i := range x {
		row := make([]float64, nObs)
		for j := range row {
			row[j] = rng.NormFloat64() + mu
		}
		x[i] = row
	}

	d := make([]float64, nObs)
	for j := range d {
		d[j] = math.Exp(rng.NormFloat64() * sigma)
	}

	y := make([][]float64, components)
	for i := range y {
		row := make([]float64, nObs)
		for j := range row {
			row[j] = x[i][j] / d[j]
		}
		y[i] = row
	}

	labels := make([]string, components)
	for i := range labels {
		labels[i] = "C" + string(rune('\u2081'+rune(i)))
	}

	return spuriousResult{
		Before: corrMatrix(x),
		After:  corrMatrix(y),
		Labels: labels,
		Sigma:  sigma,
	}
}

func corrMatrix(rows [][]float64) [][]float64 {
	n := len(rows)
	nObs := len(rows[0])
	nf := float64(nObs)

	means := make([]float64, n)
	stds := make([]float64, n)
	for i, row := range rows {
		var sum float64
		for _, v := range row {
			sum += v
		}
		means[i] = sum / nf

		var sumSq float64
		for _, v := range row {
			d := v - means[i]
			sumSq += d * d
		}
		stds[i] = math.Sqrt(sumSq / nf)
	}

	mat := make([][]float64, n)
	for i := range mat {
		mat[i] = make([]float64, n)
		mat[i][i] = 1.0
	}
	for i := range mat {
		for j := i + 1; j < n; j++ {
			var cov float64
			for k := range nObs {
				cov += (rows[i][k] - means[i]) * (rows[j][k] - means[j])
			}
			cov /= nf
			r := cov / (stds[i] * stds[j])
			mat[i][j] = r
			mat[j][i] = r
		}
	}

	return mat
}

// --- Energy Distance vs. MMD ---

type emmdResult struct {
	Shifts  []string          `json:"shifts"`
	Results []emmdShiftResult `json:"results"`
}

type emmdShiftResult struct {
	Shift  string   `json:"shift"`
	Energy emmdStat `json:"energy"`
	MMD    emmdStat `json:"mmd"`
}

type emmdStat struct {
	RejectionRate float64   `json:"rejection_rate"`
	MeanPValue    float64   `json:"mean_pvalue"`
	PValues       []float64 `json:"pvalues"`
}

func (h *Robustness) handleEMMDData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.emmdDefault)
}

func emmdCompute(ctx context.Context) emmdResult {
	const (
		nPerGroup = 200
		nReps     = 50
		permB     = 500
	)
	var (
		shifts = []string{"location", "scale", "correlation", "tail"}
		total  = len(shifts) * nReps

		results = make([]emmdShiftResult, len(shifts))

		mu   sync.Mutex
		done int
		wg   sync.WaitGroup
	)

	for si := range results {
		results[si].Shift = shifts[si]
		results[si].Energy.PValues = make([]float64, nReps)
		results[si].MMD.PValues = make([]float64, nReps)
	}

	for si, shift := range shifts {
		for rep := range nReps {
			wg.Go(func() {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var (
					rng  = rand.New(rand.NewSource(int64(si*10000 + rep)))
					n    = 2 * nPerGroup
					data = make([]float64, n*2)
				)

				for i := range nPerGroup {
					data[i*2+0] = rng.NormFloat64()
					data[i*2+1] = rng.NormFloat64()
				}

				switch shift {
				case "location":
					for i := range nPerGroup {
						row := nPerGroup + i
						data[row*2+0] = rng.NormFloat64() + 1.0
						data[row*2+1] = rng.NormFloat64()
					}
				case "scale":
					for i := range nPerGroup {
						row := nPerGroup + i
						data[row*2+0] = rng.NormFloat64() * 2.0
						data[row*2+1] = rng.NormFloat64()
					}
				case "correlation":
					var (
						rho = 0.7
						sq  = math.Sqrt(1 - rho*rho)
					)
					for i := range nPerGroup {
						row := nPerGroup + i
						z0 := rng.NormFloat64()
						z1 := rng.NormFloat64()
						data[row*2+0] = z0
						data[row*2+1] = rho*z0 + sq*z1
					}
				case "tail":
					scale := math.Sqrt(3.0 / 5.0)
					for i := range nPerGroup {
						row := nPerGroup + i
						data[row*2+0] = tSample(rng, 5) * scale
						data[row*2+1] = tSample(rng, 5) * scale
					}
				}

				var (
					pool    = mat.NewDense(n, 2, data)
					s2      = portable.NewStage2FromMatrix(pool, nPerGroup, []string{"d0", "d1"})
					eResult = s2.EnergyTest(permB)
					mResult = s2.MMDTest(permB)
				)

				mu.Lock()
				defer mu.Unlock()
				results[si].Energy.PValues[rep] = eResult.PValue
				results[si].MMD.PValues[rep] = mResult.PValue
				done++
				if done%20 == 0 || done == total {
					log.Printf("energy-vs-mmd: %d/%d complete", done, total)
				}
			})
		}
	}

	wg.Wait()

	for si := range results {
		results[si].Energy.RejectionRate = rejRate(results[si].Energy.PValues)
		results[si].Energy.MeanPValue = meanF64(results[si].Energy.PValues)
		results[si].MMD.RejectionRate = rejRate(results[si].MMD.PValues)
		results[si].MMD.MeanPValue = meanF64(results[si].MMD.PValues)
	}

	return emmdResult{Shifts: shifts, Results: results}
}

func tSample(rng *rand.Rand, nu int) float64 {
	z := rng.NormFloat64()
	var chi2 float64
	for range nu {
		x := rng.NormFloat64()
		chi2 += x * x
	}
	return z / math.Sqrt(chi2/float64(nu))
}

func rejRate(pvals []float64) float64 {
	var count int
	for _, p := range pvals {
		if p < 0.05 {
			count++
		}
	}
	return float64(count) / float64(len(pvals))
}

func meanF64(vals []float64) float64 {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// --- C2ST Power Surface ---

type c2stGridResult struct {
	Cells     []c2stCell `json:"cells"`
	RhoValues []float64  `json:"rho_values"`
	PValues   []int      `json:"p_values"`
}

type c2stCell struct {
	Rho           float64 `json:"rho"`
	P             int     `json:"p"`
	RejectionRate float64 `json:"rejection_rate"`
	MeanAcc       float64 `json:"mean_acc"`
	StdAcc        float64 `json:"std_acc"`
}

func (h *Robustness) handleC2STData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.c2stDefault)
}

// TODO(nealpatel): Without refectoring, generated
// code took ~4.5 minutes to finish on M3 MBA.
func c2stCompute(ctx context.Context) c2stGridResult {
	return c2stGridResult{}
	const (
		nReps     = 30  // set to 50 for tighter std dev
		nPerGroup = 500 // n=1000 permutations that pass Stage 1 (e.g. DSR)
		nTrees    = 100
		nFolds    = 5
	)
	var (
		rhoValues = []float64{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}
		pValues   = []int{3, 5, 8, 12, 16, 20}
		nCells    = len(rhoValues) * len(pValues) // TODO: Can be const

		mu    sync.Mutex
		cells = make([]c2stCell, nCells)
		done  int

		// sem = make(chan struct{}, runtime.GOMAXPROCS(0))
		wg sync.WaitGroup
	)
	for ri, rho := range rhoValues {
		for pi, p := range pValues {
			idx := ri*len(pValues) + pi
			wg.Go(func() {
				var (
					accs              = make([]float64, nReps)
					rejections        int
					sqrtOneMinusRhoSq = math.Sqrt(1 - rho*rho)
				)
				for rep := range nReps {
					select {
					case <-ctx.Done():
						return
					default:
					}
					var (
						rng  = rand.New(rand.NewSource(int64(idx*1000 + rep)))
						n    = 2 * nPerGroup
						data = make([]float64, n*p)
					)
					for i := range nPerGroup {
						for k := range p {
							data[i*p+k] = rng.NormFloat64()
						}
					}

					for i := range nPerGroup {
						var (
							row = nPerGroup + i
							z   = make([]float64, p)
						)
						for k := range p {
							z[k] = rng.NormFloat64()
						}
						data[row*p+0] = z[0]
						if p > 1 {
							data[row*p+1] = rho*z[0] + sqrtOneMinusRhoSq*z[1]
						}
						for k := 2; k < p; k++ {
							data[row*p+k] = z[k]
						}
					}

					var (
						pool   = mat.NewDense(n, p, data)
						labels = make([]string, p)
					)
					for k := range p {
						labels[k] = fmt.Sprintf("d%d", k)
					}

					var (
						s2     = portable.NewStage2FromMatrix(pool, nPerGroup, labels)
						result = s2.C2ST(nTrees, nFolds)
					)
					accs[rep] = result.Accuracy
					if result.PValue < 0.05 {
						rejections++
					}
				}

				var sum, sumSq float64
				for _, a := range accs {
					sum += a
					sumSq += a * a
				}
				var (
					meanAcc  = sum / float64(nReps)
					variance = max(0, sumSq/float64(nReps)-meanAcc*meanAcc)
					stdAcc   = math.Sqrt(variance)
					rejRate  = float64(rejections) / float64(nReps)
				)

				mu.Lock()
				defer mu.Unlock()
				cells[idx] = c2stCell{
					Rho:           rho,
					P:             p,
					RejectionRate: rejRate,
					MeanAcc:       meanAcc,
					StdAcc:        stdAcc,
				}
				done++
				log.Printf("c2st: cell (ρ=%.2f, p=%d) complete, rejection rate = %.2f [%d/%d]", rho, p, rejRate, done, nCells)
			})
		}
	}

	wg.Wait()

	return c2stGridResult{
		Cells:     cells,
		RhoValues: rhoValues,
		PValues:   pValues,
	}
}

// --- DD-Plots helpers ---

func ddDirectionVectors() [][2]float64 {
	dirs := make([][2]float64, ddDirections)
	for k := range dirs {
		theta := 2 * math.Pi * float64(k) / float64(ddDirections)
		dirs[k] = [2]float64{math.Cos(theta), math.Sin(theta)}
	}
	return dirs
}

func ddDepths(pool, ref [][2]float64, dirs [][2]float64) []float64 {
	n := len(ref)
	nf := float64(n)
	depths := make([]float64, len(pool))
	for i := range depths {
		depths[i] = 1.0
	}

	proj := make([]float64, n)

	for _, u := range dirs {
		for j, x := range ref {
			proj[j] = u[0]*x[0] + u[1]*x[1]
		}
		sort.Float64s(proj)

		for i, z := range pool {
			zp := u[0]*z[0] + u[1]*z[1]
			count := sort.Search(n, func(j int) bool { return proj[j] > zp })
			frac := float64(count) / nf
			if frac < depths[i] {
				depths[i] = frac
			}
		}
	}

	return depths
}
