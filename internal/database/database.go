package database

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
)

type DB struct {
	path string
	mu   *sync.RWMutex
}

type Data map[string]interface{}

var (
	ErrDBEmpty = errors.New("db is empty")
)

func NewDB(path string) (*DB, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		file, err = os.Create(path)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	defer file.Close()
	return &DB{
		path: path,
		mu:   new(sync.RWMutex),
	}, nil
}

func (db *DB) ClearDB() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	file, err := os.Create(db.path)
	if err != nil {
		return err
	}
	return file.Close()
}

func (db *DB) writeDB(data Data) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	newState, err := json.Marshal(data)
	if err != nil {
		return err
	}
	file, err := os.Create(db.path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(newState)
	return err
}

func (db *DB) readDB() ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	data, err := os.ReadFile(db.path)
	if err != nil {
		return nil, err
	}
	log.Printf("read %d bytes from db", len(data))
	return data, nil
}
