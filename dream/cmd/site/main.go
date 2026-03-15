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

//go:embed templates
var templateFS embed.FS

var addr = flag.String("addr", ":8081", "listen address")

func main() {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	layoutTmpl := template.Must(template.ParseFS(templateFS, "templates/layout.tmpl"))
	indexTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/index.tmpl"))
	fourierTmpl := template.Must(template.Must(layoutTmpl.Clone()).ParseFS(templateFS, "templates/fourier-transform.tmpl"))

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
