package database

import (
	"encoding/json"
	"time"
)

type RevokedToken struct {
	Id        string
	RevokedAt time.Time
}

func (db *DB) GetRevokedTokens() ([]RevokedToken, error) {
	data, err := db.readDB()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []RevokedToken{}, nil
	}
	var result struct {
		Tokens []RevokedToken `json:"tokens"`
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result.Tokens, nil
}
    
