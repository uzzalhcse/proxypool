package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	RedisURL       string
	WARPCount      int
	WARPBasePort   int
	ProxyAuthUser  string
	ProxyAuthPass  string
	HealthInterval time.Duration
	IPMaxAge       time.Duration
	IPCooldown     time.Duration
	APIPort        int
	APIAuthToken   string
	LBPort         int
}

func Load() *Config {
	cfg := &Config{
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6380"),
		WARPCount:      getEnvInt("WARP_COUNT", 2),
		WARPBasePort:   getEnvInt("WARP_BASE_PORT", 40001),
		ProxyAuthUser:  getEnv("PROXY_AUTH_USER", "testuser"),
		ProxyAuthPass:  getEnv("PROXY_AUTH_PASS", "testpass123"),
		HealthInterval: getEnvDuration("HEALTH_INTERVAL", 30*time.Second),
		IPMaxAge:       getEnvDuration("IP_MAX_AGE", 24*time.Hour),
		IPCooldown:     getEnvDuration("IP_COOLDOWN", 6*time.Hour),
		APIPort:        getEnvInt("API_PORT", 8080),
		APIAuthToken:   getEnv("API_AUTH_TOKEN", "secret123"),
		LBPort:         getEnvInt("LB_PORT", 40000),
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
