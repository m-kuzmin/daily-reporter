package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram"
)

func main() {
	client := telegram.NewClient("api.telegram.org", mustTelegramToken())
	client.Start(10)
	defer client.Stop()

	waitSigterm()
	log.Println("Recieved ^C (SIGTERM), stopping the bot (Graceful shutdown).")

	go func() {
		waitSigterm()
		log.Println("If you ^C again the server will force stop!")
		waitSigterm()
		log.Println("Server force-stopped.")
		os.Exit(1)
	}()
}

// Looks for the token in CLI args
func mustTelegramToken() string {
	token := flag.String("tg-token", "", "Telegram token for the bot")
	flag.Parse()
	if *token == "" {
		log.Fatal("No token has been provided for the telegram bot, use -tg-token TOKEN to pass it in")
	}
	return *token
}

// Doesn't return until Ctrl+C is pressed in the terminal or SIGTERM is received in another way
func waitSigterm() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
