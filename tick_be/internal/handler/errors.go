package handler

import (
	"encoding/json"
	"net/http"

	"tick/be/internal/middleware"
)

type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func respondError(w http.ResponseWriter, httpStatus int, code string, message string) {
	writeJSON(w, httpStatus, ApiError{Code: code, Message: message})
}

func getUserID(r *http.Request) uint {
	val := r.Context().Value(middleware.UserIDKey)
	if val == nil {
		return 0
	}
	return val.(uint)
}
