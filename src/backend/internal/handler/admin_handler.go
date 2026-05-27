package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/repository"
	"unihub-workshop/internal/service"
)

type AdminHandler struct {
	batchService    *service.BatchImportService
	aiService       *service.AISummaryService
	userRepo        *repository.UserRepo
	importDir       string
}

func NewAdminHandler(
	batchService *service.BatchImportService,
	aiService *service.AISummaryService,
	userRepo *repository.UserRepo,
	importDir string,
) *AdminHandler {
	return &AdminHandler{
		batchService: batchService,
		aiService:    aiService,
		userRepo:     userRepo,
		importDir:    importDir,
	}
}

// UploadCSV handles CSV file upload for batch import
func (h *AdminHandler) UploadCSV(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20) // 32MB max

	file, header, err := r.FormFile("file")
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	if filepath.Ext(header.Filename) != ".csv" {
		errorResponse(w, http.StatusBadRequest, "Only CSV files are accepted")
		return
	}

	// Save to import directory
	destPath := filepath.Join(h.importDir, header.Filename)
	dest, err := os.Create(destPath)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	// Queue for nightly processing (or manual trigger)
	job, err := h.batchService.QueueJob(r.Context(), header.Filename)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to queue job: %v", err))
		return
	}

	writeJSON(w, http.StatusAccepted, model.APIResponse{
		Success: true,
		Message: "CSV uploaded and scheduled for processing at 02:00 AM",
		Data:    job,
	})
}

// RunJob manually triggers a pending import job
func (h *AdminHandler) RunJob(w http.ResponseWriter, r *http.Request) {
	jobID := getURLParam(r, "id")
	if jobID == "" {
		errorResponse(w, http.StatusBadRequest, "Missing job ID")
		return
	}

	if err := h.batchService.RunJobByID(r.Context(), jobID); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Message: "Job execution started in background",
	})
}

// GetImportJobs returns import job history
func (h *AdminHandler) GetImportJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.batchService.GetJobs(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch import jobs")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: jobs})
}

// GetImportErrors returns error details for a specific job
func (h *AdminHandler) GetImportErrors(w http.ResponseWriter, r *http.Request) {
	jobID := getURLParam(r, "id")
	if jobID == "" {
		errorResponse(w, http.StatusBadRequest, "Missing job ID")
		return
	}

	errors, err := h.batchService.GetErrors(r.Context(), jobID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch import errors")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: errors})
}

// UploadPDF handles PDF upload for AI summary generation
func (h *AdminHandler) UploadPDF(w http.ResponseWriter, r *http.Request) {
	workshopID := getURLParam(r, "workshopId")

	r.ParseMultipartForm(32 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	if filepath.Ext(header.Filename) != ".pdf" {
		errorResponse(w, http.StatusBadRequest, "Only PDF files are accepted")
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	summary, err := h.aiService.ProcessPDF(r.Context(), workshopID, content)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("AI summary failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Message: "AI summary generated",
		Data:    map[string]string{"summary": summary},
	})
}

// GetStats returns system statistics
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.userRepo.GetStats(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: stats})
}
