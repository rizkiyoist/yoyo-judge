/*
 * Created on Mon Jul 29 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config interface {
	Init(prefix string)
	GetString(key string) string
}

type config struct{}

func NewConfig() Config {
	v := &config{}
	v.Init("")
	return v
}

func (c *config) Init(prefix string) {
	viper.SetEnvPrefix("go-clean")
	viper.AutomaticEnv()

	osEnv := os.Getenv("GO_ENV")

	env := "env"
	if osEnv != "" {
		env = osEnv
	}

	if prefix != "" {
		env = prefix + "." + env
	}

	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.SetConfigType("json")
	viper.SetConfigFile(env + ".json")
	err := viper.ReadInConfig()

	if err != nil {
		panic(err)
	}
}

func (c *config) GetString(key string) string {
	return viper.GetString(key)
}
