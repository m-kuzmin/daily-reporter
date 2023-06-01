package main

import (
	"log"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Telegram struct {
		Token   string
		Threads uint
	}
}

// Reads the config file from config.toml and returns it. Panics if there are any errors.
func mustNewConfig() Config {
	var conf Config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal(err)
	}
	return conf
}
