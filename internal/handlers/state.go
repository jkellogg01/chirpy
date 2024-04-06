package handlers

import (
	"encoding/base64"

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
