package main

import "fmt"

type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (u User) String() string {
	return fmt.Sprintf("id: %d, name: %s, address: %s", u.ID, u.Name, u.Address)
}
