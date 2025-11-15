package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

const (
	CodeTeamExists  = "TEAM_EXISTS"
	CodePRExists    = "PR_EXISTS"
	CodePRMerged    = "PR_MERGED"
	CodeNotAssigned = "NOT_ASSIGNED"
	CodeNoCandidate = "NO_CANDIDATE"
	CodeNotFound    = "NOT_FOUND"
)

const (
	ErrTeamExists  = "team_name already exists"
	ErrPRExists    = ""
	ErrPRMerged    = ""
	ErrNotAssigned = ""
	ErrNoCandidate = ""
	ErrNotFound    = ""
)

type Error struct {
	Code    string
	Message string
}

func WriteApiError(w http.ResponseWriter, logger *zap.Logger, message string, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	e := Error{
		Code:    code,
		Message: message,
	}

	err := json.NewEncoder(w).Encode(e)
	if err != nil {
		logger.Error("WriteError: failed to encoding response", zap.Error(err))
	}
}
