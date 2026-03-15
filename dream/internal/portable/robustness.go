// Portable extraction of distributional consistency testing from
// kanacapital.io/backtester/internal/research/stage2.go.
//
// No dependencies on backtester internals (engine, marketdb, config, database).
// Only external dependency: gonum.org/v1/gonum/mat.
//
// Extracted functions and source line ranges:
//
//   Stage2, NewStage2FromMatrix     — new constructor replacing NewStage2 (was lines 17–22, 68–105)
//   Stage2Result, PermResult        — types (lines 24–35)
//   DDPlotData                      — type (lines 37–41)
//   C2STResult, FeatureImportance   — types (lines 43–52)
//   EnergyTest, MMDTest             — methods (lines 243–252)
//   euclideanGram                   — lines 254–269
//   laplacianGram                   — lines 271–286
//   medianL1                        — lines 288–304
//   energyFromGram                  — lines 306–329
//   mmdFromGram                     — lines 331–356
//   permTest                        — lines 358–389
//   ComputeDDPlot                   — method (lines 393–422)
//   projectionDepth                 — lines 424–453
//   randomDirections                — lines 455–471
//   medianSorted                    — lines 473–482
//   mad2                            — lines 484–491
//   C2ST                            — method (lines 524–635)
//   dtree, predict, addImportance   — lines 497–522
//   trainTree, splitNode, majority  — lines 637–743
//   Outliers (placeholder)          — line 747
//   Run                             — lines 751–758
//
// The Binomial CDF for C2ST p-values is reimplemented inline using
// the regularized incomplete beta function from gonum/mathext,
// replacing the gonum/stat/distuv.Binomial dependency to keep the
// import set minimal. However, since the user specified only
// gonum/mat + stdlib, the p-value is computed via a normal
// approximation to the binomial (valid for n >= 40).

package portable

import (
	"math"
	"math/rand/v2"
	"slices"

	"gonum.org/v1/gonum/mat"
)

type Stage2 struct {
	pool   *mat.Dense
	n1     int
	p      int
	labels []string
}

type Stage2Result struct {
	Energy PermResult
	MMD    PermResult
	DDPlot DDPlotData
	C2ST   C2STResult
}

type PermResult struct {
	Observed float64
	PValue   float64
	B        int
}

type DDPlotData struct {
	ISDepth  []float64
	OOSDepth []float64
	IsOOS    []bool
}

type C2STResult struct {
	Accuracy   float64
	PValue     float64
	Importance []FeatureImportance
}

type FeatureImportance struct {
	Name       string
	Importance float64
}

func NewStage2FromMatrix(pool *mat.Dense, n1 int, labels []string) *Stage2 {
	_, p := pool.Dims()
	return &Stage2{pool: pool, n1: n1, p: p, labels: labels}
}

// --- Layer 2: Energy Distance + MMD ---

func (s *Stage2) EnergyTest(B int) PermResult {
	gram := euclideanGram(s.pool)
	return permTest(gram, s.n1, B, energyFromGram)
}

func (s *Stage2) MMDTest(B int) PermResult {
	sigma := medianL1(s.pool)
	gram := laplacianGram(s.pool, sigma)
	return permTest(gram, s.n1, B, mmdFromGram)
}

func euclideanGram(X *mat.Dense) *mat.SymDense {
	n, p := X.Dims()
	gram := mat.NewSymDense(n, nil)

	for i := range n {
		for j := i + 1; j < n; j++ {
			var sum float64
			for k := range p {
				d := X.At(i, k) - X.At(j, k)
				sum += d * d
			}
			gram.SetSym(i, j, math.Sqrt(sum))
		}
	}
	return gram
}

func laplacianGram(X *mat.Dense, sigma float64) *mat.SymDense {
	n, p := X.Dims()
	gram := mat.NewSymDense(n, nil)

	for i := range n {
		gram.SetSym(i, i, 1)
		for j := i + 1; j < n; j++ {
			var l1 float64
			for k := range p {
				l1 += math.Abs(X.At(i, k) - X.At(j, k))
			}
			gram.SetSym(i, j, math.Exp(-l1/sigma))
		}
	}
	return gram
}

func medianL1(X *mat.Dense) float64 {
	n, p := X.Dims()
	dists := make([]float64, 0, n*(n-1)/2)

	for i := range n {
		for j := i + 1; j < n; j++ {
			var l1 float64
			for k := range p {
				l1 += math.Abs(X.At(i, k) - X.At(j, k))
			}
			dists = append(dists, l1)
		}
	}

	slices.Sort(dists)
	return medianSorted(dists)
}

func energyFromGram(gram *mat.SymDense, n1 int) float64 {
	n, _ := gram.Dims()
	n2 := n - n1

	var cross, within1, within2 float64
	for i := range n {
		for j := i + 1; j < n; j++ {
			d := gram.At(i, j)
			iIS, jIS := i < n1, j < n1
			switch {
			case iIS && jIS:
				within1 += d
			case !iIS && !jIS:
				within2 += d
			default:
				cross += d
			}
		}
	}

	return 2*cross/float64(n1*n2) -
		2*within1/float64(n1*n1) -
		2*within2/float64(n2*n2)
}

func mmdFromGram(gram *mat.SymDense, n1 int) float64 {
	n, _ := gram.Dims()
	n2 := n - n1

	var kIS, kOOS, kCross float64
	for i := range n {
		for j := i; j < n; j++ {
			k := gram.At(i, j)
			iIS, jIS := i < n1, j < n1
			mult := 2.0
			if i == j {
				mult = 1.0
			}
			switch {
			case iIS && jIS:
				kIS += mult * k
			case !iIS && !jIS:
				kOOS += mult * k
			default:
				kCross += mult * k
			}
		}
	}

	return kIS/float64(n1*n1) + kOOS/float64(n2*n2) - 2*kCross/float64(n1*n2)
}

func permTest(gram *mat.SymDense, n1, B int, stat func(*mat.SymDense, int) float64) PermResult {
	observed := stat(gram, n1)

	n, _ := gram.Dims()
	perm := make([]int, n)
	permGram := mat.NewSymDense(n, nil)

	var count int
	for b := range B {
		_ = b
		for i := range perm {
			perm[i] = i
		}
		rand.Shuffle(n, func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })

		for i := range n {
			for j := i; j < n; j++ {
				permGram.SetSym(i, j, gram.At(perm[i], perm[j]))
			}
		}

		if stat(permGram, n1) >= observed {
			count++
		}
	}

	return PermResult{
		Observed: observed,
		PValue:   float64(count+1) / float64(B+1),
		B:        B,
	}
}

// --- Layer 1: DD-Plot (Projection Depth) ---

func (s *Stage2) ComputeDDPlot(nDirs int) DDPlotData {
	n, _ := s.pool.Dims()
	n2 := n - s.n1

	dirs := randomDirections(nDirs, s.p)

	isRef := s.pool.Slice(0, s.n1, 0, s.p).(*mat.Dense)
	oosRef := s.pool.Slice(s.n1, n, 0, s.p).(*mat.Dense)

	result := DDPlotData{
		ISDepth:  make([]float64, n),
		OOSDepth: make([]float64, n),
		IsOOS:    make([]bool, n),
	}
	for i := s.n1; i < n; i++ {
		result.IsOOS[i] = true
	}

	isProj := make([]float64, s.n1)
	oosProj := make([]float64, n2)
	row := make([]float64, s.p)

	for i := range n {
		mat.Row(row, i, s.pool)
		result.ISDepth[i] = projectionDepth(row, isRef, s.n1, dirs, isProj)
		result.OOSDepth[i] = projectionDepth(row, oosRef, n2, dirs, oosProj)
	}

	return result
}

func projectionDepth(z []float64, ref *mat.Dense, nRef int, dirs [][]float64, projBuf []float64) float64 {
	var maxOutly float64

	for _, u := range dirs {
		var zProj float64
		for k, uk := range u {
			zProj += uk * z[k]
		}

		for i := range nRef {
			var dot float64
			for k, uk := range u {
				dot += uk * ref.At(i, k)
			}
			projBuf[i] = dot
		}

		sorted := slices.Clone(projBuf[:nRef])
		slices.Sort(sorted)
		med := medianSorted(sorted)
		md := mad2(sorted, med)

		if md > 1e-12 {
			outly := math.Abs(zProj-med) / md
			maxOutly = max(maxOutly, outly)
		}
	}

	return 1 / (1 + maxOutly)
}

func randomDirections(nDirs, p int) [][]float64 {
	dirs := make([][]float64, nDirs)
	for i := range nDirs {
		d := make([]float64, p)
		var norm float64
		for j := range p {
			d[j] = rand.NormFloat64()
			norm += d[j] * d[j]
		}
		norm = math.Sqrt(norm)
		for j := range p {
			d[j] /= norm
		}
		dirs[i] = d
	}
	return dirs
}

func medianSorted(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func mad2(sorted []float64, med float64) float64 {
	devs := make([]float64, len(sorted))
	for i, v := range sorted {
		devs[i] = math.Abs(v - med)
	}
	slices.Sort(devs)
	return medianSorted(devs) * 1.4826
}

// --- Layer 3: C2ST (Classifier Two-Sample Test) ---

const maxTreeDepth = 5

type dtree struct {
	feat      int
	threshold float64
	left      *dtree
	right     *dtree
	class     int
}

func (t *dtree) predict(row []float64) int {
	if t.left == nil {
		return t.class
	}
	if row[t.feat] <= t.threshold {
		return t.left.predict(row)
	}
	return t.right.predict(row)
}

func (t *dtree) addImportance(counts []float64) {
	if t.left == nil {
		return
	}
	counts[t.feat]++
	t.left.addImportance(counts)
	t.right.addImportance(counts)
}

func (s *Stage2) C2ST(T, folds int) C2STResult {
	n, _ := s.pool.Dims()

	X := make([][]float64, n)
	for i := range n {
		X[i] = make([]float64, s.p)
		mat.Row(X[i], i, s.pool)
	}
	y := make([]int, n)
	for i := s.n1; i < n; i++ {
		y[i] = 1
	}

	n2 := n - s.n1
	isIdx := make([]int, s.n1)
	for i := range s.n1 {
		isIdx[i] = i
	}
	oosIdx := make([]int, n2)
	for i := range n2 {
		oosIdx[i] = s.n1 + i
	}
	rand.Shuffle(s.n1, func(i, j int) { isIdx[i], isIdx[j] = isIdx[j], isIdx[i] })
	rand.Shuffle(n2, func(i, j int) { oosIdx[i], oosIdx[j] = oosIdx[j], oosIdx[i] })

	importance := make([]float64, s.p)
	var totalCorrect, totalTest int

	for fold := range folds {
		var trainX [][]float64
		var trainY []int
		var testX [][]float64
		var testY []int

		assignFold := func(idx []int, label int) {
			foldSize := len(idx) / folds
			start := fold * foldSize
			end := start + foldSize
			if fold == folds-1 {
				end = len(idx)
			}
			for i, id := range idx {
				if i >= start && i < end {
					testX = append(testX, X[id])
					testY = append(testY, label)
				} else {
					trainX = append(trainX, X[id])
					trainY = append(trainY, label)
				}
			}
		}
		assignFold(isIdx, 0)
		assignFold(oosIdx, 1)

		trees := make([]*dtree, T)
		for t := range T {
			trees[t] = trainTree(trainX, trainY, s.p)
			trees[t].addImportance(importance)
		}

		for i, row := range testX {
			votes := [2]int{}
			for _, tree := range trees {
				votes[tree.predict(row)]++
			}
			pred := 0
			if votes[1] > votes[0] {
				pred = 1
			}
			if pred == testY[i] {
				totalCorrect++
			}
			totalTest++
		}
	}

	accuracy := float64(totalCorrect) / float64(totalTest)
	pValue := binomialPValueNormalApprox(totalCorrect, totalTest)

	var totalSplits float64
	for _, v := range importance {
		totalSplits += v
	}
	if totalSplits == 0 {
		totalSplits = 1
	}

	imp := make([]FeatureImportance, s.p)
	for i := range s.p {
		imp[i] = FeatureImportance{
			Name:       s.labels[i],
			Importance: importance[i] / totalSplits,
		}
	}
	slices.SortFunc(imp, func(a, b FeatureImportance) int {
		if a.Importance > b.Importance {
			return -1
		}
		if a.Importance < b.Importance {
			return 1
		}
		return 0
	})

	return C2STResult{
		Accuracy:   accuracy,
		PValue:     pValue,
		Importance: imp,
	}
}

func binomialPValueNormalApprox(k, n int) float64 {
	mu := float64(n) * 0.5
	sigma := math.Sqrt(float64(n) * 0.25)
	z := (float64(k) - 0.5 - mu) / sigma
	return 0.5 * math.Erfc(z/math.Sqrt2)
}

func trainTree(X [][]float64, y []int, p int) *dtree {
	n := len(X)
	bag := make([]int, n)
	for i := range n {
		bag[i] = rand.IntN(n)
	}
	return splitNode(X, y, bag, p, 0)
}

func splitNode(X [][]float64, y []int, idx []int, p, depth int) *dtree {
	if depth >= maxTreeDepth || len(idx) < 2 {
		return &dtree{class: majority(y, idx)}
	}

	pure := true
	for _, i := range idx[1:] {
		if y[i] != y[idx[0]] {
			pure = false
			break
		}
	}
	if pure {
		return &dtree{class: y[idx[0]]}
	}

	var (
		bestGini   = math.Inf(1)
		bestFeat   int
		bestThresh float64
	)
	vals := make([]float64, len(idx))
	for feat := range p {
		for i, id := range idx {
			vals[i] = X[id][feat]
		}
		sorted := slices.Clone(vals)
		slices.Sort(sorted)
		sorted = slices.Compact(sorted)

		for ti := range len(sorted) - 1 {
			threshold := (sorted[ti] + sorted[ti+1]) / 2

			var l0, l1, r0, r1 float64
			for i, id := range idx {
				if vals[i] <= threshold {
					if y[id] == 0 {
						l0++
					} else {
						l1++
					}
				} else {
					if y[id] == 0 {
						r0++
					} else {
						r1++
					}
				}
			}

			lN, rN := l0+l1, r0+r1
			if lN == 0 || rN == 0 {
				continue
			}

			giniL := 1 - (l0/lN)*(l0/lN) - (l1/lN)*(l1/lN)
			giniR := 1 - (r0/rN)*(r0/rN) - (r1/rN)*(r1/rN)
			gini := (lN*giniL + rN*giniR) / float64(len(idx))

			if gini < bestGini {
				bestGini = gini
				bestFeat = feat
				bestThresh = threshold
			}
		}
	}

	if math.IsInf(bestGini, 1) {
		return &dtree{class: majority(y, idx)}
	}

	var leftIdx, rightIdx []int
	for _, id := range idx {
		if X[id][bestFeat] <= bestThresh {
			leftIdx = append(leftIdx, id)
		} else {
			rightIdx = append(rightIdx, id)
		}
	}

	return &dtree{
		feat:      bestFeat,
		threshold: bestThresh,
		left:      splitNode(X, y, leftIdx, p, depth+1),
		right:     splitNode(X, y, rightIdx, p, depth+1),
	}
}

func majority(y []int, idx []int) int {
	var c [2]int
	for _, i := range idx {
		c[y[i]]++
	}
	if c[1] > c[0] {
		return 1
	}
	return 0
}

// --- Layer 4: MCD Outlier Flagging (deferred) ---

type OutlierFlag struct {
	Index    int
	Distance float64
	IsOOS    bool
}

func (s *Stage2) Outliers() []OutlierFlag { return nil }

// --- Convenience ---

func (s *Stage2) Run(B, T, folds int) Stage2Result {
	return Stage2Result{
		Energy: s.EnergyTest(B),
		MMD:    s.MMDTest(B),
		DDPlot: s.ComputeDDPlot(1000),
		C2ST:   s.C2ST(T, folds),
	}
}
