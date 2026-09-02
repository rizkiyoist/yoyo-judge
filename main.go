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
	// server.SeedIfEmpty(db) — disabled: all contests/judges/players should
	// come from real usage now, not auto-injected demo data. Left in
	// server/db.go (unused, not deleted) in case demo data is needed again
	// later — e.g. for local dev on a fresh database.
	store := server.NewStore(db)
	br := BuildRouterRequest{Store: store}
	BuildRouter(&br)
}
