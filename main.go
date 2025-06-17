/*
 * Created on Fri Mar 22 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package main

import (
	"fmt"
	"yoyo-judge/config"
)

func main() {
	cfg := config.NewConfig()
	db, err := config.InitSQL(cfg, "mysql")
	if err != nil {
		fmt.Printf("failed to connect to database: %v", err)
	}
	br := BuildRouterRequest{DB: db, Cfg: cfg}
	BuildRouter(&br)
}
