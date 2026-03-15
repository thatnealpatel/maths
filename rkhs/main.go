package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
)

//go:embed index.html
var indexHTML string

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

func main() {
	distP := flag.String("p", "bimodal", "Distribution P: gaussian, bimodal, uniform, skewed")
	distQ := flag.String("q", "gaussian", "Distribution Q: gaussian, bimodal, uniform, skewed")
	sigma := flag.Float64("sigma", 0.5, "RBF kernel bandwidth")
	nSamp := flag.Int("n", 12, "Number of samples per distribution")
	seed := flag.Int64("seed", 42, "Random seed")
	port := flag.Int("port", 8741, "HTTP server port")
	export := flag.Bool("export", false, "Export SVGs to disk instead of serving")
	outDir := flag.String("out", ".", "Output directory for -export")
	flag.Parse()

	P, okP := distributions[*distP]
	Q, okQ := distributions[*distQ]
	if !okP {
		fmt.Fprintf(os.Stderr, "unknown distribution P: %s\n", *distP)
		os.Exit(1)
	}
	if !okQ {
		fmt.Fprintf(os.Stderr, "unknown distribution Q: %s\n", *distQ)
		os.Exit(1)
	}

	if *export {
		exportSVGs(P, Q, *sigma, *nSamp, *seed, *outDir)
		return
	}

	tmpl := template.Must(template.New("index").Parse(indexHTML))
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
	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("Serving RKHS Explorer at %s\n", url)
	openBrowser(url)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	fmt.Println("\nShutting down.")
}
