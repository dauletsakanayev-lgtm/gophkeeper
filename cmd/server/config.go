package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// serverConfig - итоговая конфигурация сервера.
// Приоритет источников env > флаг > дефолт.
type serverConfig struct {
	Addr        string        // адрес HTTP-сервера, e.g. "8080"
	DatabaseDNS string        // PostgreSQL DNS
	JWTSecret   string        // секрет для поиска JWT (HS256)
	JWTTTL      time.Duration // TTL access-токена
	LogLevel    string        // debug/info/warn/error
}

// parseConfig собирает конфигурацию сервера.
// Пустой DatabaseDNS или JWTSecret считается ошибкой конгифурации.
func parseCongif() (serverConfig, error) {
	addr := flag.String("a", ":8081", "адрес HTTP сервера (host:port)")
	dns := flag.String("d", "", "DNS PostgreSQL (обязателен)")
	jwtSecret := flag.String("jwt-secret", "", "секрет для подписи JWT (обязателен)")
	jwtTTL := flag.Duration("jwt-ttl", 24*time.Hour, "TTL access-токена")
	logLevel := flag.String("l", "info", "уровень логирования")
	flag.Parse()

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		*addr = v
	}

	if v, ok := os.LookupEnv("DATABASE_DNS"); ok {
		*dns = v
	}

	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
		*jwtSecret = v
	}

	if v, ok := os.LookupEnv("JWT_TTL"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return serverConfig{}, fmt.Errorf("неверное значение JWT_TTL: %w", err)
		}
		*jwtTTL = d
	}

	if v, ok := os.LookupEnv("LOG_LEVEL"); ok {
		*logLevel = v
	}

	if *dns == "" {
		return serverConfig{}, fmt.Errorf("DATABASE_DNS (или флаг -d) обязателен")
	}
	if *jwtSecret == "" {
		return serverConfig{}, fmt.Errorf("JWT_SECRET (или флаг -jwt-secret) обязателен")
	}

	return serverConfig{
		Addr:        *addr,
		DatabaseDNS: *dns,
		JWTSecret:   *jwtSecret,
		JWTTTL:      *jwtTTL,
		LogLevel:    *logLevel,
	}, nil
}
