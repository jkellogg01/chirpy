package database

import (
	"cmp"
	"encoding/json"
	"errors"
	"log"
	"slices"
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	chirps, err := db.GetChirps()
	if errors.Is(err, ErrDBEmpty) {
		log.Println("hey josh dont forget to handle the case in the create chirp method where the database is empty. love u bye")
	} else if err != nil {
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
	return newChirp, db.writeDB(data)
}

func (db *DB) GetChirps() ([]Chirp, error) {
	data, err := db.readDB()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrDBEmpty
	}
	jsonData := map[string][]Chirp{
		"chirps": make([]Chirp, 0),
	}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return nil, err
	}
	return jsonData["chirps"], nil
}
