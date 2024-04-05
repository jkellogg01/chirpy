package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jkellogg01/chirpy/internal/database"
	"github.com/jkellogg01/chirpy/internal/middleware"
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
		db:             db,
		jwtSecret:      secret,
	}

    metrics := &middleware.ApiMetrics{}

	mux := http.NewServeMux()
	corsMux := middlewareCors(mux)
	logMux := middlewareLogging(corsMux)
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

	mux.HandleFunc("GET /api/chirps", apiCfg.handleGetChirps)

	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handleGetChirp)

	mux.HandleFunc("POST /api/chirps", apiCfg.handleCreateChirp)

	mux.HandleFunc("POST /api/users", apiCfg.handleCreateUser)

	mux.HandleFunc("POST /api/login", apiCfg.handleAuthenticateUser)

	mux.HandleFunc("PUT /api/users", apiCfg.handleUpdateUser)

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

func middlewareLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%7s @ %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("finished in %v", time.Since(start))
	})
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
	expTime := time.Duration(body.Expire) * time.Second
	if body.Expire == 0 {
		expTime = time.Hour * 24
	}
	token := generateToken(user.Id, expTime)
	tokenString, err := token.SignedString(cfg.jwtSecret)
	if err != nil {
		log.Printf("Failed to sign jwt: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("responding with token string: %s", tokenString)
	err = respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":    user.Id,
		"email": user.Email,
		"token": tokenString,
	})
	if err != nil {
		w.WriteHeader(500)
	}
}

func (cfg *apiConfig) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	AuthHeader := r.Header.Get("Authorization")
	log.Printf("auth header: %s", AuthHeader)
	tokenString, split := strings.CutPrefix(AuthHeader, "Bearer ")
	if !split {
		log.Print("malformed authorization header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return cfg.jwtSecret, nil
		},
	)
	switch {
	case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
		log.Printf("timing is everything: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case errors.Is(err, jwt.ErrTokenMalformed):
		log.Print("token is malformed")
		w.WriteHeader(http.StatusUnauthorized)
		return
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		log.Printf("token signature is invalid: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err != nil:
		log.Printf("something else entirely")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	strUserID, _ := token.Claims.GetSubject()
	userId, err := strconv.Atoi(strUserID)
	if err != nil {
		log.Printf("failed to fetch user id: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var body struct {
		Email string `json:"email"`
		Pass  string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&body)
	if err != nil {
		log.Printf("failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	passEncrypted, err := bcrypt.GenerateFromPassword([]byte(body.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to encrypt password: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := cfg.db.UpdateUser(database.User{
		Id:    userId,
		Email: body.Email,
		Pass:  string(passEncrypted),
	})
	if err != nil {
		log.Printf("failed to update user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"email": user.Email,
		"id":    user.Id,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func generateToken(id int, exp time.Duration) *jwt.Token {
	if exp > 24*time.Hour {
		exp = 24 * time.Hour
	}
	nowUTC := time.Now().UTC()
	issueTime := jwt.NewNumericDate(nowUTC)
	expireTime := jwt.NewNumericDate(nowUTC.Add(exp))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  issueTime,
		ExpiresAt: expireTime,
		Subject:   strconv.Itoa(id),
	})
	return token
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
