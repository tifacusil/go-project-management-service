package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-retry"

	_ "github.com/go-sql-driver/mysql"
)

// Config holds runtime config sourced from env / .env (CLAUDE.md §1.7 — nada hardcodeado).
type Config struct {
	// Service identity (CLAUDE.md §6.5 — naming + tags para Consul).
	ServiceName    string
	ServiceVersion string
	Env            string

	HTTPAddr      string // listen addr, e.g. ":8084"
	AdvertiseHost string // host/ip anunciado a Consul (prefer-ip-address)
	HealthPath    string

	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string

	DBMaxOpenConns int
	DBMaxIdleConns int

	// Consul service discovery (CLAUDE.md §4, §6.5).
	ConsulEnabled        bool
	ConsulAddr           string // host:port
	ConsulToken          string
	ConsulHealthInterval string
}

// Load reads .env (if present) then env vars. Env wins over .env.
func Load() Config {
	// Ignore error: .env optional in prod (vars come from environment / Consul KV).
	_ = godotenv.Load()

	return Config{
		ServiceName:    env("SERVICE_NAME", "project-management-service"),
		ServiceVersion: env("SERVICE_VERSION", "0.1.0"),
		Env:            env("ENV", "dev"),

		HTTPAddr:      env("HTTP_ADDR", ":8084"),
		AdvertiseHost: env("ADVERTISE_HOST", "127.0.0.1"),
		HealthPath:    env("HEALTH_PATH", "/health"),

		DBHost:         env("DB_HOST", "127.0.0.1"),
		DBPort:         env("DB_PORT", "3306"),
		DBUser:         env("DB_USER", "proj-manager"),
		DBPass:         env("DB_PASS", ""),
		DBName:         env("DB_NAME", "project_management_db"),
		DBMaxOpenConns: envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: envInt("DB_MAX_IDLE_CONNS", 25),

		ConsulEnabled:        envBool("CONSUL_ENABLED", true),
		ConsulAddr:           env("CONSUL_HOST", "localhost") + ":" + env("CONSUL_PORT", "8500"),
		ConsulToken:          env("CONSUL_TOKEN", ""),
		ConsulHealthInterval: env("CONSUL_HEALTH_INTERVAL", "10s"), // CLAUDE.md §4 (Java baseline usaba 15s)
	}
}

// DSN builds the go-sql-driver/mysql connection string.
//
// TODO(SE-practice): database-per-service (CLAUDE.md §1.2). El sistema baseline
// comparte un único MySQL entre los 6 servicios (Shared Persistency, bad smell —
// Taibi & Lenarduzzi 2018). Este servicio ya apunta a su propio schema
// `project_management_db` con el usuario de mínimos privilegios `proj-manager`
// (ver scripts/01_min_privileges.sql). Cutover Strangler Fig pendiente:
// extraer a instancia MySQL dedicada por servicio.
func (c Config) DSN() string {
	cfg := mysql.NewConfig()
	cfg.User = c.DBUser
	cfg.Passwd = c.DBPass
	cfg.Net = "tcp"
	cfg.Addr = c.DBHost + ":" + c.DBPort
	cfg.DBName = c.DBName
	cfg.ParseTime = true // scan DATETIME into time.Time
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	return cfg.FormatDSN()
}

// Connect opens the pool and pings with retry/backoff until ready.
func Connect(ctx context.Context, c Config) (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", c.DSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(c.DBMaxOpenConns)
	db.SetMaxIdleConns(c.DBMaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Backoff: 1s base, x2, cap 10s, max 30s total.
	b := retry.WithMaxDuration(30*time.Second, retry.WithCappedDuration(10*time.Second, retry.NewExponential(time.Second)))
	pingErr := retry.Do(ctx, b, func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return retry.RetryableError(err)
		}
		return nil
	})
	if pingErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", pingErr)
	}

	return db, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
