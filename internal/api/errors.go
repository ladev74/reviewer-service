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
	ErrTeamExists  = "already exists"
	ErrPRExists    = "PR id already exists"
	ErrPRMerged    = ""
	ErrNotAssigned = ""
	ErrNoCandidate = ""
	ErrNotFound    = "not found"
)

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WriteApiError(w http.ResponseWriter, logger *zap.Logger, message string, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	e := apiError{}
	e.Error.Code = code
	e.Error.Message = message

	err := json.NewEncoder(w).Encode(e)
	if err != nil {
		logger.Error("WriteError: failed to encoding response", zap.Error(err))
	}
}
