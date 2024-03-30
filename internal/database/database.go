package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

type DB struct {
	file *os.File
	mu   *sync.RWMutex
}

type Data map[string]interface{}

func NewDB(path string) (*DB, error) {
	if !strings.HasSuffix(path, ".json") {
		return nil, fmt.Errorf("got an invalid format for database file: %s", path)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm%0644)
	if errors.Is(err, os.ErrNotExist) {
		log.Print("db does not exist, creating instead")
		file, err = os.Create(path)
		if err != nil {
			log.Printf("failed to create db: %s", err)
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &DB{
		file: file,
		mu:   new(sync.RWMutex),
	}, nil
}

func (db *DB) writeDB(data Data) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	newState, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = db.file.Write(newState)
	return err
}

func (db *DB) readDB() ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return io.ReadAll(db.file)
}

func (db *DB) clearDB() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbFilename := db.file.Name()
	err := db.file.Close()
	if err != nil {
		return err
	}
	newDBFile, err := os.Create(dbFilename)
	if err != nil {
		return err
	}
	db.file = newDBFile
	return nil
}
