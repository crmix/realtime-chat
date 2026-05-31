package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Logger   LoggerConfig
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

type PostgresConfig struct {
	DSN             string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Channel  string
}

type JWTConfig struct {
	Secret string
	TTL    time.Duration
	Issuer string
}

type LoggerConfig struct {
	Level  string
	Format string
}

// Load reads configuration from the environment, optionally pre-loading a
// .env file when present in the working directory. Missing required values
// return an error so we fail fast at startup.
func Load() (Config, error) {
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	pgDSN := os.Getenv("POSTGRES_DSN")
	if pgDSN == "" {
		return Config{}, fmt.Errorf("POSTGRES_DSN is required")
	}

	cfg := Config{
		HTTP: HTTPConfig{
			Addr:            getEnv("HTTP_ADDR", ":8080"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			AllowedOrigins:  splitCSV(getEnv("HTTP_ALLOWED_ORIGINS", "*")),
		},
		Postgres: PostgresConfig{
			DSN:             pgDSN,
			MaxConns:        getInt("POSTGRES_MAX_CONNS", 10),
			MinConns:        getInt("POSTGRES_MIN_CONNS", 2),
			MaxConnLifetime: getDuration("POSTGRES_MAX_CONN_LIFETIME", time.Hour),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       getInt("REDIS_DB", 0),
			Channel:  getEnv("REDIS_CHANNEL", "chat.messages"),
		},
		JWT: JWTConfig{
			Secret: jwtSecret,
			TTL:    getDuration("JWT_TTL", 24*time.Hour),
			Issuer: getEnv("JWT_ISSUER", "realtime-chat"),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		if r == ' ' && cur == "" {
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
