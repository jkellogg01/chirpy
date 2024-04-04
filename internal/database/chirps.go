package database

import (
	"cmp"
	"encoding/json"
	"errors"
	"slices"
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

func (db *DB) CreateChirp(chirp Chirp) (Chirp, error) {
	chirps, err := db.GetChirps()
	if errors.Is(err, ErrDBEmpty) {
		chirps = make([]Chirp, 0)
	} else if err != nil {
		return Chirp{}, err
	}
	newChirp := chirp
	if len(chirps) > 0 {
		maxID := slices.MaxFunc(chirps, func(a, b Chirp) int {
			return cmp.Compare(a.Id, b.Id)
		}).Id
		newChirp.Id = maxID + 1
	} else {
		newChirp.Id = 1
	}
	chirps = append(chirps, newChirp)
	return newChirp, db.writeDB("chirps", chirps)
}

func (db *DB) GetChirp(id int) (Chirp, error) {
	data, err := db.readDB()
	if err != nil {
		return Chirp{}, err
	}
	if len(data) == 0 {
		return Chirp{}, ErrDBEmpty
	}
	jsonData := map[string][]Chirp{
		"chirps": make([]Chirp, 0),
	}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return Chirp{}, err
	}
	for _, chirp := range jsonData["chirps"] {
		if chirp.Id == id {
			return chirp, nil
		}
	}
	return Chirp{}, ErrNotFound
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
