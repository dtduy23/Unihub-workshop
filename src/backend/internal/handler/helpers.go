package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"unihub-workshop/internal/middleware"
	"unihub-workshop/internal/model"
)

// Helper functions shared across handlers

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getUserID(r *http.Request) string {
	return middleware.GetUserID(r.Context())
}

func getURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func errorResponse(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.APIResponse{Error: msg})
}
