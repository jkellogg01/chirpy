package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/jkellogg01/chirpy/internal/database"
)

type apiConfig struct {
	fileServerHits int
	db             *database.DB
}

func main() {
	db, err := database.NewDB("./db.json")
	if err != nil {
		log.Printf("Failed to connect to DB: %s", err)
	}
	db.ClearDB()
	apiCfg := &apiConfig{
		db: db,
	}

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

	mux.HandleFunc("GET /api/chirps", apiCfg.handleGetChirps)

	mux.HandleFunc("POST /api/chirps", apiCfg.handleCreateChirp)

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

func (cfg *apiConfig) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps()
	if errors.Is(err, database.ErrDBEmpty) {
        log.Println("nothing to see here")
        w.WriteHeader(http.StatusNoContent)
        return
	} else if err != nil {
		log.Printf("failed to retrieve chirps from database: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusOK, chirps)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) handleCreateChirp(w http.ResponseWriter, r *http.Request) {

}

func handleValidateChirp(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("Error decoding request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(data.Body) > 140 {
		err = respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Chirp is too long",
		})
	} else {
		badWords := []string{"kerfuffle", "sharbert", "fornax"}
		clean := replaceWords(data.Body, "****", badWords)
		err = respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"cleaned_body": clean,
		})
	}
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func replaceWords(msg, clean string, replace []string) string {
	words := strings.Split(msg, " ")
	for i, word := range words {
		for _, bad := range replace {
			if strings.ToLower(word) == strings.ToLower(bad) {
				words[i] = clean
			}
		}
	}
	return strings.Join(words, " ")
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.WriteHeader(code)
	w.Write(body)
	return nil
}
