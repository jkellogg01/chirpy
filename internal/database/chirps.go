package database

import (
	"cmp"
	"encoding/json"
	"slices"
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	chirps, err := GetChirps()
	if err != nil {
		return Chirp{}, err
	}
	maxID := slices.MaxFunc(chirps, func(a, b Chirp) int {
		return cmp.Compare(a.Id, b.Id)
	}).Id
	newChirp := Chirp{
		Id:   maxID + 1,
		Body: body,
	}
	chirps = append(chirps, newChirp)
	data := Data{
		"chirps": chirps,
	}
	newDBState, err := json.Marshal(data)
	if err != nil {
		return newChirp, err
	}
	return newChirp, db.writeDB(newDBState)
}

func (db *DB) GetChirps() ([]Chirp, error)
