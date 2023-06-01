package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram"
)

func main() {
	conf := mustNewConfig()

	client := telegram.NewClient("api.telegram.org", conf.Telegram.Token)
	client.Start(conf.Telegram.Threads)
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

// Doesn't return until Ctrl+C is pressed in the terminal or SIGTERM is received in another way
func waitSigterm() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
