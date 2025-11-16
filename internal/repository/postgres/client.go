package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"reviewer-service/internal/domain"
	"reviewer-service/internal/repository"
)

func New(ctx context.Context, config *Config, logger *zap.Logger) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	dsn := buildDSN(config)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Client{
		pool:    pool,
		logger:  logger,
		timeout: config.Timeout,
	}, nil
}

func (c *Client) SaveTeam(ctx context.Context, team *domain.Team) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	exists, err := c.teamExists(ctx, team.TeamName)
	if err != nil {
		return err
	}

	if exists {
		c.logger.Warn(repository.ErrTeamAlreadyExists.Error(), zap.Any("team_name", team.TeamName))
		return fmt.Errorf("%w: %s", repository.ErrTeamAlreadyExists, team.TeamName)
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		c.logger.Error("failed to start transaction", zap.Error(err))
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, querySetTeamName, team.TeamName)
	if err != nil {
		c.logger.Error("failed to set team name", zap.Error(err), zap.String("team_name", team.TeamName))
		return fmt.Errorf("failed to set team name: %s: %w", team.TeamName, err)
	}

	if tag.RowsAffected() == 0 {
		c.logger.Error("failed to set team name: no rows affected", zap.String("team_name", team.TeamName))
		return fmt.Errorf("failed to set team name: no rows affected: %s", team.TeamName)
	}

	for _, member := range team.Members {
		tag, err = tx.Exec(ctx, querySaveTeamMember, member.UserID, member.UserName, team.TeamName, member.IsActive)
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

func (c *Client) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	exists, err := c.teamExists(ctx, teamName)
	if err != nil {
		return nil, err
	}

	if !exists {
		c.logger.Warn(repository.ErrTeamNotFound.Error(), zap.String("team_name", teamName))
		return nil, repository.ErrTeamNotFound
	}

	rows, err := c.pool.Query(ctx, queryGetTeam, teamName)
	if err != nil {
		c.logger.Error("failed to get team member", zap.String("team_name", teamName), zap.Error(err))
		return nil, fmt.Errorf("failed to get team member: %w", err)
	}
	defer rows.Close()

	members := make([]domain.TeamMember, 0)
	for rows.Next() {
		var member domain.TeamMember

		err = rows.Scan(&member.UserID, &member.UserName, &member.IsActive)
		if err != nil {
			c.logger.Error("failed to scan member", zap.String("team_name", teamName), zap.Error(err))
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}

		members = append(members, member)
	}
	err = rows.Err()
	if err != nil {
		c.logger.Error("rows error", zap.String("team_name", teamName), zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}

	c.logger.Info("successfully retrieved team members", zap.String("team_name", teamName))
	return &domain.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (c *Client) SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var user domain.User
	err := c.pool.QueryRow(ctx, querySetIsActive, userID, isActive).
		Scan(&user.UserID, &user.UserName, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.logger.Warn(repository.ErrUserNotFound.Error(), zap.String("user_id", userID))
			return nil, repository.ErrUserNotFound
		}

		c.logger.Error("failed to set is_active", zap.String("user_id", userID))
		return nil, fmt.Errorf("failed to set is_active: %w", err)
	}

	c.logger.Info("successfully set is_active", zap.String("user_id", userID))
	return &user, nil
}

func (c *Client) Close() {
	c.pool.Close()
}

func (c *Client) teamExists(ctx context.Context, teamName string) (bool, error) {
	var exists bool

	err := c.pool.QueryRow(ctx, queryTeamExists, teamName).Scan(&exists)
	if err != nil {
		c.logger.Error("failed to check if team exists", zap.Error(err))
		return false, fmt.Errorf("failed to check if team exists: %w", err)
	}

	return exists, nil
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
