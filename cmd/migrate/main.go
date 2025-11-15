package main

import (
	"errors"
	"flag"
	"fmt"
	stdlog "log"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"go.uber.org/zap"

	"reviewer-service/internal/config"
	"reviewer-service/internal/logger"
)

func main() {
	var configPath, migrationPath string

	flag.StringVar(&configPath, "config_path", "", "Path to the config file")
	flag.StringVar(&migrationPath, "migration_path", "", "Path to the migration file")
	flag.Parse()

	cfg, err := config.New(configPath)
	if err != nil {
		stdlog.Fatal(err)
	}

	log, err := logger.New(&cfg.Logger)
	if err != nil {
		stdlog.Fatal(err)
	}

	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.Database,
	)

	migration, err := migrate.New("file://"+migrationPath, url)
	if err != nil {
		log.Fatal("failed to create migration", zap.Error(err))
	}

	err = migration.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal("failed to run migration", zap.Error(err))
	}

	log.Info("successfully migrated")
}
