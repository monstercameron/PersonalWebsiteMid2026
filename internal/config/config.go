// Package config loads runtime configuration from the environment.
//
// Non-secret settings will also be editable live via the admin console (added later) and read
// here without a restart. Secrets (OpenAI key, signing keys) come from the environment only and
// are never web-editable.
package config

import (
	"os"
	"strings"
)

// Config is the process configuration.
type Config struct {
	Addr   string // ingress listen address, e.g. 127.0.0.1:8095
	DBPath string // path to the SQLite file
	// AllowedOrigins are extra WebSocket upgrade origins permitted beyond same-origin (comma-
	// separated ALLOWED_ORIGINS). Same-origin requests are always allowed; everything else is
	// rejected to prevent cross-site WebSocket hijacking.
	AllowedOrigins []string
	// AdminPassword gates the /admin config page. Empty disables admin login entirely.
	AdminPassword string
	// AdminSecret signs admin session cookies (defaults to an insecure dev value if empty).
	AdminSecret string
	// BaseURL is the public origin, used to build absolute RSS links.
	BaseURL string
}

// Load reads configuration from the environment, applying sane local defaults.
func Load() Config {
	return Config{
		Addr:           env("LISTEN_ADDR", "127.0.0.1:8095"),
		DBPath:         env("DB_PATH", "web/data/site.db"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),
		AdminSecret:    os.Getenv("ADMIN_SECRET"),
		BaseURL:        env("BASE_URL", "http://127.0.0.1:8095"),
	}
}

// splitCSV splits a comma-separated list, trimming spaces and dropping empties.
func splitCSV(csv string) []string {
	if csv == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(csv, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// env returns the value of environment variable key, or def when it is unset or empty.
func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
