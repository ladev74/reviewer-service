package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"reviewer-service/internal/api"
	"reviewer-service/internal/repository"
)

type getReviewResponse struct {
	UserID       string                 `json:"user_id"`
	PullRequests []api.PullRequestShort `json:"pull_requests"`
}

func GetReview(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			logger.Warn("GetReview: user_id is required")
			writeError(w, logger, "user_id is required", http.StatusBadRequest)
			return
		}

		reviewers, err := repo.GetReviewers(ctx, userID)
		if err != nil {
			logger.Error("failed to get PRs by reviewer", zap.Error(err))
			writeError(w, logger, "failed to get reviewers", http.StatusInternalServerError)
			return
		}

		apiReviewers := make([]api.PullRequestShort, 0, len(reviewers))
		for _, rr := range reviewers {
			apiReviewer := api.PullRequestShort{
				PullRequestId:   rr.PullRequestId,
				PullRequestName: rr.PullRequestName,
				AuthorId:        rr.AuthorId,
				Status:          rr.Status,
			}

			apiReviewers = append(apiReviewers, apiReviewer)
		}

		resp := getReviewResponse{
			UserID:       userID,
			PullRequests: apiReviewers,
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("failed to encode response", zap.Error(err))
		}

		logger.Info("GetReview: success give review")
	}
}
