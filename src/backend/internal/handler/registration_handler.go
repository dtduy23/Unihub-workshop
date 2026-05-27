package handler

import (
	"net/http"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/ratelimiter"
	"unihub-workshop/internal/service"
)

type RegistrationHandler struct {
	regService *service.RegistrationService
}

func NewRegistrationHandler(rs *service.RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{regService: rs}
}

// Register pushes a registration request to the queue (HTTP 202 Accepted)
func (h *RegistrationHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterWorkshopRequest
	if err := decodeJSON(r, &req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID := getUserID(r)

	// ==========================================
	// 1. VIRTUAL WAITING ROOM CHECK
	// ==========================================
	// We use the waiting room to control the flow of users trying to register.
	// Only users with an active token can proceed to the registration queue.
	status, err := h.regService.CheckWaitingRoom(r.Context(), req.WorkshopID, userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to check waiting room")
		return
	}

	if status.Status == ratelimiter.QueueWaiting || status.Status == ratelimiter.QueueAlreadyQueued {
		// User is in queue, tell them to wait
		writeJSON(w, http.StatusTooManyRequests, model.APIResponse{
			Success: false,
			Message: "Workshop is busy. You are in the waiting room.",
			Data:    status,
		})
		return
	}

	// If status is QueueGranted or QueueAlreadyActive, proceed to register
	// ==========================================

	correlationID, err := h.regService.EnqueueRegistration(r.Context(), userID, req.WorkshopID)
	if err != nil {
		errorResponse(w, http.StatusConflict, err.Error())
		return
	}

	// After successful enqueue, we could theoretically release the waiting room token,
	// but it's often better to let it expire naturally or release it after successful processing.
	// For now, we'll let it expire based on TTL.

	writeJSON(w, http.StatusAccepted, model.APIResponse{
		Success: true,
		Message: "Registration is being processed",
		Data: map[string]interface{}{
			"correlation_id": correlationID,
			"waiting_room":   status,
		},
	})
}

// GetWaitingRoomStatus allows a client to poll their queue position without triggering a registration attempt
func (h *RegistrationHandler) GetWaitingRoomStatus(w http.ResponseWriter, r *http.Request) {
	workshopID := getURLParam(r, "workshopId")
	userID := getUserID(r)

	status, err := h.regService.CheckWaitingRoom(r.Context(), workshopID, userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch waiting room status")
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: status})
}

// GetStatus polls the registration status by correlation ID
func (h *RegistrationHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	correlationID := getURLParam(r, "correlationId")
	status := h.regService.GetStatus(correlationID)
	if status == nil {
		errorResponse(w, http.StatusNotFound, "Status not found")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: status})
}

func (h *RegistrationHandler) MyRegistrations(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	regs, err := h.regService.GetUserRegistrationsWithWorkshop(r.Context(), userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch registrations")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: regs})
}

func (h *RegistrationHandler) GetByWorkshopID(w http.ResponseWriter, r *http.Request) {
	workshopID := getURLParam(r, "workshopId")
	regs, err := h.regService.GetByWorkshop(r.Context(), workshopID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch registrations for workshop")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: regs})
}
