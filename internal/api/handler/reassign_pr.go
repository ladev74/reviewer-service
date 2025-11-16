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

type ReassignPRRequest struct {
	PullRequestId string `json:"pull_request_id"`
	OldUserId     string `json:"old_user_id"`
}

func ReassignPR(repo repository.Repository, requestTimeout time.Duration, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		var req ReassignPRRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Warn("ReassignPR: failed to decode body", zap.Error(err))
			writeError(w, logger, "failed to decode body", http.StatusBadRequest)
			return
		}

		pr, err := repo.ReassignReviewer(ctx, req.PullRequestId, req.OldUserId)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrPRNotFound):
				logger.Warn("ReassignPR: pull request not found", zap.String("pull_request_id", req.PullRequestId), zap.Error(err))
				api.WriteApiError(w, logger, api.ErrNotFound, api.CodeNotFound, http.StatusNotFound)
				return

			case errors.Is(err, repository.ErrNoCandidate):
				logger.Warn("ReassignPR: "+err.Error(), zap.String("pull_request_id", req.PullRequestId))
				api.WriteApiError(w, logger, api.ErrNoCandidate, api.CodeNoCandidate, http.StatusConflict)
				return

			case errors.Is(err, repository.ErrPRMerged):
				logger.Warn("ReassignPR: "+err.Error(), zap.String("pull_request_id", req.PullRequestId))
				api.WriteApiError(w, logger, api.ErrPRMerged, api.CodePRMerged, http.StatusConflict)
				return

			case errors.Is(err, repository.ErrReviewerNotAssigned):
				logger.Warn("ReassignPR: "+err.Error(), zap.String("pull_request_id", req.PullRequestId))
				api.WriteApiError(w, logger, api.ErrNotAssigned, api.CodeNotAssigned, http.StatusConflict)
				return
			}

			logger.Error("ReassignPR: failed to get pull request status", zap.String("pull_request_id", req.PullRequestId), zap.Error(err))
			writeError(w, logger, "failed to get pull request", http.StatusInternalServerError)
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
			logger.Error("ReassignPR: failed to encode response", zap.Error(err))
		}

		logger.Info("ReassignPR: successfully created pull request", zap.String("pull_request_id", apiPR.PullRequestId))
	}
}

//curl -X POST http://localhost:8080/team/add \
//-H "Content-Type application/json" \
//-d '{
//"team_name": "backend",
//"members": [
//{
//"user_id": "1",
//"username": "Alice",
//"is_active": true
//},
//{
//"user_id": "1",
//"username": "Bob",
//"is_active": true
//}
//]
//}'
