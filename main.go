package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jkellogg01/chirpy/internal/database"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type apiConfig struct {
	fileServerHits int
	db             *database.DB
	jwtSecret      []byte
}

func main() {
	godotenv.Load()
	devMode := flag.Bool("dev", false, "clear the database on startup")
	flag.Parse()

	db, err := database.NewDB("db.json")
	if err != nil {
		log.Printf("Failed to connect to DB: %s", err)
	}
	secret, err := base64.RawStdEncoding.DecodeString(os.Getenv("JWT_SECRET"))
	if err != nil {
		log.Fatalf("Failed to decode JWT secret: %s", err)
	}
	if *devMode {
		log.Print("dev mode: clearing database")
		db.ClearDB()
	}
	apiCfg := &apiConfig{
		fileServerHits: 0,
		db:             db,
		jwtSecret:      secret,
	}

	mux := http.NewServeMux()
	corsMux := middlewareCors(mux)
	logMux := middlewareLogging(corsMux)
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

	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handleGetChirp)

	mux.HandleFunc("POST /api/chirps", apiCfg.handleCreateChirp)

	mux.HandleFunc("POST /api/users", apiCfg.handleCreateUser)

	mux.HandleFunc("POST /api/login", apiCfg.handleAuthenticateUser)

	app := http.Server{
		Addr:    ":8080",
		Handler: logMux,
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

func middlewareLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%5s @ %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("finished in %v", time.Since(start))
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
	decoder := json.NewDecoder(r.Body)
	var body database.Chirp
	err := decoder.Decode(&body)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body.Body, err = validateChirp(body.Body)
	if err != nil {
		log.Printf("invalid chirp")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	chirp, err := cfg.db.CreateChirp(body)
	if err != nil {
		log.Printf("failed to create chirp: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusCreated, chirp)
	if err != nil {
		log.Printf("failed to respond: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("chirpID")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("failed to convert provided id to integer: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	data, err := cfg.db.GetChirp(id)
	if err == database.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusOK, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func validateChirp(body string) (string, error) {
	if len(body) > 140 {
		return "", errors.New("body is too long")
	}
	result := replaceWords(body, "****", []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	})
	return result, nil
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

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var body database.User
	err := decoder.Decode(&body)
	if err != nil {
		log.Printf("Failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	passEncrypt, err := bcrypt.GenerateFromPassword([]byte(body.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to encrypt password: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body.Pass = string(passEncrypt)
	newUser, err := cfg.db.CreateUser(body)
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusCreated, map[string]any{
		"id":    newUser.Id,
		"email": newUser.Email,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) handleAuthenticateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var body struct {
		Email  string `json:"email"`
		Pass   string `json:"password"`
		Expire int    `json:"expires_in_seconds"`
	}
	err := decoder.Decode(&body)
	if err != nil {
		log.Printf("Failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := cfg.db.GetUserByEmail(body.Email)
	switch err {
	case nil:
	case database.ErrNotFound:
		log.Printf("User does not exist: %s", err)
		w.WriteHeader(http.StatusNotFound)
		return
	default:
		log.Printf("Failed to fetch user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Pass), []byte(body.Pass))
	if err != nil {
		log.Printf("Unable to authenticate user: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token, err := cfg.generateToken(user.Id, time.Second*time.Duration(body.Expire))
	if err != nil {
		log.Printf("Failed to generate jwt: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":    user.Id,
		"email": user.Email,
		"token": token,
	})
	if err != nil {
		w.WriteHeader(500)
	}
}

func (cfg *apiConfig) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	AuthHeader := r.Header.Get("Authorization")
	prefix, tokenString, split := strings.Cut(AuthHeader, " ")
	if !split || prefix != "Bearer" {
		log.Print("malformed authorization header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token, err := jwt.ParseWithClaims(
        tokenString,
        jwt.RegisteredClaims{},
        func(token *jwt.Token) (interface{}, error) {
            return cfg.jwtSecret, nil
    })
    if err != nil {
        log.Printf("jwt invalid or expired: %s", err)
        w.WriteHeader(http.StatusUnauthorized)
        return
    }
    strUserID, _ := token.Claims.GetSubject()
    userId, err := strconv.Atoi(strUserID)
    if err != nil {
        log.Printf("failed to fetch user id: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    // now to handle updating the user in the db package
}

func (cfg *apiConfig) generateToken(id int, exp time.Duration) (string, error) {
	if exp > 24*time.Hour {
		exp = 24 * time.Hour
	}
	nowUTC := time.Now().UTC()
	issueTime := jwt.NewNumericDate(nowUTC)
	expireTime := jwt.NewNumericDate(nowUTC.Add(exp))
	strid := strconv.Itoa(id)
	tokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  issueTime,
		ExpiresAt: expireTime,
		Subject:   strid,
	})
	return tokenString.SignedString(cfg.jwtSecret)
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
