package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/tifacusil/go-project-management-service/internal/config"
	"github.com/tifacusil/go-project-management-service/internal/discovery"
	"github.com/tifacusil/go-project-management-service/internal/handler"
	"github.com/tifacusil/go-project-management-service/internal/observability"
	"github.com/tifacusil/go-project-management-service/internal/repository"
	"github.com/tifacusil/go-project-management-service/internal/seed"
	"github.com/tifacusil/go-project-management-service/internal/service"
)

func main() {
	cfg := config.Load()
	// El orquestador puede inyectar INSTANCE_ID; si no, se genera uno único.
	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID = cfg.ServiceName + "-" + uuid.NewString()
	}

	logger := observability.NewLogger(cfg.ServiceName, instanceID)
	slog.SetDefault(logger)

	if err := run(cfg, logger, instanceID); err != nil {
		logger.Error("fatal", "event", "startup_failed", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger, instanceID string) error {
	// Root context cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Apply schema migrations as root before serving (goose = fuente de verdad).
	// Runtime queries luego usan el usuario de mínimos privilegios.
	logger.Info("running migrations", "event", "db_migrate")
	if err := config.MigrateUp(cfg); err != nil {
		return err
	}

	logger.Info("connecting to mysql", "event", "db_connect", "host", cfg.DBHost, "db", cfg.DBName, "user", cfg.DBUser)
	db, err := config.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	// Wire dependencies by hand (no DI container).
	projectRepo := repository.NewProjectRepository(db)
	sensorRepo := repository.NewSensorRepository(db)
	equipmentRepo := repository.NewEquipmentRepository(db)

	// Seed inicial idempotente (DataInitializer). Falla suave: un error de seed
	// no debe impedir que el servicio arranque (paridad Java).
	if cfg.SeedEnabled {
		if err := seed.Run(ctx, projectRepo, sensorRepo, logger); err != nil {
			logger.Warn("seed inicial omitido", "event", "seed", "err", err)
		}
	}

	projectSvc := service.NewProjectService(projectRepo, sensorRepo)
	sensorSvc := service.NewSensorService(projectRepo, sensorRepo)
	equipmentSvc := service.NewEquipmentService(projectRepo, equipmentRepo)

	validate := validator.New(validator.WithRequiredStructEnabled())
	projectHandler := handler.NewProjectHandler(projectSvc, validate)
	sensorHandler := handler.NewSensorHandler(sensorSvc)
	equipmentHandler := handler.NewEquipmentHandler(equipmentSvc)

	metrics := observability.NewMetrics()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(observability.RequestLogger(logger))
	r.Use(metrics.Middleware)
	r.Use(middleware.Recoverer)

	// Métricas Prometheus (CLAUDE.md §5): throughput + latencia por percentil;
	// CPU/memoria vía colectores default (go_*/process_*).
	r.Handle("/metrics", metrics.Handler())

	// Routes under /api/v1 (baseline parity). Sensors/equipment nest inside the
	// project's /{project_id} block so leaf + nested routes coexist.
	projectsRouter := projectHandler.Routes(func(pr chi.Router) {
		pr.Mount("/sensors", sensorHandler.Routes())
		pr.Mount("/equipment", equipmentHandler.Routes())
	})
	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/projects", projectsRouter)
	})

	// Health check: real DB connectivity (CLAUDE.md §1.6). RabbitMQ check N/A —
	// this service is REST/CRUD only, no broker consumer/producer.
	r.Get(cfg.HealthPath, func(w http.ResponseWriter, req *http.Request) {
		log := observability.FromContext(req.Context())
		status := http.StatusOK
		body := map[string]string{"status": "UP"}
		if err := db.PingContext(req.Context()); err != nil {
			log.Warn("health degraded", "event", "health_check", "err", err)
			status = http.StatusServiceUnavailable
			body = map[string]string{"status": "DOWN", "db": "unreachable"}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Serve in background; push error to channel.
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("http listening", "event", "http_start", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	// Register in Consul (CLAUDE.md §6.5). fail-fast=false: a Consul blip must
	// not crash-loop the service, so registration errors are logged, not fatal.
	//
	// TODO(M8-verify): deregister-on-shutdown está verificado por lectura pero no
	// ejercitado por señal — en Windows/Git Bash `kill -INT` no llega a un exe
	// nativo. Confirmar con SIGINT real: `Ctrl+C` en una terminal Windows con la
	// infra levantada, o corriendo el binario en Linux/contenedor. Debe loguear
	// "consul deregistered" y desaparecer de /v1/agent/services.
	reg := registerConsul(cfg, logger, instanceID)
	if reg != nil {
		defer func() {
			if err := reg.Deregister(); err != nil {
				logger.Warn("consul deregister failed", "event", "consul_deregister", "err", err)
			} else {
				logger.Info("consul deregistered", "event", "consul_deregister")
			}
		}()
	}

	// Block until signal or serve error.
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received", "event", "shutdown_begin")
	}

	// Graceful drain, 15s budget.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("shutdown complete", "event", "shutdown_done")
	return nil
}

// registerConsul registers the service or returns nil on failure (fail-fast=false).
func registerConsul(cfg config.Config, logger *slog.Logger, instanceID string) *discovery.Registration {
	if !cfg.ConsulEnabled {
		logger.Info("consul disabled", "event", "consul_skip")
		return nil
	}

	port, err := portFromAddr(cfg.HTTPAddr)
	if err != nil {
		logger.Warn("cannot derive port for consul", "event", "consul_register", "err", err)
		return nil
	}

	healthURL := "http://" + net.JoinHostPort(cfg.AdvertiseHost, strconv.Itoa(port)) + cfg.HealthPath
	reg, err := discovery.Register(discovery.Options{
		ConsulAddr:  cfg.ConsulAddr,
		Token:       cfg.ConsulToken,
		ServiceName: cfg.ServiceName,
		InstanceID:  instanceID,
		Address:     cfg.AdvertiseHost,
		Port:        port,
		HealthURL:   healthURL,
		Interval:    cfg.ConsulHealthInterval,
		Tags:        []string{"version-" + cfg.ServiceVersion, "lang-go", "env-" + cfg.Env},
	})
	if err != nil {
		logger.Warn("consul registration failed; continuing", "event", "consul_register", "err", err)
		return nil
	}
	logger.Info("consul registered", "event", "consul_register", "consul", cfg.ConsulAddr, "id", instanceID)
	return reg
}

// portFromAddr extracts the numeric port from a listen addr like ":8084" or "0.0.0.0:8084".
func portFromAddr(addr string) (int, error) {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(portStr)
}
