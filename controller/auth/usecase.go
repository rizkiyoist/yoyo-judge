/*
 * Created on Fri Jul 26 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package auth

import "gorm.io/gorm"

type Usecase interface {
	LoginGoogle()
}

type usecase struct {
	db gorm.DB
}

func NewUsecase(db gorm.DB) Usecase {
	return &usecase{
		db: db,
	}
}
