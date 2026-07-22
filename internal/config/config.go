// Package config loads runtime configuration from the environment.
//
// Values here are startup defaults. Some are also editable live via the admin settings page
// (stored in the DB, which overrides the env value without a restart) — currently the OpenAI key
// and model. Signing keys and the admin password remain environment-only.
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
	// AdminUsername is the admin login username (default "cam").
	AdminUsername string
	// AdminPassword gates the /admin config page. Empty disables admin login entirely.
	AdminPassword string
	// AdminSecret signs admin session cookies (defaults to an insecure dev value if empty).
	AdminSecret string
	// BudgetPassword gates the CashFlux app at /budget/: entering it grants a "full" session (your
	// synced budget), while a guest bypass grants a local-only session. Defaults to AdminPassword;
	// empty disables the gate entirely (the app is served open).
	BudgetPassword string
	// BaseURL is the public origin, used to build absolute RSS links.
	BaseURL string
	// OpenAIKey enables the résumé tailoring tool. Empty disables it. Secret — env only.
	OpenAIKey string
	// OpenAIModel is the chat model used for tailoring (default gpt-4o-mini).
	OpenAIModel string
	// CashFluxDataDir is where the embedded CashFlux data-sync engine keeps its encrypted server-side
	// SQLite store (cashflux-server.db). CashFlux's SyncService runs in-process over gRPC-over-
	// WebSocket at /grpc, making multi-device sync a managed service. Empty disables the embedded
	// sync engine (frontend-only hosting).
	CashFluxDataDir string
}

// Load reads configuration from the environment, applying sane local defaults.
func Load() Config {
	return Config{
		Addr:           env("LISTEN_ADDR", "127.0.0.1:8095"),
		DBPath:         env("DB_PATH", "web/data/site.db"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		AdminUsername:  env("ADMIN_USERNAME", "cam"),
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),
		AdminSecret:    os.Getenv("ADMIN_SECRET"),
		BudgetPassword: env("BUDGET_PASSWORD", os.Getenv("ADMIN_PASSWORD")),
		BaseURL:        env("BASE_URL", "http://127.0.0.1:8095"),
		OpenAIKey:       os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:     env("OPENAI_MODEL", "gpt-4o-mini"),
		CashFluxDataDir: env("CASHFLUX_DATA_DIR", "web/data/cashflux"),
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
