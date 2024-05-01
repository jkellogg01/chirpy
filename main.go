package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/jkellogg01/chirpy/internal/handlers"
	"github.com/jkellogg01/chirpy/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	devMode := flag.Bool("dev", false, "clear the database on startup")
	flag.Parse()
	if *devMode {
		os.Setenv("ENV", "DEV")
	}

	apiCfg, err := handlers.NewApiConfig(os.Getenv("JWT_SECRET"), "db.json")
	if err != nil {
		log.Fatalf("failed to generate api state: %s", err)
	}
	if os.Getenv("ENV") == "DEV" {
		log.Print("dev mode: clearing database")
		apiCfg.ClearDB()
	}
	metrics := &middleware.ApiMetrics{}

	mux := http.NewServeMux()
	corsMux := middleware.MiddlewareCors(mux)
	logMux := middleware.MiddlewareLogging(corsMux)
	mux.Handle(
		"/app/*",
		metrics.MiddlewareMetricsInc(
			http.StripPrefix(
				"/app",
				http.FileServer(http.Dir("."))),
		),
	)

	mux.HandleFunc("GET /admin/metrics", metrics.HandleGetMetrics)

	mux.HandleFunc("/api/reset", metrics.HandleResetMetrics)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /api/chirps", apiCfg.GetChirps)

	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.GetChirp)

	mux.HandleFunc("POST /api/chirps", apiCfg.CreateChirp)

    mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.DeleteChirp)

	mux.HandleFunc("POST /api/users", apiCfg.CreateUser)

	mux.HandleFunc("POST /api/login", apiCfg.AuthenticateUser)

	mux.HandleFunc("PUT /api/users", apiCfg.UpdateUser)
    
    mux.HandleFunc("POST /api/refresh", apiCfg.RefreshUser)

    mux.HandleFunc("POST /api/revoke", apiCfg.RevokeToken)

	app := http.Server{
		Addr:    ":8080",
		Handler: logMux,
	}
	app.ListenAndServe()
}
