package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wallet-payments-plugin/internal/model"
	"wallet-payments-plugin/internal/providers"
	"wallet-payments-plugin/internal/response"
	"wallet-payments-plugin/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	Store     store.Store
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
		v1.GET("/health", h.Health)
		v1.POST("/payments", h.InitiatePayment)
		v1.GET("/payments", h.ListPayments)
		v1.GET("/payments/:id", h.GetPayment)
		v1.POST("/payments/:id/complete", h.CompletePayment)
		v1.POST("/callbacks/:provider", h.ProviderCallback)
	}
}

func (h *Handler) InitiatePayment(c *gin.Context) {
	ctx := c.Request.Context()
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

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey != "" {
		existing, err := h.Store.GetByIdempotencyKey(ctx, idempotencyKey)
		if err == nil {
			if existing.PaymentMethod != method || existing.Amount != req.Amount || existing.PhoneNumber != req.PhoneNumber {
				response.Error(c, http.StatusConflict, "Idempotency key conflict", "idempotency key already used with different parameters")
				return
			}
			resp := ProcessTransactionResponse{
				Status:      string(existing.Status),
				Channel:     string(existing.Channel),
				PaymentURL:  existing.PaymentURL,
				RedirectURL: strings.TrimRight(h.BaseURL, "/") + "/api/v1/payments/" + existing.ID,
			}
			response.Success(c, "Payment already initiated.", resp)
			return
		}
		if err != nil && err != store.ErrNotFound {
			response.Error(c, http.StatusInternalServerError, "Failed to check idempotency", err.Error())
			return
		}
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
		IdempotencyKey: idempotencyKey,
	}
	if payment.PaymentURL == "" && channel == model.ChannelWeb {
		payment.PaymentURL = strings.TrimRight(h.BaseURL, "/") + "/pay/" + transactionID
	}

	if err := h.Store.Create(ctx, payment); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to store payment", err.Error())
		return
	}

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
	ctx := c.Request.Context()
	id := c.Param("id")
	payment, err := h.Store.Get(ctx, id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payment not found", err.Error())
		return
	}
	response.Success(c, "Payment fetched successfully", payment)
}

func (h *Handler) CompletePayment(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	payment, err := h.Store.Get(ctx, id)
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

	if err := h.Store.Update(ctx, payment); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update payment", err.Error())
		return
	}

	response.Success(c, "Payment updated successfully", payment)
}

func (h *Handler) ProviderCallback(c *gin.Context) {
	ctx := c.Request.Context()
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

	payment, err := h.Store.Get(ctx, req.TransactionID)
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

	if err := h.Store.Update(ctx, payment); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update payment", err.Error())
		return
	}

	response.Success(c, "Callback processed successfully", payment)
}

func (h *Handler) ListPayments(c *gin.Context) {
	ctx := c.Request.Context()

	statusStr := strings.TrimSpace(c.Query("status"))
	method := normalizeMethod(c.Query("method"))
	if c.Query("method") != "" && method == "" {
		response.Error(c, http.StatusBadRequest, "Invalid payment method", "method not supported")
		return
	}

	var status model.PaymentStatus
	if statusStr != "" {
		switch strings.ToLower(statusStr) {
		case "pending":
			status = model.StatusPending
		case "success":
			status = model.StatusSuccess
		case "failed":
			status = model.StatusFailed
		default:
			response.Error(c, http.StatusBadRequest, "Invalid status", "status must be pending, success, or failed")
			return
		}
	}

	limit := parseIntDefault(c.Query("limit"), 50)
	if limit > 200 {
		limit = 200
	}
	offset := parseIntDefault(c.Query("offset"), 0)

	var createdFrom *time.Time
	if v := strings.TrimSpace(c.Query("created_from")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid created_from", "must be RFC3339 timestamp")
			return
		}
		createdFrom = &t
	}

	var createdTo *time.Time
	if v := strings.TrimSpace(c.Query("created_to")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid created_to", "must be RFC3339 timestamp")
			return
		}
		createdTo = &t
	}

	payments, err := h.Store.List(ctx, store.ListFilter{
		Status:        status,
		PaymentMethod: method,
		CreatedFrom:   createdFrom,
		CreatedTo:     createdTo,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list payments", err.Error())
		return
	}
	response.Success(c, "Payments fetched successfully", payments)
}

func (h *Handler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.Store.Health(ctx); err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Database unavailable", err.Error())
		return
	}
	response.Success(c, "ok", gin.H{"database": "ok"})
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

func parseIntDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
