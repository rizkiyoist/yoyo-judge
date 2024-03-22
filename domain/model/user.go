/*
 * Created on Fri Jul 26 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package model

type User struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (u *User) TableName() string {
	return "users"
}
