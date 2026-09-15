
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {

	Port string

	DataPath string

	CycleMs int64


	SyncLeadMs int64

	DefaultSyncDurationMs int64


	AllowedOrigins []string


	StaticDir string
}

func Load() Config {
	cfg := Config{
		Port:                  getString("PORT", "8080"),
		DataPath:              getString("DATA_PATH", "data/state.json"),
		CycleMs:               int64(getInt("CYCLE_SECONDS", 5*60*60)) * 1000,
		SyncLeadMs:            int64(getInt("SYNC_LEAD_MS", 1200)),
		DefaultSyncDurationMs: int64(getInt("SYNC_DEFAULT_SECONDS", 10)) * 1000,
		AllowedOrigins:        splitAndTrim(getString("ALLOWED_ORIGINS", "*")),
		StaticDir:             getString("STATIC_DIR", ""),
	}

	if cfg.CycleMs <= 0 {
		log.Printf("config: CYCLE_SECONDS must be positive, falling back to 5 hours")
		cfg.CycleMs = int64(5 * time.Hour / time.Millisecond)
	}
	return cfg
}

func getString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func getInt(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		log.Printf("config: %s=%q is not a number, using %d", key, raw, fallback)
		return fallback
	}
	return n
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
