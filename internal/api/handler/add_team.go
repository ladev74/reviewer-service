package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"reviewer-service/internal/api"
	"reviewer-service/internal/repository"
	"reviewer-service/internal/repository/postgres"
)

func AddTeam(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		var team api.Team
		err := json.NewDecoder(r.Body).Decode(&team)
		if err != nil {
			logger.Warn("failed to decode body", zap.Error(err))
			WriteError(w, logger, "failed to decode body", http.StatusBadRequest)
			return
		}

		err = repo.SaveTeam(ctx, &team)
		if err != nil {
			if errors.Is(err, postgres.ErrTeamAlreadyExists) {
				logger.Warn("team already exists", zap.Error(err))
				api.WriteApiError(w, logger, api.ErrTeamExists, api.CodeTeamExists, http.StatusBadRequest)
				return
			}

			logger.Error("failed to save team", zap.Error(err))
			WriteError(w, logger, "failed to save team", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(team)
		if err != nil {
			logger.Error("failed to encode response", zap.Error(err))
		}

		logger.Info("successfully saved team", zap.String("team_name", team.TeamName))
	}
}
