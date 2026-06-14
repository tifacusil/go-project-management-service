package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// SensorAPI is what the handler needs from the sensor service.
type SensorAPI interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Sensor, error)
	GetByID(ctx context.Context, projectID, sensorID string) (*domain.Sensor, error)
	Create(ctx context.Context, projectID string, body *domain.Sensor) (*domain.Sensor, error)
	Update(ctx context.Context, projectID, sensorID string, body *domain.Sensor) (*domain.Sensor, error)
	SoftDelete(ctx context.Context, projectID, sensorID string) error
}

// SensorHandler serves /projects/{project_id}/sensors.
type SensorHandler struct {
	svc SensorAPI
}

func NewSensorHandler(svc SensorAPI) *SensorHandler {
	return &SensorHandler{svc: svc}
}

// Routes returns the sub-router; mount under a parent route carrying {project_id}.
func (h *SensorHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{sensor_id}", h.get)
	r.Put("/{sensor_id}", h.update)
	r.Delete("/{sensor_id}", h.delete)
	return r
}

func (h *SensorHandler) list(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "project_id")
	sensors, err := h.svc.ListByProject(r.Context(), pid)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sensors)
}

func (h *SensorHandler) get(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "project_id")
	sid := chi.URLParam(r, "sensor_id")
	s, err := h.svc.GetByID(r.Context(), pid, sid)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *SensorHandler) create(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "project_id")
	var body domain.Sensor
	if !decodeJSON(w, r, &body) {
		return
	}
	created, err := h.svc.Create(r.Context(), pid, &body)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *SensorHandler) update(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "project_id")
	sid := chi.URLParam(r, "sensor_id")
	var body domain.Sensor
	if !decodeJSON(w, r, &body) {
		return
	}
	updated, err := h.svc.Update(r.Context(), pid, sid, &body)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *SensorHandler) delete(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "project_id")
	sid := chi.URLParam(r, "sensor_id")
	if err := h.svc.SoftDelete(r.Context(), pid, sid); err != nil {
		failErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
