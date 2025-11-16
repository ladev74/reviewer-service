package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	stdlog "log"
	"net/http"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"reviewer-service/internal/config"
	"reviewer-service/internal/logger"
	"reviewer-service/internal/repository/postgres"
	"reviewer-service/internal/server"
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

	pgClient, err := postgres.New(ctx, &cfg.Postgres, log)
	if err != nil {
		log.Fatal("cannot initialize postgres", zap.Error(err))
	}

	router := server.NewRouter(pgClient, log, &cfg.Logger, cfg.HTTP.Timeout)
	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)

	srv := http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Info("starting http server", zap.String("addr", srv.Addr))
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()

	log.Info("received shutdown signal")

	pgClient.Close()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Error("failed to shutdown server", zap.Error(err))
	}

	log.Info("application shutdown completed successfully")
}

func fetchConfigPath() string {
	var path string

	flag.StringVar(&path, "config_path", "", "Path to the config file")
	flag.Parse()

	return path
}
