package main

import (
	"log"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Telegram TelegramConfig `toml:"telegram,omitempty"`
	Logging  LoggingConfig  `toml:"logging,omitempty"`
}

type TelegramConfig struct {
	Token    string `toml:"token,omitempty"`
	Threads  uint   `toml:"threads,omitempty"`
	Template string `toml:"template,omitempty"`
}

type LoggingConfig struct {
	Level string `toml:"level,omitempty"`
}

// Reads the config file from config.toml and returns it. Panics if there are any errors.
func mustNewConfig() Config {
	conf := Config{
		Telegram: TelegramConfig{
			Token:    "",
			Threads:  1,
			Template: "assets/telegram/strings.yaml",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}

	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal(err)
	}

	return conf
}
