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

type setIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

func SetIsActive(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		var req setIsActiveRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Warn("SetIsActive: failed to decode body", zap.Error(err))
			writeError(w, logger, "failed to decode body", http.StatusBadRequest)
			return
		}

		user, err := repo.SetIsActive(ctx, req.UserID, req.IsActive)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				logger.Warn("SetIsActive: user not found", zap.Error(err))
				msg := fmt.Sprintf("%s %s", req.UserID, api.ErrNotFound)
				api.WriteApiError(w, logger, msg, api.CodeNotFound, http.StatusNotFound)
				return
			}

			logger.Error("SetIsActive: failed to set is_active", zap.String("user_id", req.UserID), zap.Error(err))
			writeError(w, logger, "failed to set is_active", http.StatusInternalServerError)
			return
		}

		apiUser := api.User{
			UserID:   user.UserID,
			UserName: user.UserName,
			TeamName: user.TeamName,
			IsActive: user.IsActive,
		}

		resp := map[string]api.User{"user": apiUser}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("SetIsActive: failed to encode response", zap.Error(err))
		}

		logger.Info("SetIsActive: successfully set is_active", zap.String("user_id", user.UserID))
	}
}
