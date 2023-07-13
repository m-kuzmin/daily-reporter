package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/state"
	"github.com/m-kuzmin/daily-reporter/internal/template"
	"github.com/m-kuzmin/daily-reporter/internal/util/logging"
)

func main() {
	conf := mustNewConfig()

	setupLogger(conf.Logging.Level)

	client := setupTgClient(conf.Telegram.Token, conf.Telegram.Template)
	fail := client.Start(conf.Telegram.Threads)

	ctrlC := make(chan os.Signal, 1)
	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-fail:
		logging.Fatalf("Bot crashed with error: %s", err)
	case <-ctrlC:
		logging.Infof("Received ^C (SIGTERM), stopping the bot (Graceful shutdown).")
		client.Stop()
	}
}

func setupLogger(level string) {
	switch strings.ToLower(level) {
	case "trace":
		logging.LogLevel = logging.LogLevelTrace
	case "debug":
		logging.LogLevel = logging.LogLevelDebug
	case "info":
		logging.LogLevel = logging.LogLevelInfo
	case "error":
		logging.LogLevel = logging.LogLevelError
	case "fatal":
		logging.LogLevel = logging.LogLevelFatal
	}
}

func setupTgClient(token, templateFile string) telegram.Client {
	if token == "" {
		logging.Fatalf("No telegram token in config.toml, exiting.")
	}

	templ, err := template.LoadYAMLTemplate(templateFile)
	if err != nil {
		logging.Fatalf("While loading yaml template from %s: %s", templateFile, err)
	}

	var responses state.Responses
	if err = templ.Populate(&responses); err != nil {
		logging.Fatalf("While populating state.Responses: %s", err)
	}

	return telegram.NewClient("api.telegram.org", token, responses)
}
