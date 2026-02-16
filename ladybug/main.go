package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cheggaaa/pb/v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	const N = 101
	nodes := make([]int, N)
	for i := range N {
		nodes[i] = i
	}

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		endsOn = make(map[int]int, N)
		sims   atomic.Int64
		// started = time.Now()
	)

	go func() {
		const tmpl = pb.ProgressBarTemplate(`{{ string . "probability" }}`)
		bars := make([]*pb.ProgressBar, 0, N)
		for range N {
			bars = append(bars, tmpl.New(0))
		}

		top := pb.ProgressBarTemplate(`{{ etime . }} | {{ string . "sims" }}`).New(0)
		pool, err := pb.StartPool(append([]*pb.ProgressBar{top}, bars...)...)
		if err != nil {
			panic(err)
		}
		defer func() {
			top.Finish()
			for i := range bars {
				bars[i].Finish()
			}
			pool.Stop()
		}()

		tick := time.NewTicker(250 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				top.Set("sims", sims.Load())
				mu.Lock()
				var total int
				for _, counts := range endsOn {
					total += counts
				}
				for i := range N {
					bars[i].Set("probability", fmt.Sprintf("P(%2d|12)=%.7f%%", i, float64(endsOn[i])/float64(total)*100))
				}
				mu.Unlock()
			}
		}
	}()

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				sims.Add(1)
				var (
					seen      = make(map[int]struct{}, N)
					last, idx int
				)
				for len(seen) < N {
					switch idx {
					case N:
						idx = 0
					case -1:
						idx = N - 1
					}

					last = nodes[idx]
					if _, found := seen[last]; !found {
						seen[last] = struct{}{}
					}

					j, err := rand.Int(rand.Reader, big.NewInt(2))
					if err != nil {
						panic(err)
					}
					switch j.Int64() {
					case 0:
						idx++
					case 1:
						idx--
					}
				}

				mu.Lock()
				endsOn[last]++
				mu.Unlock()
			}
		})
	}

	wg.Wait()
}

// markov generalizes the
// binary choice to n-choice.
type markov struct {
	value    int
	neighors []*markov
}

/*
6h27m17s | Finished simulations: 11412260950
P(11|12)=9.0906339%
P( 1|12)=9.0908978%
P(10|12)=9.0904659%
P( 2|12)=9.0913577%
P( 9|12)=9.0905983%
P( 3|12)=9.0909970%
P( 8|12)=9.0912162%
P( 4|12)=9.0912440%
P( 7|12)=9.0914132%
P( 5|12)=9.0903219%
P( 6|12)=9.0908541%
P(12|12)=0.0000000%
*/
