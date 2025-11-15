package main

import (
	"context"
	"flag"
	stdlog "log"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"reviewer-service/internal/config"
	"reviewer-service/internal/logger"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	configPath := fetchConfigPath()
	if configPath == "" {
		stdlog.Fatal("config path must specify")
	}

	cfg, err := config.New(configPath)
	if err != nil {
		stdlog.Fatalf("cannot initialize config: %v", err)
	}

	log, err := logger.New(&cfg.Logger)
	if err != nil {
		stdlog.Fatalf("cannot initialize logger: %v", err)
	}
	defer log.Sync()

	log.Info("hello", zap.Any("cfg", cfg))

	<-ctx.Done()
	log.Info("received shutdown signal")

	log.Info("application shutdown completed successfully")
}

func fetchConfigPath() string {
	var path string

	flag.StringVar(&path, "config_path", "", "Path to the config file")
	flag.Parse()

	return path
}
