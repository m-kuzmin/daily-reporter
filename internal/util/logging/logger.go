// Abstracts over some underlying logger
package logging

import (
	"fmt"
	"log"
)

//nolint:gochecknoglobals,golint // Global log level of the application
var LogLevel = LogLevelInfo

type logLevel int

const (
	LogLevelTrace logLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelError
	LogLevelFatal
)

type Loggable interface {
	Log() string
}

func Tracef(fmtStr string, v ...any) {
	if LogLevel <= LogLevelTrace {
		log.Printf(fmt.Sprintf("TRACE   : %s\n", fmtStr), v...) //nolint:forbidigo // Allowed here only
	}
}

func Debugf(fmtStr string, v ...any) {
	if LogLevel <= LogLevelDebug {
		log.Printf(fmt.Sprintf("DEBUG   : %s\n", fmtStr), v...) //nolint:forbidigo // Allowed here only
	}
}

func Infof(fmtStr string, v ...any) {
	if LogLevel <= LogLevelInfo {
		log.Printf(fmt.Sprintf("INFO    : %s\n", fmtStr), v...) //nolint:forbidigo // Allowed here only
	}
}

func Errorf(fmtStr string, v ...any) {
	if LogLevel <= LogLevelError {
		log.Printf(fmt.Sprintf("ERROR   : %s\n", fmtStr), v...) //nolint:forbidigo // Allowed here only
	}
}

func Fatalf(fmtStr string, v ...any) {
	if LogLevel <= LogLevelFatal {
		log.Fatalf(fmt.Sprintf("FATAL   : %s\n", fmtStr), v...) //nolint:forbidigo // Allowed here only
	}
}
