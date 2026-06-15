package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics expone los colectores HTTP (CLAUDE.md §5: throughput + latencia por
// percentil). CPU y memoria residente los aporta el registry default de
// Prometheus (go_* y process_*).
type Metrics struct {
	reqTotal    *prometheus.CounterVec
	reqDuration *prometheus.HistogramVec
}

// NewMetrics registra los colectores en el registry default.
func NewMetrics() *Metrics {
	m := &Metrics{
		reqTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total de requests HTTP por método, ruta y código.",
		}, []string{"method", "route", "status"}),
		reqDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Latencia de requests HTTP. Buckets default cubren p50..p99.",
			// Buckets default de Prometheus (.005s..10s) → percentiles vía histogram_quantile.
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
	}
	prometheus.MustRegister(m.reqTotal, m.reqDuration)
	return m
}

// Middleware mide cada request: incrementa el contador y observa la latencia.
// Usa el patrón de ruta de chi (ej. /api/v1/projects/{project_id}) para evitar
// explosión de cardinalidad por IDs concretos.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(ww.Status())
		m.reqTotal.WithLabelValues(r.Method, route, status).Inc()
		m.reqDuration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())
	})
}

// Handler sirve el endpoint /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}
