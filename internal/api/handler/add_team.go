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
	"reviewer-service/internal/domain"
	"reviewer-service/internal/repository"
)

func AddTeam(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		var team api.Team
		err := json.NewDecoder(r.Body).Decode(&team)
		if err != nil {
			logger.Warn("AddTeam: failed to decode body", zap.Error(err))
			writeError(w, logger, "failed to decode body", http.StatusBadRequest)
			return
		}

		members := make([]domain.TeamMember, len(team.Members))
		for i, m := range team.Members {
			members[i] = domain.TeamMember{
				UserID:   m.UserID,
				UserName: m.UserName,
				IsActive: m.IsActive,
			}
		}

		dTeam := &domain.Team{
			TeamName: team.TeamName,
			Members:  members,
		}

		err = repo.SaveTeam(ctx, dTeam)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrTeamAlreadyExists):
				logger.Warn("AddTeam: team already exists", zap.Error(err))
				msg := fmt.Sprintf("%s %s", team.TeamName, api.ErrTeamExists)
				api.WriteApiError(w, logger, msg, api.CodeTeamExists, http.StatusBadRequest)
				return

			case errors.Is(err, repository.ErrDuplicateKey):
				logger.Warn("AddTeam: duplicate key", zap.Error(err))
				writeError(w, logger, "duplicate key", http.StatusBadRequest)
				return
			}

			logger.Error("AddTeam: failed to save team", zap.Error(err))
			writeError(w, logger, "failed to save team", http.StatusInternalServerError)
			return
		}

		resp := map[string]api.Team{"team": team}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("AddTeam: failed to encode response", zap.Error(err))
		}

		logger.Info("AddTeam: successfully saved team", zap.String("team_name", team.TeamName))
	}
}
