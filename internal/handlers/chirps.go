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

func (a *ApiConfig) GetChirps(w http.ResponseWriter, r *http.Request) {
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

func (a *ApiConfig) GetChirp(w http.ResponseWriter, r *http.Request) {
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

func (a *ApiConfig) CreateChirp(w http.ResponseWriter, r *http.Request) {
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

func (a *ApiConfig) DeleteChirp(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("authorization")
    token, err := a.validateAccessToken(authHeader)
    if err != nil {
        log.Printf("access token failed: %s", err)
        w.WriteHeader(http.StatusForbidden)
        return
    }
    userIdStr, err := token.Claims.GetSubject()
    if err != nil {
        log.Printf("couldn't get user id from token claims: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    userId, err := strconv.Atoi(userIdStr)
    if err != nil {
        log.Printf("couldn't convert user id to integer: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    chirpIdStr := r.PathValue("chirpID")
    if chirpIdStr == "" {
        log.Print("no chirp id provided")
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    chirpId, err := strconv.Atoi(chirpIdStr)
    if err != nil {
        log.Printf("couldn't convert chirp id to integer: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    chirp, err := a.db.GetChirp(chirpId)
    switch {
    case err == nil:
        // as you were
    case errors.Is(err, database.ErrDBEmpty):
        log.Print("nice try cowboy, no chirps here")
        w.WriteHeader(http.StatusTeapot)
        return
    case errors.Is(err, database.ErrNotFound):
        log.Printf("you missed: %s", err)
        w.WriteHeader(http.StatusNotFound)
        return
    default:
        log.Printf("hard to say: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    if chirp.AuthorId != userId {
        log.Printf("user %d not authorized to delete this chirp by user %d", userId, chirp.AuthorId)
        w.WriteHeader(http.StatusForbidden)
        return
    }
    err = a.db.DeleteChirp(chirpId)
    if err != nil {
        log.Printf("failed to delete chirp: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
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
