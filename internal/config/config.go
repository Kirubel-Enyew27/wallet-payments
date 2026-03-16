package config

import (
	"os"
	"strconv"
)

type Config struct {
	BaseURL       string
	TelebirrBase  string
	MpesaBase     string
	YayaBase      string
	KachaBase     string
	AwashBase     string
	SimulateOnly  bool
	DatabaseURL        string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetimeM int
}

func Load() Config {
	cfg := Config{
		BaseURL:      getenv("BASE_URL", "http://localhost:8080"),
		TelebirrBase: getenv("TELEBIRR_BASE_URL", ""),
		MpesaBase:    getenv("MPESA_BASE_URL", ""),
		YayaBase:     getenv("YAYA_BASE_URL", ""),
		KachaBase:    getenv("KACHA_BASE_URL", ""),
		AwashBase:    getenv("AWASH_BASE_URL", ""),
		DatabaseURL: getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/wallet_payments?sslmode=disable"),
		DBMaxOpenConns: getenvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: getenvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetimeM: getenvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30),
	}
	cfg.SimulateOnly = getenv("SIMULATE_ONLY", "true") == "true"
	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}
