package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN     string
	RedisAddr string
	HTTPPort  string
	BaseURL   string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("warning: .env file not found, using environment variables")
	}

	cfg := &Config{
		DBDSN:     os.Getenv("DB_DSN"),
		RedisAddr: os.Getenv("REDIS_ADDR"),
		HTTPPort:  os.Getenv("HTTP_PORT"),
		BaseURL:   os.Getenv("BASE_URL"),
	}

	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	if cfg.RedisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR is required")
	}
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "8080"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:" + cfg.HTTPPort
	}

	return cfg, nil
}
