package database

import (
	"encoding/json"
	"time"
)

type RevokedToken struct {
	Id        string
	RevokedAt time.Time
}

func (db *DB) Revoke(token string) (RevokedToken, error) {
	revoked, err := db.GetRevokedTokens()
	if err != nil {
		return RevokedToken{}, err
	}
	toRevoke := RevokedToken{
		Id:        token,
		RevokedAt: time.Now(),
	}
	revoked = append(revoked, toRevoke)
    db.writeDB("tokens", revoked)
    return toRevoke, nil
}

func (db *DB) IsRevoked(token string) (bool, error) {
    revoked, err := db.GetRevokedTokens()
    if err != nil {
        return false, err
    }
    for _, tkn := range revoked {
        if tkn.Id == token {
            return true, nil
        }
    }
    return false, nil
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
