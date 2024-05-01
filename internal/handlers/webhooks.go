package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// I have a strong suspicion that I would want to implement eventData as an 
// interface of some kind, but we'll jump off that bridge when we get there

type PolkaUserEvent struct {
	Event string        `json:"event"`
	Data  struct {
        UserId int `json:"user_id"`
    } `json:"data"`
}

func (a *ApiConfig) DispatchPolkaEvent(w http.ResponseWriter, r *http.Request) {
	bodyDecoder := json.NewDecoder(r.Body)
	var event PolkaUserEvent
	err := bodyDecoder.Decode(&event)
	if err != nil {
		log.Print("failed to decode request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch event.Event {
	case "user.upgraded":
		err = a.handleUserUpgraded(event.Data.UserId)
        if err != nil {
            // this will need later to specify a not found error, ignoring that for now
            log.Printf("failed to upgrade user: %s", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
	default:
		log.Printf("unhandled polka event: %s", event.Event)
		w.WriteHeader(200)
	}
}

func (a *ApiConfig) handleUserUpgraded(id int) error
