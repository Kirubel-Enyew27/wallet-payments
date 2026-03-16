package config

import "os"

type Config struct {
	BaseURL       string
	TelebirrBase  string
	MpesaBase     string
	YayaBase      string
	KachaBase     string
	AwashBase     string
	SimulateOnly  bool
}

func Load() Config {
	cfg := Config{
		BaseURL:      getenv("BASE_URL", "http://localhost:8080"),
		TelebirrBase: getenv("TELEBIRR_BASE_URL", ""),
		MpesaBase:    getenv("MPESA_BASE_URL", ""),
		YayaBase:     getenv("YAYA_BASE_URL", ""),
		KachaBase:    getenv("KACHA_BASE_URL", ""),
		AwashBase:    getenv("AWASH_BASE_URL", ""),
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
