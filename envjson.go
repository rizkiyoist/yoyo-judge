/*
 * Created on Wed Sep 02 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package main

import (
	"encoding/json"
	"log"
	"os"
)

// envJSONConfig mirrors env.json's shape. Only the Google OAuth
// credentials live here — env.json used to also carry a "mysql" section
// for the superseded config/ package, removed since nothing reads it.
type envJSONConfig struct {
	Google struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURL  string `json:"redirect_url"`
	} `json:"google"`
}

// loadEnvJSON reads env.json (if present, next to the binary/working dir)
// and populates GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_REDIRECT_URL
// from it — but only for whichever of those isn't already set in the real
// environment, so a real deploy's env vars always take priority and local
// dev can just fill in env.json instead of remembering to export vars each
// time. env.json is optional; a missing or unparsable file is not fatal
// (server/oauth.go already degrades gracefully to "Google login disabled"
// when the credentials are empty).
func loadEnvJSON() {
	data, err := os.ReadFile("env.json")
	if err != nil {
		return
	}
	var cfg envJSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("warning: failed to parse env.json: %v", err)
		return
	}
	setEnvIfUnset("GOOGLE_CLIENT_ID", cfg.Google.ClientID)
	setEnvIfUnset("GOOGLE_CLIENT_SECRET", cfg.Google.ClientSecret)
	setEnvIfUnset("GOOGLE_REDIRECT_URL", cfg.Google.RedirectURL)
}

func setEnvIfUnset(key, value string) {
	if value != "" && os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}
