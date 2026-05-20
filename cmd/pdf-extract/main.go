package main

import (
	"os"

	"github.com/leshchenko/pdf-extract/internal/config"
	"github.com/leshchenko/pdf-extract/internal/httpserver"
	"github.com/leshchenko/pdf-extract/internal/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = os.Stderr.WriteString("config: " + err.Error() + "\n")
		os.Exit(1)
	}

	level, err := logging.ParseLevel(cfg.LogLevel)
	if err != nil {
		_, _ = os.Stderr.WriteString("config: " + err.Error() + "\n")
		os.Exit(1)
	}
	log := logging.New(level)
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Error("mkdir uploads", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		log.Error("mkdir outputs", "err", err)
		os.Exit(1)
	}

	if _, _, err := httpserver.Run(cfg, log); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}
