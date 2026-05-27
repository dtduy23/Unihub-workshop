package handler

import (
	"net/http"

	"unihub-workshop/internal/model"
	"unihub-workshop/internal/service"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(ps *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: ps}
}

// InitiatePayment starts payment flow for a paid workshop registration
func (h *PaymentHandler) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	registrationID := getURLParam(r, "registrationId")

	payment, checkoutURL, err := h.paymentService.InitiatePayment(r.Context(), registrationID)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"payment":      payment,
			"checkout_url": checkoutURL,
		},
	})
}

// Webhook handles payment gateway callbacks
func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	var req model.PaymentWebhookRequest
	if err := decodeJSON(r, &req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid webhook payload")
		return
	}

	if err := h.paymentService.HandleWebhook(r.Context(), &req); err != nil {
		if err.Error() == "invalid signature" {
			errorResponse(w, http.StatusUnauthorized, "Invalid signature")
			return
		}
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Message: "Webhook processed"})
}

// GetPendingPayments returns all registrations currently in PENDING_PAYMENT status
func (h *PaymentHandler) GetPendingPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := h.paymentService.GetPendingPayments(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: payments})
}

// GetGatewayStatus returns the current up/down status of the mock gateway
func (h *PaymentHandler) GetGatewayStatus(w http.ResponseWriter, r *http.Request) {
	isDown := h.paymentService.IsGatewayDown(r.Context())
	status := "up"
	if isDown {
		status = "down"
	}
	writeJSON(w, http.StatusOK, model.APIResponse{Success: true, Data: map[string]string{"status": status}})
}

// GetCircuitBreakerStatus returns the current state of the payment circuit breaker
func (h *PaymentHandler) GetCircuitBreakerStatus(w http.ResponseWriter, r *http.Request) {
	state := h.paymentService.GetCircuitBreakerState()
	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Data:    map[string]string{"circuit_breaker_state": state},
	})
}

// GetPaymentStatus returns the current status of a transaction
func (h *PaymentHandler) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	txID := getURLParam(r, "transactionId")
	status, err := h.paymentService.GetStatus(r.Context(), txID)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "Transaction not found")
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Data:    map[string]string{"status": string(status)},
	})
}
