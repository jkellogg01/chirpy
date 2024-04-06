package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/jkellogg01/chirpy/internal/database"
)

func (a *ApiState) GetChirps(w http.ResponseWriter, r *http.Request) {
    chirps, err := a.db.GetChirps()
    if errors.Is(err, database.ErrDBEmpty) || len(chirps) == 0 {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if err != nil {
        log.Printf("failed to fetch chirps: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}
