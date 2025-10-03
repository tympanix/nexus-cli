package main

import (
	"os"
)

var (
	nexusURL  = getenv("NEXUS_URL", "http://localhost:8081")
	username  = getenv("NEXUS_USER", "admin")
	password  = getenv("NEXUS_PASS", "admin")
	quietMode = false
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func isatty() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
