// @navigator-project: navigator
// @navigator-path: internal/config/env.go
package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultHTTPAddr  = ":8084"
	DefaultNexusAddr = "http://127.0.0.1:8080"
	DefaultAtlasAddr = "http://127.0.0.1:8081"
)

func EnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ExpandHome(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
