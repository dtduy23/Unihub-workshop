package handler

import (
	"io"
	"net/http"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/service"
)

type AIHandler struct {
	aiService *service.AISummaryService
}

func NewAIHandler(aiService *service.AISummaryService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
	}
}

// SummarizePDF handles the PDF upload and returns an AI-generated summary
// POST /api/v1/ai/summarize
func (h *AIHandler) SummarizePDF(w http.ResponseWriter, r *http.Request) {
	// 1. Parse multi-part form (limit 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		errorResponse(w, http.StatusBadRequest, "Không thể xử lý form dữ liệu: " + err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Vui lòng đính kèm file PDF trong trường 'file'")
		return
	}
	defer file.Close()

	// 2. Read file content
	pdfData, err := io.ReadAll(file)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Lỗi đọc file: " + err.Error())
		return
	}

	// 3. Check if it's actually a PDF
	if len(pdfData) < 4 || string(pdfData[:4]) != "%PDF" {
		errorResponse(w, http.StatusBadRequest, "Định dạng file không hỗ trợ. Chỉ chấp nhận file PDF.")
		return
	}

	// 4. Call AI Service (Interactive Summarization)
	summary, err := h.aiService.Summarize(r.Context(), pdfData)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 5. Return success
	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Data: map[string]string{
			"summary": summary,
		},
		Message: "Tóm tắt AI thành công",
	})
}
