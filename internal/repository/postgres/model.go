package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Config struct {
	Host        string        `env:"POSTGRES_HOST" env-required:"true"`
	Port        string        `env:"POSTGRES_PORT" env-required:"true"`
	User        string        `env:"POSTGRES_USER" env-required:"true"`
	Password    string        `env:"POSTGRES_PASSWORD" env-required:"true"`
	Database    string        `env:"POSTGRES_DATABASE" env-required:"true"`
	Timeout     time.Duration `env:"POSTGRES_TIMEOUT" env-required:"true"`
	MaxRetries  int           `env:"POSTGRES_MAX_RETRIES" env-required:"true"`
	BaseBackoff time.Duration `env:"POSTGRES_BASE_BACKOFF" env-required:"true"`
	MaxConns    int           `env:"POSTGRES_MAX_CONNECTIONS" env-required:"true"`
	MinConns    int           `env:"POSTGRES_MIN_CONNECTIONS" env-required:"true"`
}

type Client struct {
	pool        *pgxpool.Pool
	logger      *zap.Logger
	timeout     time.Duration
	retryConfig retryConfig
}

type retryConfig struct {
	maxRetries  int
	baseBackoff time.Duration
}
