/*
 * Created on Fri Jul 26 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package model

type UserSocial struct {
	ID          uint   `json:"id"`
	UserID      uint   `json:"users_id"`
	SocialType  string `json:"social_type"`
	SocialToken string `json:"social_token"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (u *UserSocial) TableName() string {
	return "user_socials"
}
