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

func (a *ApiState) CreateUser(w http.ResponseWriter, r *http.Request) {
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
		"id":    newUser.Id,
		"email": newUser.Email,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (a *ApiState) AuthenticateUser(w http.ResponseWriter, r *http.Request) {
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
	expTime := time.Duration(body.Expire) * time.Second
	if body.Expire == 0 {
		expTime = time.Hour * 24
	}
	token := generateToken(user.Id, expTime)
	tokenString, err := token.SignedString(a.jwtSecret)
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

func (a *ApiState) UpdateUser(w http.ResponseWriter, r *http.Request) {
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
			return a.jwtSecret, nil
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
