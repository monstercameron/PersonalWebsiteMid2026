// Package config loads runtime configuration from the environment.
//
// Non-secret settings will also be editable live via the admin console (added later) and read
// here without a restart. Secrets (OpenAI key, signing keys) come from the environment only and
// are never web-editable.
package config

import "os"

// Config is the process configuration.
type Config struct {
	Addr   string // ingress listen address, e.g. 127.0.0.1:8095
	DBPath string // path to the SQLite file
}

// Load reads configuration from the environment, applying sane local defaults.
func Load() Config {
	return Config{
		Addr:   env("LISTEN_ADDR", "127.0.0.1:8095"),
		DBPath: env("DB_PATH", "web/data/site.db"),
	}
}

// env returns the value of environment variable key, or def when it is unset or empty.
func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
