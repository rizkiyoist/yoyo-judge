/*
 * Created on Mon Jul 29 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package config

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitSQL(env Config, key string) (db *gorm.DB, err error) {
	dbUser := env.GetString(key + ".user")
	dbPass := env.GetString(key + ".pass")
	dbHost := env.GetString(key + ".host")
	dbPort := env.GetString(key + ".port")
	dbName := env.GetString(key + ".name")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed connecting to database")
	}

	return
}
