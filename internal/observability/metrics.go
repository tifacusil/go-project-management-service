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

// httpBuckets son los cortes del histograma de latencia HTTP, compartidos por
// los 4 servicios Go y alineados al SLO declarado en application.yml de los
// servicios Spring (Sistema A): 5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s, mas
// 2.5s/5s para no perder resolucion en colas bajo carga. prometheus.DefBuckets
// produciria percentiles no comparables contra Sistema A via histogram_quantile.
var httpBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

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
			Name:    "http_request_duration_seconds",
			Help:    "Latencia de requests HTTP, buckets alineados al SLO de Sistema A -> percentiles via histogram_quantile.",
			Buckets: httpBuckets,
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
