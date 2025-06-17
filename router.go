/*
 * Created on Wed Mar 19 2025
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package main

import (
	"fmt"
	"net/http"
	"yoyo-judge/config"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gorm.io/gorm"
)

type Router interface {
}

type router struct {
	// routes   map[string]map[string]demo.Handler
	handlers map[string]map[string]interface{}
	// middlewares []http.HandlerFunc
}

type BuildRouterRequest struct {
	DB  *gorm.DB
	Cfg config.Config
}

func NewRouter() Router {
	r := &router{
		// routes: make(map[string]map[string]demo.Handler),
	}
	return r
}

func BuildRouter(br *BuildRouterRequest) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// { // route group demo
	// 	router.HandleFunc("/file-spaces", demo.NewHandler().ListFileSpaces).Methods("GET")
	// }

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://app.mimo.id", "https://app-staging.mimo.id"}, // Allow frontend URL
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	err := http.ListenAndServe(":5000", corsHandler)
	if err != nil {
		fmt.Println("failed to start server:", err)
	}
}
