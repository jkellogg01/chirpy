package main

import (
	"html/template"
	"net/http"
)

func main() {
	apiCfg := new(apiConfig)
	mux := http.NewServeMux()
	corsMux := middlewareCors(mux)
	mux.Handle(
		"/app/*",
		apiCfg.middlewareMetricsInc(
			http.StripPrefix(
				"/app",
				http.FileServer(http.Dir("."))),
		),
	)

	mux.HandleFunc("GET /admin/metrics", apiCfg.handleGetMetrics)

    mux.HandleFunc("/api/reset", apiCfg.handleResetMetrics)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	app := http.Server{
		Addr:    ":8080",
		Handler: corsMux,
	}
	app.ListenAndServe()
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type apiConfig struct {
	fileServerHits int
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits += 1
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
    tmpl, err := template.ParseGlob("*.html")
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
    }
	w.WriteHeader(http.StatusOK)
    tmpl.ExecuteTemplate(w, "metrics.html", map[string]interface{}{
        "Hits": cfg.fileServerHits,
    })
}

func (cfg *apiConfig) handleResetMetrics(w http.ResponseWriter, r *http.Request) {
    cfg.fileServerHits = 0
    w.WriteHeader(http.StatusOK)
}
