package main

import (
	"encoding/json"
)

type User struct {
	Mail				string	`json:"mail"`
	Username			string	`json:"username"`
	HashedPassword		string	`json:"password"`
}

func (u *User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, u)
}

