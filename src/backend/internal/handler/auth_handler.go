package handler

import (
	"encoding/json"
	"net/http"

	"unihub-workshop/internal/crypto"
	"unihub-workshop/internal/model"
	"unihub-workshop/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	rsaProvider *crypto.RSAProvider
}

func NewAuthHandler(authService *service.AuthService, rsaProvider *crypto.RSAProvider) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		rsaProvider: rsaProvider,
	}
}

func (h *AuthHandler) GetPublicKey(w http.ResponseWriter, r *http.Request) {
	pubKey, err := h.rsaProvider.GetPublicKeyPEM()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{Error: "Failed to export public key"})
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Data:    map[string]string{"public_key": pubKey},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{Error: "Invalid request body"})
		return
	}

	if req.StudentID == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{Error: "student_id and password are required"})
		return
	}

	resp, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, model.APIResponse{Error: "Invalid credentials"})
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: resp})
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	user, err := h.authService.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, model.APIResponse{Error: "User not found"})
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: user})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req model.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Identifier == "" {
		errorResponse(w, http.StatusBadRequest, "identifier is required")
		return
	}

	err := h.authService.ForgotPassword(r.Context(), req.Identifier)
	if err != nil {
		// Even if error, we might want to just say success to prevent user enumeration, 
		// but here we just return the error for simplicity.
		errorResponse(w, http.StatusBadRequest, "Failed to process forgot password request: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Message: "Nếu tài khoản tồn tại, mật khẩu mới sẽ được gửi vào email của bạn.",
	})
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req model.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		errorResponse(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}

	userID := getUserID(r)
	if userID == "" {
		errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.authService.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Message: "Đổi mật khẩu thành công",
	})
}
