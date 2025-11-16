package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"reviewer-service/internal/api"
	"reviewer-service/internal/domain"
	"reviewer-service/internal/repository"
)

type createPRRequest struct {
	PullRequestId   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorId        string `json:"author_id"`
}

func CreatePR(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		_ = ctx

		var req createPRRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Warn("CreatePR: failed to decode body", zap.Error(err))
			writeError(w, logger, "failed to decode body", http.StatusBadRequest)
			return
		}

		tn := time.Now()
		pr := domain.PullRequest{
			PullRequestId:     req.PullRequestId,
			PullRequestName:   req.PullRequestName,
			AuthorId:          req.AuthorId,
			Status:            api.PRStatusOpen,
			AssignedReviewers: nil,
			CreatedAt:         &tn,
			MergedAt:          nil,
		}

		newPR, err := repo.SavePR(ctx, pr)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrPRAlreadyExists):
				logger.Warn("CreatePR: PR already exists", zap.Error(err))
				api.WriteApiError(w, logger, api.ErrPRExists, api.CodePRExists, http.StatusConflict)
				return

			case errors.Is(err, repository.ErrUserNotFound) || errors.Is(err, repository.ErrTeamNotFound):
				logger.Warn("CreatePR: not found", zap.Error(err))
				api.WriteApiError(w, logger, api.ErrNotFound, api.CodeNotFound, http.StatusNotFound)
				return
			}

			logger.Error("CreatePR: failed to save pull request", zap.Error(err))
			writeError(w, logger, "failed to save pull request", http.StatusInternalServerError)
			return
		}

		apiPR := api.PullRequest{
			PullRequestId:     newPR.PullRequestId,
			PullRequestName:   newPR.PullRequestName,
			AuthorId:          newPR.AuthorId,
			Status:            newPR.Status,
			AssignedReviewers: newPR.AssignedReviewers,
			CreatedAt:         newPR.CreatedAt,
			MergedAt:          newPR.MergedAt,
		}

		resp := map[string]api.PullRequest{"pull_request": apiPR}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("CreatePR: failed to encode response", zap.Error(err))
		}

		logger.Info("CreatePR: successfully created pull request", zap.String("pull_request_id", apiPR.PullRequestId))
	}
}
