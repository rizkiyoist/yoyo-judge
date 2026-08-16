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
	// NOTE: no MySQL/GORM connection yet — the ScoringApi backend (server/)
	// is in-memory for now. See docs/PROGRESS.md for why and what's next.
	store := server.NewStore()
	br := BuildRouterRequest{Store: store}
	BuildRouter(&br)
}
