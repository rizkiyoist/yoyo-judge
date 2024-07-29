/*
 * Created on Mon Jul 29 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package main

import "yoyo-judge/config"

var env config.Config

func startServer() {
	sqlConn, err := config.InitSQL(env, "mysql")
	if err != nil {
		panic(err)
	}

}
