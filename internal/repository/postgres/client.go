package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"reviewer-service/internal/api"
)

var ErrTeamAlreadyExists = errors.New("team already exists")

func New(ctx context.Context, config *Config, logger *zap.Logger) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	dsn := buildDSN(config)

	retryCfg := retryConfig{
		maxRetries:  config.MaxRetries,
		baseBackoff: config.BaseBackoff,
	}

	pool, err := withRetry(ctx, retryCfg, logger, func() (*pgxpool.Pool, error) {
		pool, err := pgxpool.New(ctx, dsn)
		return pool, err

	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Client{
		pool:        pool,
		logger:      logger,
		timeout:     config.Timeout,
		retryConfig: retryCfg,
	}, nil
}

func (c *Client) SaveTeam(ctx context.Context, team *api.Team) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	exists, err := c.teamExists(ctx, team.TeamName)
	if err != nil {
		return err
	}

	if exists {
		c.logger.Warn(ErrTeamAlreadyExists.Error(), zap.Any("team_name", team.TeamName))
		return fmt.Errorf("%w: %s", ErrTeamAlreadyExists, team.TeamName)
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		c.logger.Error("failed to start transaction", zap.Error(err))
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := withRetry(ctx, c.retryConfig, c.logger, func() (pgconn.CommandTag, error) {
		tag, err := tx.Exec(ctx, querySetTeamName, team.TeamName)
		return tag, err
	})
	if err != nil {
		c.logger.Error("failed to set team name", zap.Error(err), zap.String("team_name", team.TeamName))
		return fmt.Errorf("failed to set team name: %s: %w", team.TeamName, err)
	}

	if tag.RowsAffected() == 0 {
		c.logger.Error("failed to set team name: no rows affected", zap.String("team_name", team.TeamName))
		return fmt.Errorf("failed to set team name: no rows affected: %s", team.TeamName)
	}

	for _, member := range team.Members {
		tag, err = withRetry(ctx, c.retryConfig, c.logger, func() (pgconn.CommandTag, error) {
			tag, err = tx.Exec(ctx, querySaveTeamMember, member.UserID, member.UserName, team.TeamName, member.IsActive)
			return tag, err
		})
		if err != nil {
			c.logger.Error("failed to save team member", zap.Error(err), zap.String("user_id", member.UserID))
			return fmt.Errorf("failed to save team member: %s: %w", member.UserID, err)
		}

		if tag.RowsAffected() == 0 {
			c.logger.Error("failed to save team member: no rows affected", zap.String("user_id", member.UserID))
			return fmt.Errorf("failed to save team member: no rows affected: %s", member.UserName)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.logger.Error("failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	c.logger.Info("successfully stored team to database", zap.String("team_name", team.TeamName))
	return nil
}

func (c *Client) Close() {
	c.pool.Close()
}

func (c *Client) teamExists(ctx context.Context, teamName string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var exists bool

	_, err := withRetry(ctx, c.retryConfig, c.logger, func() (struct{}, error) {
		err := c.pool.QueryRow(ctx, queryTeamExists, teamName).Scan(&exists)
		return struct{}{}, err
	})
	if err != nil {
		c.logger.Error("failed to check if team exists", zap.Error(err))
		return false, fmt.Errorf("failed to check if team exists: %w", err)
	}

	return exists, nil
}

func withRetry[T any](ctx context.Context, retryConfig retryConfig, logger *zap.Logger, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for i := 0; i < retryConfig.maxRetries; i++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "42P01" {
				logger.Error("withRetry: non-retryable Postgres error", zap.Error(err))
				return zero, err
			}
		}

		lastErr = err

		if i == retryConfig.maxRetries-1 {
			break
		}

		backoff := retryConfig.baseBackoff * time.Duration(math.Pow(2, float64(i)))
		jitter := time.Duration(rand.Float64() * float64(retryConfig.baseBackoff))
		pause := backoff + jitter

		select {
		case <-time.After(pause):
		case <-ctx.Done():
			logger.Error(
				"withRetry: context canceled",
				zap.Int("attempts", i+1),
				zap.Duration("backoff", retryConfig.baseBackoff),
			)

			return zero, ctx.Err()
		}

		logger.Warn("withRetry: retrying", zap.Int("attempt", i+1), zap.Duration("backoff", pause))
	}

	return zero, fmt.Errorf("withRetry: all retries failed, lastErr: %w", lastErr)
}

func buildDSN(config *Config) string {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s pool_max_conns=%d pool_min_conns=%d",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.MaxConns,
		config.MinConns,
	)

	return dsn
}
