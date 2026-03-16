package api

import (
	"errors"
	"net/http"
	"strings"

	"wallet-payments-plugin/internal/model"
	"wallet-payments-plugin/internal/providers"
	"wallet-payments-plugin/internal/response"
	"wallet-payments-plugin/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	Store     *store.Store
	Providers map[string]providers.Provider
	BaseURL   string
}

type InitiatePaymentRequest struct {
	PaymentMethod string `json:"payment_method"`
	Amount        string `json:"amount"`
	PhoneNumber   string `json:"phone_number"`
}

type ProcessTransactionResponse struct {
	Status      string `json:"status"`
	Channel     string `json:"payment_channel"`
	PaymentURL  string `json:"payment_url"`
	RedirectURL string `json:"redirect_url"`
}

type CompletePaymentRequest struct {
	Status       string `json:"status"`
	FailedReason string `json:"failed_reason"`
}

type CallbackRequest struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	FailedReason  string `json:"failed_reason"`
	PaymentURL    string `json:"payment_url"`
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.POST("/payments", h.InitiatePayment)
		v1.GET("/payments/:id", h.GetPayment)
		v1.POST("/payments/:id/complete", h.CompletePayment)
		v1.POST("/callbacks/:provider", h.ProviderCallback)
	}
}

func (h *Handler) InitiatePayment(c *gin.Context) {
	var req InitiatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	method := normalizeMethod(req.PaymentMethod)
	if method == "" {
		response.Error(c, http.StatusBadRequest, "Unsupported payment method", "payment_method is required")
		return
	}

	provider, ok := h.Providers[method]
	if !ok {
		response.Error(c, http.StatusBadRequest, "Unsupported payment method", "payment_method not supported")
		return
	}

	if err := validatePhone(req.PhoneNumber); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid phone number", err.Error())
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		response.Error(c, http.StatusBadRequest, "Invalid amount", "amount must be greater than 0")
		return
	}

	transactionID := uuid.NewString()
	initResult, err := provider.Initiate(c.Request.Context(), providers.InitiateRequest{
		TransactionID: transactionID,
		PaymentMethod: method,
		Amount:        req.Amount,
		PhoneNumber:   req.PhoneNumber,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to initiate payment", err.Error())
		return
	}

	channel := model.ChannelUSSD
	if strings.EqualFold(initResult.Channel, "WEB") {
		channel = model.ChannelWeb
	}

	payment := &model.Payment{
		ID:            transactionID,
		PaymentMethod: method,
		Amount:        req.Amount,
		PhoneNumber:   req.PhoneNumber,
		Status:        model.StatusPending,
		Channel:       channel,
		PaymentURL:    initResult.PaymentURL,
		ProviderRef:   initResult.ProviderRef,
	}
	if payment.PaymentURL == "" && channel == model.ChannelWeb {
		payment.PaymentURL = strings.TrimRight(h.BaseURL, "/") + "/pay/" + transactionID
	}

	h.Store.Create(payment)

	resp := ProcessTransactionResponse{
		Status:      string(payment.Status),
		Channel:     string(payment.Channel),
		PaymentURL:  payment.PaymentURL,
		RedirectURL: strings.TrimRight(h.BaseURL, "/") + "/api/v1/payments/" + payment.ID,
	}

	if channel == model.ChannelWeb {
		response.Success(c, "Payment link generated successfully, please complete your payment.", resp)
		return
	}
	response.Success(c, "Payment initiated. Please complete the USSD prompt.", resp)
}

func (h *Handler) GetPayment(c *gin.Context) {
	id := c.Param("id")
	payment, err := h.Store.Get(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payment not found", err.Error())
		return
	}
	response.Success(c, "Payment fetched successfully", payment)
}

func (h *Handler) CompletePayment(c *gin.Context) {
	id := c.Param("id")
	payment, err := h.Store.Get(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payment not found", err.Error())
		return
	}

	var req CompletePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch status {
	case "success":
		payment.Status = model.StatusSuccess
		payment.FailedReason = ""
	case "failed":
		payment.Status = model.StatusFailed
		payment.FailedReason = req.FailedReason
	default:
		response.Error(c, http.StatusBadRequest, "Invalid status", "status must be success or failed")
		return
	}

	if err := h.Store.Update(payment); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update payment", err.Error())
		return
	}

	response.Success(c, "Payment updated successfully", payment)
}

func (h *Handler) ProviderCallback(c *gin.Context) {
	providerName := strings.ToUpper(c.Param("provider"))
	if providerName == "MPESA" {
		providerName = "M-PESA"
	}

	var req CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.TransactionID == "" {
		response.Error(c, http.StatusBadRequest, "Invalid request body", "transaction_id is required")
		return
	}

	payment, err := h.Store.Get(req.TransactionID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payment not found", err.Error())
		return
	}
	if payment.PaymentMethod != providerName {
		response.Error(c, http.StatusBadRequest, "Payment method mismatch", "provider does not match payment method")
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch status {
	case "success":
		payment.Status = model.StatusSuccess
		payment.FailedReason = ""
	case "failed":
		payment.Status = model.StatusFailed
		payment.FailedReason = req.FailedReason
	default:
		response.Error(c, http.StatusBadRequest, "Invalid status", "status must be success or failed")
		return
	}

	if req.PaymentURL != "" {
		payment.PaymentURL = req.PaymentURL
	}

	if err := h.Store.Update(payment); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update payment", err.Error())
		return
	}

	response.Success(c, "Callback processed successfully", payment)
}

func normalizeMethod(method string) string {
	m := strings.ToUpper(strings.TrimSpace(method))
	switch m {
	case "TELEBIRR":
		return "TELEBIRR"
	case "M-PESA", "MPESA":
		return "M-PESA"
	case "YAYA":
		return "YAYA"
	case "KACHA":
		return "KACHA"
	case "AWASH":
		return "AWASH"
	default:
		return ""
	}
}

func validatePhone(phone string) error {
	if len(phone) != 12 {
		return errors.New("phone_number must be 12 digits and start with 251")
	}
	if !strings.HasPrefix(phone, "251") {
		return errors.New("phone_number must start with 251")
	}
	return nil
}
