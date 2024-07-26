/*
 * Created on Fri Jul 26 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package service

import "yoyo-judge/domain/model"

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
