package config

import "os"

// Local instances listen only on this PC and use their own credentials.
func loadLocalConfig(cfg *Config) {
	loadDevelopmentConfig(cfg)
	cfg.ListenHost = "127.0.0.1"
	cfg.DatabaseHost = "127.0.0.1"
	cfg.DatabasePassword = os.Getenv("DATABASE_PASSWORD")
	cfg.JWTSecret = []byte(os.Getenv("JWT_SECRET"))
	cfg.RollbarToken = ""
	cfg.UmamiURL = ""
	cfg.UmamiWebsiteID = ""
	cfg.RollbarClientToken = ""
	if Version != "" {
		cfg.Version = Version
	}
	cfg.CORSAllowedOrigins = []string{"http://localhost:9898", "http://127.0.0.1:9898"}
}
