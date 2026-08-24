// Package config loads the process configuration from the environment. Every
// knob that affects the runnable entry point is expressed here so that the
// service, the smoke script and the Docker image share a single, documented
// configuration surface.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the process configuration.
type Config struct {
	Addr        string
	DBPath      string
	LogRequests bool
	Shutdown    time.Duration
}

// Load reads the configuration from the environment, applying documented
// defaults for any unset value.
func Load() Config {
	return Config{
		Addr:        env("ADDR", ":8080"),
		DBPath:      env("DB_PATH", "mycocycle.db"),
		LogRequests: envBool("LOG_REQUESTS", true),
		Shutdown:    envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return time.Duration(n) * time.Second
	}
	if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
		return d
	}
	return def
}
