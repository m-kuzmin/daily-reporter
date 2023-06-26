package main

import (
	"log"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Telegram TelegramConfig `toml:"telegram,omitempty"`
}

type TelegramConfig struct {
	Token    string `toml:"token,omitempty"`
	Threads  uint   `toml:"threads,omitempty"`
	Template string `toml:"template,omitempty"`
}

// Reads the config file from config.toml and returns it. Panics if there are any errors.
func mustNewConfig() Config {
	conf := Config{
		Telegram: TelegramConfig{
			Token:    "",
			Threads:  1,
			Template: "assets/telegram/strings.yaml",
		},
	}

	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal(err)
	}

	return conf
}
