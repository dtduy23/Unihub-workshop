package handler

import (
	"net/http"
	"unihub-workshop/internal/middleware"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/service"
)

type CheckinHandler struct {
	checkinService *service.CheckinService
}

func NewCheckinHandler(cs *service.CheckinService) *CheckinHandler {
	return &CheckinHandler{checkinService: cs}
}

// LiveCheckin handles online QR check-in
func (h *CheckinHandler) LiveCheckin(w http.ResponseWriter, r *http.Request) {
	var req model.CheckinRequest
	if err := decodeJSON(r, &req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !h.checkinService.Allowed(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()), req.WorkshopID) {
		errorResponse(w, 403, "Workshop không được phân công")
		return
	}
	if err := h.checkinService.LiveCheckin(r.Context(), &req); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Message: "Check-in successful"})
}

// BulkSync handles offline check-in synchronization from mobile app
func (h *CheckinHandler) BulkSync(w http.ResponseWriter, r *http.Request) {
	var req model.BulkCheckinRequest
	if err := decodeJSON(r, &req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Records) > 500 {
		errorResponse(w, http.StatusBadRequest, "Maximum 500 records per sync request")
		return
	}

	for _, record := range req.Records {
		if !h.checkinService.Allowed(r.Context(), getUserID(r), middleware.GetUserRole(r.Context()), record.WorkshopID) {
			errorResponse(w, 403, "Workshop không được phân công")
			return
		}
	}
	synced, failed := h.checkinService.BulkSync(r.Context(), req.Records)

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"synced": synced,
			"failed": failed,
		},
	})
}
