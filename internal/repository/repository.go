package repository

import (
	"context"

	"reviewer-service/internal/api"
)

type Repository interface {
	SaveTeam(ctx context.Context, team *api.Team) error
	Close()
}
