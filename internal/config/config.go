package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr    string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// Load reads configuration from environment and .env file.
func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		ServerAddr: ":8080",
		RedisAddr:  "127.0.0.1:6379",
		RedisDB:    0,
	}

	if v := os.Getenv("PORT"); v != "" {
		cfg.ServerAddr = ":" + v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.RedisAddr = v
	}
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")
	if v := os.Getenv("REDIS_DB"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.RedisDB = i
		}
	}
	return cfg
}
