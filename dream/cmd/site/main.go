package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/thatnealpatel/maths/dream/cmd/site/handlers"
)

var (
	//go:embed templates
	templateFS embed.FS

	addr = flag.String("addr", ":4111", "listen address")
)

// TODO(nealpatel): Fix idioms and commit CLAUDE.md
func main() {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	layoutTmpl := template.Must(template.ParseFS(templateFS, "templates/layout.tmpl"))
	indexTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/index.tmpl"))
	fourierTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/fourier-transform.tmpl"))
	ddplotsTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/dd-plots.tmpl"))
	concTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/concentration.tmpl"))
	c2stTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/c2st.tmpl"))
	spuriousTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/spurious-correlation.tmpl"))
	emmdTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/energy-vs-mmd.tmpl"))
	dsrTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/dsr.tmpl"))

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("cmd/site/static"))))

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := indexTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "Home"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /fourier-transform", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := fourierTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "Fourier Transform"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	handlers.RegisterFourier(mux)
	handlers.RegisterDSR(mux)

	log.Println("precomputing visualizations...")
	rob := handlers.NewRobustness(ctx)
	rob.Register(mux)
	log.Println("precomputation complete")

	mux.HandleFunc("GET /robustness/dd-plots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := ddplotsTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "DD-Plots"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /robustness/concentration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := concTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "Concentration of Measure"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /robustness/c2st", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := c2stTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "C2ST Power Surface"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /robustness/spurious-correlation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := spuriousTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "Spurious Correlation"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /robustness/energy-vs-mmd", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := emmdTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "Energy vs. MMD"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /dsr", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := dsrTmpl.ExecuteTemplate(w, "layout.tmpl", map[string]string{"Title": "Deflated Sharpe Ratio"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	srv := &http.Server{Addr: *addr, Handler: mux}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	fmt.Fprintf(os.Stderr, "listening on http://localhost%s\n", *addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
