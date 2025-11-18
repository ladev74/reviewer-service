package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				c.logger.Error("failed to save team member: duplicate key", zap.String("user_id", member.UserID))
				return repository.ErrDuplicateKey
			}

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

func (c *Client) SavePR(ctx context.Context, pr domain.PullRequest) (*domain.PullRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	exists, err := c.prExists(ctx, pr.PullRequestId)
	if err != nil {
		return nil, err
	}

	if exists {
		c.logger.Warn(repository.ErrPRAlreadyExists.Error(), zap.String("pull_request_id", pr.PullRequestId))
		return nil, repository.ErrPRAlreadyExists
	}

	teamName, err := c.getTeamName(ctx, pr.AuthorId)
	if err != nil {
		return nil, err
	}

	reviewers, err := c.getActiveReviewers(ctx, teamName, pr.AuthorId)
	if err != nil {
		return nil, err
	}

	if len(reviewers) == 0 {
		c.logger.Warn(repository.ErrReviewersNotFound.Error(), zap.String("pull_request_id", pr.PullRequestId))
		return nil, repository.ErrReviewersNotFound
	}

	tag, err := c.pool.Exec(ctx, querySavePR,
		pr.PullRequestId,
		pr.PullRequestName,
		pr.AuthorId,
		&pr.Status,
		reviewers,
		pr.CreatedAt,
	)
	if err != nil {
		c.logger.Error("failed to save pull request", zap.String("pull_request_id", pr.PullRequestId), zap.Error(err))
		return nil, fmt.Errorf("failed to save pull request: %w", err)
	}

	if tag.RowsAffected() == 0 {
		c.logger.Error("failed to save pull request: no rows affected", zap.String("pull request_id", pr.PullRequestId))
		return nil, fmt.Errorf("failed to save pull request: no rows affected: %s", pr.PullRequestId)
	}

	pr.AssignedReviewers = reviewers

	c.logger.Info("successfully saved pull request", zap.String("pull_request_id", pr.PullRequestId))
	return &pr, nil
}

func (c *Client) SetPRStatus(ctx context.Context, prID string, status string, mergedAt time.Time) (*domain.PullRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var pr domain.PullRequest

	err := c.pool.QueryRow(ctx, querySetPRStatus, prID, status, mergedAt).Scan(
		&pr.PullRequestId,
		&pr.PullRequestName,
		&pr.AuthorId,
		&pr.Status,
		&pr.AssignedReviewers,
		&pr.CreatedAt,
		&pr.MergedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.logger.Warn(repository.ErrPRNotFound.Error(), zap.String("pull_request_id", prID))
			return nil, repository.ErrPRNotFound
		}

		c.logger.Error("failed to set status", zap.String("pull_request_id", prID), zap.Error(err))
		return nil, fmt.Errorf("failed to set status: %w", err)
	}

	c.logger.Info("successfully set status", zap.String("pull_request_id", prID))
	return &pr, nil
}

func (c *Client) ReassignReviewer(ctx context.Context, oldUserID string, prID string) (*domain.PullRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var pr domain.PullRequest
	pr.PullRequestId = prID

	var reviewers pgtype.Array[string]
	err := c.pool.QueryRow(ctx, queryGetPR, prID).Scan(
		&pr.PullRequestName,
		&pr.AuthorId,
		&pr.Status,
		&reviewers,
		&pr.CreatedAt,
		&pr.MergedAt,
	)
	fmt.Println(pr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.logger.Warn(repository.ErrPRNotFound.Error(), zap.String("pull_request_id", prID))
			return nil, repository.ErrPRNotFound
		}

		c.logger.Error("failed to get pull request", zap.String("pull_request_id", prID), zap.Error(err))
		return nil, fmt.Errorf("failed to get pull request: %s: %w", prID, err)
	}

	pr.AssignedReviewers = reviewers.Elements

	if pr.Status == domain.PRStatusMerged {
		c.logger.Warn(repository.ErrPRMerged.Error(), zap.String("pull_request_id", prID))
		return nil, repository.ErrPRMerged
	}

	var found bool
	for _, r := range pr.AssignedReviewers {
		if r == oldUserID {
			found = true
			break
		}
	}

	if !found {
		c.logger.Warn(repository.ErrReviewerNotAssigned.Error(), zap.String("pull_request_id", prID))
		return nil, repository.ErrReviewerNotAssigned
	}

	teamName, err := c.getTeamName(ctx, pr.AuthorId)
	if err != nil {
		c.logger.Error("failed to get team name", zap.String("user_id", pr.AuthorId), zap.Error(err))
		return nil, fmt.Errorf("failed to get team name: %s: %w", pr.AuthorId, err)
	}

	activeReviewers, err := c.getActiveReviewers(ctx, teamName, pr.AuthorId)
	if err != nil {
		return nil, err
	}

	var newReviewer string
	for _, candidate := range activeReviewers {
		if candidate == oldUserID {
			continue
		}
		alreadyAssigned := false
		for _, assigned := range pr.AssignedReviewers {
			if assigned == candidate {
				alreadyAssigned = true
				break
			}
		}
		if !alreadyAssigned {
			newReviewer = candidate
			break
		}
	}

	if newReviewer == "" {
		c.logger.Warn(repository.ErrNoCandidate.Error())
		return nil, repository.ErrNoCandidate
	}

	for i, uid := range pr.AssignedReviewers {
		if uid == oldUserID {
			pr.AssignedReviewers[i] = newReviewer
			break
		}
	}

	tag, err := c.pool.Exec(ctx, queryUpdateAssignedReviewers, pr.AssignedReviewers, pr.PullRequestId)
	if err != nil {
		return nil, fmt.Errorf("failed to update assigned reviewers: %w", err)
	}

	if tag.RowsAffected() == 0 {
		c.logger.Error("failed to update assigned reviewers", zap.String("pull_request_id", pr.PullRequestId))
		return nil, fmt.Errorf("failed to update assigned reviewers: %s: %w", pr.PullRequestId, err)
	}

	c.logger.Info("successfully updated assigned reviewers", zap.String("pull_request_id", pr.PullRequestId))
	return &pr, nil
}

func (c *Client) GetReviewers(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	rows, err := c.pool.Query(ctx, queryGetReviewers, userID)
	if err != nil {
		c.logger.Error("failed to get reviewers", zap.String("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}
	defer rows.Close()

	prs := make([]domain.PullRequestShort, 0)
	for rows.Next() {
		var pr domain.PullRequestShort
		err = rows.Scan(
			&pr.PullRequestId,
			&pr.PullRequestName,
			&pr.AuthorId,
			&pr.Status,
		)
		if err != nil {
			c.logger.Error("failed to scan pull request", zap.String("user_id", userID), zap.Error(err))
			return nil, fmt.Errorf("failed to scan pull request: %w", err)
		}

		prs = append(prs, pr)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	c.logger.Info("successfully got reviewers", zap.Int("prs", len(prs)))
	return prs, nil
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

func (c *Client) prExists(ctx context.Context, prID string) (bool, error) {
	var exists bool

	err := c.pool.QueryRow(ctx, queryPRExists, prID).Scan(&exists)
	if err != nil {
		c.logger.Error("failed to check if pull request exists", zap.Error(err))
		return false, fmt.Errorf("failed to check if pull request exists: %w", err)
	}

	return exists, nil
}

func (c *Client) getTeamName(ctx context.Context, userID string) (string, error) {
	var teamName string

	err := c.pool.QueryRow(ctx, queryGetTeamName, userID).Scan(&teamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.logger.Warn(repository.ErrTeamNotFound.Error(), zap.String("user_id", userID))
			return "", repository.ErrTeamNotFound
		}

		c.logger.Error("failed to get team name", zap.Error(err))
		return "", fmt.Errorf("failed to get team name: %w", err)
	}

	return teamName, nil
}

func (c *Client) getActiveReviewers(ctx context.Context, teamName string, authorId string) ([]string, error) {
	reviewers := make([]string, 0, 1)

	rows, err := c.pool.Query(ctx, queryGetActiveReviewers, teamName, authorId)
	if err != nil {
		c.logger.Error("failed to get active reviewers", zap.String("team_name", teamName), zap.Error(err))
		return nil, fmt.Errorf("failed to get active reviewers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reviewer string

		err = rows.Scan(&reviewer)
		if err != nil {
			c.logger.Error("failed to scan reviewers", zap.String("team_name", teamName), zap.Error(err))
			return nil, fmt.Errorf("failed to scan reviewers: %w", err)
		}

		reviewers = append(reviewers, reviewer)
	}
	err = rows.Err()
	if err != nil {
		c.logger.Error("rows error", zap.String("team_name", teamName), zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}

	c.logger.Info("successfully retrieved active reviewers", zap.String("team_name", teamName))
	return reviewers, nil
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
