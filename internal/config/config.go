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

	InstanceID    string // id de instancia inyectado por el orquestador (vacío => generado)
	HTTPAddr      string // listen addr, e.g. ":8080"
	AdvertiseHost string // host/ip anunciado a Consul (prefer-ip-address)
	HealthPath    string

	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string

	// Migration credentials: goose corre como root (DDL) al startup, separado
	// del usuario runtime de mínimos privilegios (CLAUDE.md §2.4).
	DBRootUser string
	DBRootPass string

	DBMaxOpenConns int
	DBMaxIdleConns int

	// Consul service discovery (CLAUDE.md §4, §6.5).
	ConsulEnabled        bool
	ConsulAddr           string // host:port
	ConsulToken          string
	ConsulHealthInterval string

	// Seed inicial (puerto de DataInitializer). Default true (paridad Java).
	SeedEnabled bool
}

// Load reads .env (if present) then env vars. Env wins over .env.
func Load() Config {
	// Ignore error: .env optional in prod (vars come from environment / Consul KV).
	_ = godotenv.Load()

	return Config{
		ServiceName:    env("SERVICE_NAME", "project-management-service"),
		ServiceVersion: env("SERVICE_VERSION", "0.1.0"),
		Env:            env("ENV", "dev"),
		InstanceID:     os.Getenv("INSTANCE_ID"), // del orquestador (compose/k8s); vacío => se genera

		// HTTP_ADDR tiene prioridad si está seteado; si no, se construye
		// desde HTTP_PORT (contrato del compose de infra). Default :8080.
		HTTPAddr:      httpAddr(),
		AdvertiseHost: env("ADVERTISE_HOST", "127.0.0.1"),
		HealthPath:    env("HEALTH_PATH", "/health"),

		DBHost:         env("DB_HOST", "127.0.0.1"),
		DBPort:         env("DB_PORT", "3306"),
		DBUser:         env("DB_USER", "proj-manager"),
		DBPass:         env("DB_PASS", ""),
		DBName:         env("DB_NAME", "project_management_db"),
		DBRootUser:     env("DB_ROOT_USER", "root"),
		DBRootPass:     env("DB_ROOT_PASS", ""),
		DBMaxOpenConns: envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: envInt("DB_MAX_IDLE_CONNS", 25),

		ConsulEnabled:        envBool("CONSUL_ENABLED", true),
		ConsulAddr:           env("CONSUL_HOST", "localhost") + ":" + env("CONSUL_PORT", "8500"),
		ConsulToken:          env("CONSUL_TOKEN", ""),
		ConsulHealthInterval: env("CONSUL_HEALTH_INTERVAL", "10s"), // CLAUDE.md §4 (Java baseline usaba 15s)
		SeedEnabled:          envBool("SEED_ENABLED", true),
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
	return c.dsnFor(c.DBUser, c.DBPass, nil)
}

// RootDSN builds the connection string used only for migrations (goose up as
// root, DDL). multiStatements habilitado para statements de migración.
func (c Config) RootDSN() string {
	return c.dsnFor(c.DBRootUser, c.DBRootPass, map[string]string{"multiStatements": "true"})
}

func (c Config) dsnFor(user, pass string, extra map[string]string) string {
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = pass
	cfg.Net = "tcp"
	cfg.Addr = c.DBHost + ":" + c.DBPort
	cfg.DBName = c.DBName
	cfg.ParseTime = true // scan DATETIME into time.Time
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	for k, v := range extra {
		cfg.Params[k] = v
	}
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

// httpAddr resuelve la dirección de escucha. HTTP_ADDR (e.g. ":8080") gana si
// está seteado; si no, se construye desde HTTP_PORT (default 8080) para cumplir
// el contrato del compose de infra.
func httpAddr() string {
	if a := os.Getenv("HTTP_ADDR"); a != "" {
		return a
	}
	return ":" + env("HTTP_PORT", "8080")
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
