package handler

import (
	"log"
	"net/http"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/service"
)

type WorkshopHandler struct {
	workshopService *service.WorkshopService
}

func NewWorkshopHandler(ws *service.WorkshopService) *WorkshopHandler {
	return &WorkshopHandler{workshopService: ws}
}

func (h *WorkshopHandler) List(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	workshops, err := h.workshopService.ListAll(r.Context(), title)
	if err != nil {
		log.Printf("[ERROR] ListAll failed: %v", err)
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch workshops")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: workshops})
}

func (h *WorkshopHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	workshop, err := h.workshopService.GetByID(r.Context(), id)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "Workshop not found")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: workshop})
}

func (h *WorkshopHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateWorkshopRequest
	if err := decodeJSON(r, &req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	workshop, err := h.workshopService.Create(r.Context(), &req)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResponse{Success: true, Data: workshop})
}

func (h *WorkshopHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	var req model.UpdateWorkshopRequest
	if err := decodeJSON(r, &req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.workshopService.Update(r.Context(), id, &req); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Message: "Workshop updated"})
}

func (h *WorkshopHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	if err := h.workshopService.Delete(r.Context(), id); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Message: "Workshop cancelled"})
}
