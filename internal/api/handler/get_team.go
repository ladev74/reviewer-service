package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"reviewer-service/internal/api"
	"reviewer-service/internal/repository"
)

func GetTeam(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		teamName := r.URL.Query().Get("team_name")
		if teamName == "" {
			logger.Warn("GetTeam: team_name is required")
			writeError(w, logger, "team_name is required", http.StatusBadRequest)
			return
		}

		team, err := repo.GetTeam(ctx, teamName)
		if err != nil {
			if errors.Is(err, repository.ErrTeamNotFound) {
				logger.Warn("GetTeam: team not found", zap.String("team_name", teamName), zap.Error(err))
				msg := fmt.Sprintf("%s %s", teamName, api.ErrNotFound)
				api.WriteApiError(w, logger, msg, api.CodeNotFound, http.StatusNotFound)
				return
			}
			logger.Error("GetTeam: get team failed", zap.Error(err))
			writeError(w, logger, "get team failed", http.StatusInternalServerError)
			return
		}

		members := make([]api.TeamMember, len(team.Members))
		for i, m := range team.Members {
			members[i] = api.TeamMember{
				UserID:   m.UserID,
				UserName: m.UserName,
				IsActive: m.IsActive,
			}
		}

		apiTeam := api.Team{
			TeamName: team.TeamName,
			Members:  members,
		}

		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(apiTeam)
		if err != nil {
			logger.Error("GetTeam: failed to encode response", zap.Error(err))
		}

		logger.Info("GetTeam: successfully give team", zap.String("team_name", teamName))
	}
}
