package main

import (
	"log"

	"wallet-payments-plugin/internal/api"
	"wallet-payments-plugin/internal/config"
	"wallet-payments-plugin/internal/providers"
	"wallet-payments-plugin/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	providersMap := map[string]providers.Provider{
		"TELEBIRR": providers.TelebirrProvider{},
		"M-PESA":   providers.MpesaProvider{},
		"YAYA":     providers.YayaProvider{BaseURL: cfg.YayaBase},
		"KACHA":    providers.KachaProvider{},
		"AWASH":    providers.AwashProvider{},
	}

	handler := &api.Handler{
		Store:     store.New(),
		Providers: providersMap,
		BaseURL:   cfg.BaseURL,
	}

	r := gin.Default()
	handler.RegisterRoutes(r)

	addr := ":8080"
	log.Printf("wallet payments plugin listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
