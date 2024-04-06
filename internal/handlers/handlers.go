package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/jkellogg01/chirpy/internal/database"
)

type ApiState struct {
	db        *database.DB
	jwtSecret []byte
}

func NewApiState(secret, dbPath string) (*ApiState, error) {
	jwtSecret, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, err
	}
	db, err := database.NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &ApiState{
		db:        db,
		jwtSecret: jwtSecret,
	}, nil
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
