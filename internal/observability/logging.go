// Package observability provides structured logging per CLAUDE.md §1.8 / §5:
// JSON logs with fields service, instance_id, trace_id, event.
package observability

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey int

const loggerKey ctxKey = iota

// NewLogger builds the base JSON logger carrying the always-on fields.
func NewLogger(service, instanceID string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With(
		slog.String("service", service),
		slog.String("instance_id", instanceID),
	)
}

// RequestLogger injects a request-scoped logger carrying trace_id (chi RequestID).
// Mount AFTER middleware.RequestID so the ID exists.
//
// trace_id correlates project-management HTTP requests; for the async ingest
// flow (Data Receiving → RabbitMQ → consumers) it should be the message_id
// (CLAUDE.md §5) — N/A here since this service is REST/CRUD only.
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := middleware.GetReqID(r.Context())
			l := base.With(slog.String("trace_id", traceID))
			ctx := context.WithValue(r.Context(), loggerKey, l)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext returns the request-scoped logger, or the default if absent.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
