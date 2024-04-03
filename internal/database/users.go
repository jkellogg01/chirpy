package database

import (
	"cmp"
	"encoding/json"
	"slices"
)

type User struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
}

func (db *DB) CreateUser(email string) (User, error) {
	data, err := db.readDB()
	if err != nil {
		return User{}, err
	}
	users := make([]User, 0)
	newUser := User{Id: 1, Email: email}
	if len(data) > 0 {
		jsonData := map[string][]User{
			"users": make([]User, 0),
		}
		err = json.Unmarshal(data, &jsonData)
		if err != nil {
			return User{}, err
		}
		users = jsonData["users"]
		maxId := slices.MaxFunc(users, func(a, b User) int {
			return cmp.Compare(a.Id, b.Id)
		}).Id
		newUser.Id = maxId + 1
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
