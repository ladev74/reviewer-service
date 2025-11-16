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
)

type MergePRRequest struct {
	PullRequestId string `json:"pull_request_id"`
}

func MergePR(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		var req MergePRRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Warn("MergePR: failed to decode body", zap.Error(err))
			writeError(w, logger, "failed to decode body", http.StatusBadRequest)
			return
		}

		tn := time.Now()

		pr, err := repo.SetPRStatus(ctx, req.PullRequestId, api.PRStatusMerged, tn)
		if err != nil {
			if errors.Is(err, repository.ErrPRNotFound) {
				logger.Warn("MergePR: pull request not found", zap.String("pull_request_id", req.PullRequestId), zap.Error(err))
				api.WriteApiError(w, logger, api.ErrNotFound, api.CodeNotFound, http.StatusNotFound)
				return
			}

			logger.Error("MergePR: failed to set pull request status", zap.String("pull_request_id", req.PullRequestId), zap.Error(err))
			writeError(w, logger, "failed to set pull request status", http.StatusInternalServerError)
			return
		}

		apiPR := api.PullRequest{
			PullRequestId:     pr.PullRequestId,
			PullRequestName:   pr.PullRequestName,
			AuthorId:          pr.AuthorId,
			Status:            pr.Status,
			AssignedReviewers: pr.AssignedReviewers,
			CreatedAt:         pr.CreatedAt,
			MergedAt:          pr.MergedAt,
		}

		resp := map[string]api.PullRequest{"pull_request": apiPR}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("MergePR: failed to encode response", zap.Error(err))
		}

		logger.Info("MergePR successfully set pull request status", zap.String("pull_request_id", req.PullRequestId))
	}
}
