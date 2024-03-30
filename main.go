package main

import (
	"encoding/json"
	"html/template"
	"log"
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

	mux.HandleFunc("POST /api/validate_chirp", handleValidateChirp)

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
	/*
		resBody := make(map[string]interface{})
		var status int
		if len(data.Body) > 140 {
			resBody["error"] = "Chirp is too long"
			status = http.StatusBadRequest
		} else {
			resBody["valid"] = true
			status = http.StatusOK
		}
		res, err := json.Marshal(resBody)
		if err != nil {
			log.Printf("Error encoding response body: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(status)
		w.Write(res)
	*/
	if len(data.Body) > 140 {
		err = respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Chirp is too long",
		})
	} else {
		err = respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"valid": true,
		})
	}
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.WriteHeader(code)
	w.Write(body)
	return nil
}
