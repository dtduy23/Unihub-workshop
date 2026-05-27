package handler

import (
	"encoding/json"
	"net/http"
	"unihub-workshop/internal/model"
	"github.com/redis/go-redis/v9"
)

type MockHandler struct {
	redisClient *redis.Client
}

func NewMockHandler(redisClient *redis.Client) *MockHandler {
	return &MockHandler{redisClient: redisClient}
}

// ToggleGatewayStatus allows the admin to simulate gateway UP or DOWN
func (h *MockHandler) ToggleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"` // "up" or "down"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Status != "up" && req.Status != "down" {
		http.Error(w, "Status must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	err := h.redisClient.Set(r.Context(), "mock:gateway:status", req.Status, 0).Err()
	if err != nil {
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.APIResponse{
		Success: true,
		Message: "Gateway status updated to " + req.Status,
	})
}

func (h *MockHandler) GetGatewayStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.redisClient.Get(r.Context(), "mock:gateway:status").Result()
	if err != nil && err != redis.Nil {
		http.Error(w, "Failed to get status", http.StatusInternalServerError)
		return
	}

	if status == "" {
		status = "up" // Default
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.APIResponse{
		Success: true,
		Data:    map[string]string{"status": status},
	})
}
