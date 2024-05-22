package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jkellogg01/chirpy/internal/database"
)

type ApiConfig struct {
	db   *database.DB
	keys map[string][]byte
}

func NewApiConfig(dbPath string, strKeys map[string]string) (*ApiConfig, error) {
	keys := make(map[string][]byte)
	for k, v := range strKeys {
        if v == "" {
            return nil, fmt.Errorf("expected key %s was left empty!", k)
        }
		key, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("encountered error %s on value %s", err, v)
		}
        if os.Getenv("ENV") == "DEV" {
            log.Printf("added api secret: %v", k)
        }
		keys[k] = key
	}
	db, err := database.NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &ApiConfig{
		db:   db,
		keys: keys,
	}, nil
}

func (a *ApiConfig) ClearDB() error {
	return a.db.ClearDB()
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
