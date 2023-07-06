package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram"
	"github.com/m-kuzmin/daily-reporter/internal/template"
)

func main() {
	conf := mustNewConfig()
	if conf.Telegram.Token == "" {
		log.Fatal("No telegram token in config.toml, exiting.")
	}

	templ, err := template.LoadYAMLTemplate(conf.Telegram.Template)
	if err != nil {
		log.Fatalf("while loading yaml template from %s: %s", conf.Telegram.Template, err)
	}

	client := telegram.NewClient("api.telegram.org", conf.Telegram.Token, templ)

	fail := client.Start(conf.Telegram.Threads)

	ctrlC := make(chan os.Signal, 1)
	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-fail:
		log.Printf("Bot crashed with error: %s", err)

		os.Exit(1)
	case <-ctrlC:
		log.Println("Received ^C (SIGTERM), stopping the bot (Graceful shutdown).")
		client.Stop()
	}
}
