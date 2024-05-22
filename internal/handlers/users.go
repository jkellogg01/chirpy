package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jkellogg01/chirpy/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func (a *ApiConfig) CreateUser(w http.ResponseWriter, r *http.Request) {
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
	newUser, err := a.db.CreateUser(body)
	if err != nil {
		log.Printf("Failed to create user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusCreated, map[string]any{
		"id":            newUser.Id,
		"email":         newUser.Email,
		"is_chirpy_red": newUser.IsChirpyRed,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// api/login
func (a *ApiConfig) AuthenticateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var body struct {
		Email string `json:"email"`
		Pass  string `json:"password"`
	}
	err := decoder.Decode(&body)
	if err != nil {
		log.Printf("Failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := a.db.GetUserByEmail(body.Email)
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
	accessToken := generateAccessToken(user.Id)
	refreshToken := generateRefreshToken(user.Id)
	accessTokenString, err := accessToken.SignedString(a.keys["jwt-secret"])
	if err != nil {
		log.Printf("Failed to sign jwt: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	refreshTokenString, err := refreshToken.SignedString(a.keys["jwt-secret"])
	if err != nil {
		log.Printf("Failed to sign jwt (refresh): %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("responding with token string: %s", accessTokenString)
	err = respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":            user.Id,
		"email":         user.Email,
        "is_chirpy_red": user.IsChirpyRed,
		"token":         accessTokenString,
		"refresh_token": refreshTokenString,
	})
	if err != nil {
		w.WriteHeader(500)
	}
}

func (a *ApiConfig) UpdateUser(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token, err := a.validateAccessToken(authHeader)
	switch {
	case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
		log.Printf("timing is everything: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case errors.Is(err, jwt.ErrTokenMalformed):
		log.Printf("token is malformed: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		log.Printf("token signature is invalid: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case errors.Is(err, ErrIssuerInvalid):
		log.Print("invalid token issuer; this may be a refresh token or it may have come from a different site.")
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err != nil:
		log.Printf("something else entirely: %s", err)
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
	user, err := a.db.UpdateUser(database.User{
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

func (a *ApiConfig) RefreshUser(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token, err := a.validateRefreshToken(authHeader)
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
	case errors.Is(err, ErrIssuerInvalid):
		log.Print("invalid token issuer")
		w.WriteHeader(http.StatusUnauthorized)
		return
	case errors.Is(err, ErrTokenRevoked):
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err != nil:
		log.Printf("something else entirely: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	userID, err := token.Claims.GetSubject()
	if err != nil {
		log.Printf("failed to fetch jwt subject: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	idstr, err := strconv.Atoi(userID)
	if err != nil {
		log.Printf("failed to convert user id to int: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	newToken := generateAccessToken(idstr)
	tokenString, err := newToken.SignedString(a.keys["jwt-secret"])
	if err != nil {
		log.Printf("failed to write token string")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusOK, map[string]any{
		"token": tokenString,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (a *ApiConfig) RevokeToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	_, err := a.validateRefreshToken(authHeader)
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
	case errors.Is(err, ErrIssuerInvalid):
		log.Print("invalid token issuer")
		w.WriteHeader(http.StatusUnauthorized)
		return
	case errors.Is(err, ErrTokenRevoked):
		log.Print(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err != nil:
		log.Printf("something else entirely: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tokenString, _ := strings.CutPrefix(authHeader, "Bearer ")
	revoked, err := a.db.Revoke(tokenString)
	if err != nil {
		log.Printf("failed to revoke token: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("token revoked at %v", revoked.RevokedAt)
	w.WriteHeader(http.StatusOK)
}

var (
	ErrMalformedAuthHeader = errors.New("malformed authorization header")
	ErrTokenRevoked        = errors.New("this token has been revoked")
	ErrIssuerInvalid       = errors.New("this is not a chirpy access token")
)

func generateAccessToken(id int) *jwt.Token {
	exp := 1 * time.Hour
	nowUTC := time.Now().UTC()
	issueTime := jwt.NewNumericDate(nowUTC)
	expireTime := jwt.NewNumericDate(nowUTC.Add(exp))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  issueTime,
		ExpiresAt: expireTime,
		Subject:   strconv.Itoa(id),
	})
	return token
}

func (a *ApiConfig) validateAccessToken(authHeader string) (*jwt.Token, error) {
	tokenString, split := strings.CutPrefix(authHeader, "Bearer ")
	if !split {
		return nil, ErrMalformedAuthHeader
	}
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return a.keys["jwt-secret"], nil
		},
	)
	if err != nil {
		return nil, err
	}
	i, err := token.Claims.GetIssuer()
	if err != nil {
		return nil, err
	}
	if i != "chirpy-access" {
		return nil, ErrIssuerInvalid
	}
	return token, nil
}

func generateRefreshToken(id int) *jwt.Token {
	exp := 60 * 24 * time.Hour
	nowUTC := time.Now().UTC()
	issueTime := jwt.NewNumericDate(nowUTC)
	expireTime := jwt.NewNumericDate(nowUTC.Add(exp))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-refresh",
		IssuedAt:  issueTime,
		ExpiresAt: expireTime,
		Subject:   strconv.Itoa(id),
	})
	return token
}

func (a *ApiConfig) validateRefreshToken(authHeader string) (*jwt.Token, error) {
	tokenString, split := strings.CutPrefix(authHeader, "Bearer ")
	if !split {
		return nil, errors.New("malformed authorization header")
	}
	if r, _ := a.db.IsRevoked(tokenString); r {
		return nil, ErrTokenRevoked
	}
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return a.keys["jwt-secret"], nil
		},
	)
	if err != nil {
		return nil, err
	}
	i, err := token.Claims.GetIssuer()
	if err != nil {
		return nil, err
	}
	if i != "chirpy-refresh" {
		return nil, ErrIssuerInvalid
	}
	return token, nil
}
