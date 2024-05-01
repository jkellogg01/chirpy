package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jkellogg01/chirpy/internal/database"
)

func (a *ApiState) GetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := a.db.GetChirps()
	if errors.Is(err, database.ErrDBEmpty) || len(chirps) == 0 {
		log.Printf("found no chirps")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		log.Printf("failed to fetch chirps: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"chirps": chirps,
	})
	if err != nil {
		log.Printf("failed to respond: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (a *ApiState) GetChirp(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("chirpID")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("failed to convert provided id to integer: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	data, err := a.db.GetChirp(id)
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

func (a *ApiState) CreateChirp(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token, err := a.validateAccessToken(authHeader)
	if err != nil {
		log.Printf("failed to validate access token: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	authorIdStr, err := token.Claims.GetSubject()
	if err != nil {
		log.Printf("failed to get subject from token: %s", err)
		log.Print("this should never happen with valid chirpy tokens...")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	authorId, err := strconv.Atoi(authorIdStr)
	bodyDecoder := json.NewDecoder(r.Body)
	var body struct {
		Body string
	}
	err = bodyDecoder.Decode(&body)
	if err != nil {
		log.Printf("Failed to decode request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	clean, err := validateChirp(body.Body)
	if err != nil {
		log.Printf("Failed to validate chirp body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	newChirp, err := a.db.CreateChirp(database.Chirp{
		Body:     clean,
		AuthorId: authorId,
	})
	if err != nil {
		log.Printf("Failed to create chirp: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":   newChirp.Id,
        "author_id": newChirp.AuthorId,
		"body": newChirp.Body,
	})
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
