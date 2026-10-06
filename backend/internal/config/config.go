package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Addr         string
	DataDir      string
	CookieSecure bool
	// Registration lets anyone create an account (every account is a full admin).
	Registration bool
}

func Load() Config {
	return Config{
		Addr:         env("RESTOS_ADDR", ":8080"),
		DataDir:      env("RESTOS_DATA_DIR", "./data"),
		CookieSecure: os.Getenv("RESTOS_COOKIE_SECURE") == "true",
		Registration: os.Getenv("RESTOS_REGISTRATION") == "true",
	}
}

func (c Config) DBPath() string   { return filepath.Join(c.DataDir, "restos.db") }
func (c Config) MediaDir() string { return filepath.Join(c.DataDir, "media") }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
