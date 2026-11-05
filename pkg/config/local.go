package config

import "os"

// Local instances listen only on this PC and use their own credentials.
func loadLocalConfig(cfg *Config) {
	loadDevelopmentConfig(cfg)
	cfg.DatabasePassword = os.Getenv("DATABASE_PASSWORD")
	cfg.JWTSecret = []byte(os.Getenv("JWT_SECRET"))
	isolateLocalConfig(cfg)
	cfg.CORSAllowedOrigins = []string{"http://localhost:9898", "http://127.0.0.1:9898"}
}

// Apply local isolation after environment overrides as well as defaults.
func isolateLocalConfig(cfg *Config) {
	cfg.ListenHost = "127.0.0.1"
	cfg.DatabaseHost = "127.0.0.1"
	cfg.RollbarToken = ""
	cfg.UmamiURL = ""
	cfg.UmamiWebsiteID = ""
	cfg.RollbarClientToken = ""
	cfg.BlogURL = ""
	if Version != "" {
		cfg.Version = Version
	}
}
