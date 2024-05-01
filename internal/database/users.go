package database

import (
	"encoding/json"
	"errors"
	"math"
)

var (
	ErrUserExist = errors.New("user already exists")
)

type User struct {
	Id          int    `json:"id"`
	Email       string `json:"email"`
	Pass        string `json:"password"`
	IsChirpyRed bool   `json:"is_chirpy_red"`
}

func (db *DB) CreateUser(user User) (User, error) {
    user.IsChirpyRed = true
	users, err := db.getUsers()
	if err != nil {
		return User{}, err
	}
	if len(users) < 1 {
		user.Id = 1
	} else {
		maxId := math.MinInt
		for _, usr := range users {
			if usr.Id > maxId {
				maxId = usr.Id
			}
		}
		user.Id = maxId + 1
	}
	users = append(users, user)
	err = db.writeDB("users", users)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (db *DB) GetUserByEmail(email string) (User, error) {
	users, err := db.getUsers()
	if err != nil {
		return User{}, err
	}
	if len(users) == 0 {
		return User{}, ErrNotFound
	}
	for _, user := range users {
		if user.Email == email {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

func (db *DB) GetUser(id int) (User, error) {
	users, err := db.getUsers()
	if err != nil {
		return User{}, err
	}
	if len(users) == 0 {
		return User{}, ErrNotFound
	}
	for _, user := range users {
		if user.Id == id {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

func (db *DB) UpdateUser(newUser User) (User, error) {
	if newUser.Email == "" || newUser.Pass == "" {
		return User{}, errors.New("fill all fields to update user")
	}
	users, err := db.getUsers()
	if err != nil {
		return User{}, err
	}
	if len(users) == 0 {
		return User{}, ErrNotFound
	}
	for i, user := range users {
		if user.Id != newUser.Id {
			continue
		}
		users[i] = newUser
		err = db.writeDB("users", users)
		return newUser, err
	}
	return User{}, ErrNotFound
}

func (db *DB) UpgradeUser(id int) (User, error) {
    users, err := db.getUsers()
    if err != nil {
        return User{}, err
    }
    for i, user := range users {
        if user.Id != id {
            continue
        }
        user.IsChirpyRed = true
        users[i] = user
        return user, db.writeDB("users", users)
    }
    return User{}, ErrNotFound
}

func (db *DB) getUsers() ([]User, error) {
	data, err := db.readDB()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []User{}, nil
	}
	var result struct {
		Users []User `json:"users"`
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result.Users, nil
}
