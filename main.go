/*
 * Created on Fri Mar 22 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package main

import (
	"yoyo-judge/server"
)

func main() {
	loadEnvJSON()
	db := server.OpenDB()
	server.SeedIfEmpty(db)
	store := server.NewStore(db)
	br := BuildRouterRequest{Store: store}
	BuildRouter(&br)
}
