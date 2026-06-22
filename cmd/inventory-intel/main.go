package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	"inventoryintel/internal/app"
)

func main() {
	loadDotEnv(".env")
	addr := envOrDefault("APP_ADDR", ":8080")
	dbPath := envOrDefault("APP_DB_PATH", "data/inventory-intel.db")

	application, err := app.New(app.Config{
		DBPath: dbPath,
		AI: app.AIConfig{
			Mode:    envOrDefault("AI_INSIGHTS_MODE", "simulation"),
			Model:   envOrDefault("OPENAI_MODEL", "gpt-5-nano"),
			BaseURL: envOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			APIKey:  strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	log.Printf("inventory intel listening on %s", addr)
	if err := http.ListenAndServe(addr, application.Routes()); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}
