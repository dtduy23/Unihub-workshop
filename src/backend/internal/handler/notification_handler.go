package handler

import (
	"net/http"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/service"
)

type NotificationHandler struct {
	notifService *service.NotificationService
}

func NewNotificationHandler(ns *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: ns}
}

// GetMyNotifications returns notification history for the authenticated user
func (h *NotificationHandler) GetMyNotifications(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	notifs, err := h.notifService.GetUserNotifications(r.Context(), userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: notifs})
}
