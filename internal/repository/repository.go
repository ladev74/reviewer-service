package repository

import (
	"context"
	"errors"
	"time"

	"reviewer-service/internal/domain"
)

var (
	ErrTeamAlreadyExists = errors.New("team already exists")
	ErrPRAlreadyExists   = errors.New("pull request already exists")

	ErrTeamNotFound      = errors.New("team not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrReviewersNotFound = errors.New("reviewers not found")
	ErrPRNotFound        = errors.New("pull request not found")
)

type Repository interface {
	SaveTeam(ctx context.Context, team *domain.Team) error
	GetTeam(ctx context.Context, teamName string) (*domain.Team, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
	SavePR(ctx context.Context, pr domain.PullRequest) (*domain.PullRequest, error)
	SetPRStatus(ctx context.Context, prID string, status string, mergedAt time.Time) (*domain.PullRequest, error)
	Close()
}
