/*
 * Created on Fri Jul 26 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package service

import (
	"yoyo-judge/domain/model"

	"gorm.io/gorm"
)

type UserFindBy struct {
	Criteria map[string]interface{}
	Asc      bool
	Page     int
	Size     int
}

type User interface {
	FindBy(req UserFindBy) ([]model.User, int, error)
	FindOneBy(criteria map[string]interface{}) (model.User, error)
	Create(user model.User) (model.User, error)
	Update(user model.User) error
	Delete(user model.User) error
}

type user struct {
	DB *gorm.DB
}

func NewUser(db *gorm.DB) User {
	return &user{
		DB: db,
	}
}

func (u *user) FindBy(req UserFindBy) ([]model.User, int, error) {
	return nil, 0, nil
}

func (u *user) FindOneBy(criteria map[string]interface{}) (model.User, error) {
	return model.User{}, nil
}

func (u *user) Create(user model.User) (model.User, error) {
	return model.User{}, nil
}

func (u *user) Update(user model.User) error {
	return nil
}

func (u *user) Delete(user model.User) error {
	return nil
}
