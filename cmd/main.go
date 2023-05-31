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
	waitForCtrlC()
	log.Println("Recieved ^C, stopping the bot (Graceful shutdown).")
	go func() {
		waitForCtrlC()
		log.Println("If you ^C again the server will force stop!")
		waitForCtrlC()
		log.Println("Server force-stopped.")
		os.Exit(1)
	}()
	client.Stop()
}

func mustTelegramToken() string {
	token := flag.String("tg-token", "", "Telegram token for the bot")
	flag.Parse()
	if *token == "" {
		log.Fatal("No token has been provided for the telegram bot, use -tg-token TOKEN to pass it in")
	}
	return *token
}

func waitForCtrlC() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
