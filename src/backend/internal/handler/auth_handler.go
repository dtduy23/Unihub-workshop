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
