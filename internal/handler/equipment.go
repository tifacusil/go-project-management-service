package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// EquipmentAPI is what the handler needs from the equipment service.
type EquipmentAPI interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Equipment, error)
	Create(ctx context.Context, projectID string, body *domain.Equipment) (*domain.Equipment, error)
}

// EquipmentHandler serves /projects/{project_id}/equipment (list + create only).
type EquipmentHandler struct {
	svc EquipmentAPI
}

func NewEquipmentHandler(svc EquipmentAPI) *EquipmentHandler {
	return &EquipmentHandler{svc: svc}
}

// Routes returns the sub-router; mount under a parent route carrying {project_id}.
func (h *EquipmentHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	return r
}

func (h *EquipmentHandler) list(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "project_id")
	items, err := h.svc.ListByProject(r.Context(), pid)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *EquipmentHandler) create(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "project_id")
	var body domain.Equipment
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
