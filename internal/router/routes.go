package router

import (
	"wallet-payments-plugin/internal/api"

	"github.com/gin-gonic/gin"
)

func Registerroutes(r *gin.Engine, h *api.Handler) {
	router := r.Group("/api/v1")

	router.GET("/health", h.Health)
	router.POST("/payments", h.InitiatePayment)
	router.GET("/payments", h.ListPayments)
	router.GET("/payments/:id", h.GetPayment)
	router.POST("/payments/:id/complete", h.CompletePayment)
	router.POST("/callbacks/:provider", h.ProviderCallback)

}
