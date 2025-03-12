package logger

import (
	"log"
	"os"
	"path/filepath"
)

var Logger *log.Logger

func init() {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal(err)
	}

	logFile, err := os.OpenFile(
		filepath.Join(logDir, "anigo.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}

	Logger = log.New(logFile, "", log.LstdFlags)
}
