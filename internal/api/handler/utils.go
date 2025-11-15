package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, logger *zap.Logger, errMessage string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := ErrorResponse{
		Status:  statusCode,
		Message: errMessage,
	}

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.Error("WriteError: failed to encoding response", zap.Error(err))
	}
}
