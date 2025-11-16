package repository

import (
	"context"
	"errors"

	"reviewer-service/internal/domain"
)

var (
	ErrTeamAlreadyExists = errors.New("team already exists")
	ErrTeamNotFound      = errors.New("team not found")
	ErrUserNotFound      = errors.New("user not found")
)

type Repository interface {
	SaveTeam(ctx context.Context, team *domain.Team) error
	GetTeam(ctx context.Context, teamName string) (*domain.Team, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
	Close()
}
