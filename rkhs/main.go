package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	distP  = flag.String("p", "bimodal", "Distribution P: gaussian, bimodal, uniform, skewed")
	distQ  = flag.String("q", "gaussian", "Distribution Q: gaussian, bimodal, uniform, skewed")
	sigma  = flag.Float64("sigma", 0.5, "RBF kernel bandwidth")
	nSamp  = flag.Int("n", 12, "Number of samples per distribution")
	seed   = flag.Int64("seed", 42, "Random seed")
	port   = flag.Int("port", 8741, "HTTP server port")
	export = flag.Bool("export", false, "Export SVGs to disk instead of serving")
	outDir = flag.String("out", ".", "Output directory for -export")

	//go:embed index.tmpl
	indexTmpl string
)

func main() {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	P, okP := distributions[*distP]
	if !okP {
		log.Fatalf("unknown distribution P: %s\n", *distP)
	}

	Q, okQ := distributions[*distQ]
	if !okQ {
		log.Fatalf("unknown distribution Q: %s\n", *distQ)
	}

	if *export {
		exportSVGs(P, Q, *sigma, *nSamp, *seed, *outDir)
		return
	}

	tmpl := template.Must(template.New("index").Parse(indexTmpl))
	srv := NewServer(Params{
		DistP: *distP,
		DistQ: *distQ,
		Sigma: *sigma,
		N:     *nSamp,
		Seed:  *seed,
	}, tmpl)

	http.HandleFunc("/", srv.HandleIndex)
	http.HandleFunc("/plot/", srv.HandlePlot)
	http.HandleFunc("/regenerate", srv.HandleRegenerate)
	http.HandleFunc("/update", srv.HandleUpdate)

	addr := fmt.Sprintf("localhost:%d", *port)
	go http.ListenAndServe(addr, nil)
	fmt.Printf("Serving RKHS Explorer at http://%s\n", addr)
	<-ctx.Done()
}
