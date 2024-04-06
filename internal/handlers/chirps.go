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
