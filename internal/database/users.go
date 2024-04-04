package database

import (
	"cmp"
	"encoding/json"
	"math"
	"slices"
)

var (
    ErrUserExist = errors.New("user already exists")
)

type User struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
	Pass  string `json:"password"`
}

func (db *DB) CreateUser(user User) (User, error) {
	data, err := db.readDB()
	if err != nil {
		return User{}, err
	}
	users := make([]User, 0)
	newUser := user
	if len(data) > 0 {
		jsonData := map[string][]User{
			"users": make([]User, 0),
		}
		err = json.Unmarshal(data, &jsonData)
		if err != nil {
			return User{}, err
		}
		users = jsonData["users"]
        maxId := math.MinInt
        for _, user := range users {
            if user.Email == newUser.Email {
                return User{}, ErrUserExist
            }
            if user.Id > maxId {
                maxId = user.Id
            }
        }
	} else {
		newUser.Id = 1
	}
	users = append(users, newUser)
	err = db.writeDB(Data{
		"users": users,
	})
	if err != nil {
		return User{}, err
	}
	return newUser, nil
}
