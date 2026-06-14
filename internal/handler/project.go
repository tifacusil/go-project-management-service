// Package handler holds HTTP handlers (equivalent to @RestController).
package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// ProjectAPI defines what the handler needs from the service (testable seam).
type ProjectAPI interface {
	ListAll(ctx context.Context) ([]domain.Project, error)
	GetByIDWithSensors(ctx context.Context, id string) (*domain.Project, error)
	Create(ctx context.Context, p *domain.Project) (*domain.Project, error)
	Update(ctx context.Context, id string, body *domain.Project) (*domain.Project, error)
	SoftDelete(ctx context.Context, id string) error
}

// ProjectHandler wires routes for /projects (contract: contracts/openapi/project-management.openapi.yaml).
type ProjectHandler struct {
	svc      ProjectAPI
	validate *validator.Validate
}

func NewProjectHandler(svc ProjectAPI, v *validator.Validate) *ProjectHandler {
	return &ProjectHandler{svc: svc, validate: v}
}

// Routes mounts the project sub-router. The per-project leaf routes and any
// nested resources (sensors, equipment) must share ONE Route("/{project_id}")
// block — registering a separate mount on the same param node shadows the leaf
// methods (→ 404). `extend` runs inside that block; pass nil for none.
func (h *ProjectHandler) Routes(extend func(pr chi.Router)) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Route("/{project_id}", func(pr chi.Router) {
		pr.Get("/", h.get)
		pr.Put("/", h.update)
		pr.Delete("/", h.delete)
		if extend != nil {
			extend(pr)
		}
	})
	return r
}

func (h *ProjectHandler) list(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.ListAll(r.Context())
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "project_id")
	p, err := h.svc.GetByIDWithSensors(r.Context(), id)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) create(w http.ResponseWriter, r *http.Request) {
	var p domain.Project
	if !decodeJSON(w, r, &p) {
		return
	}
	if err := h.validate.Struct(&p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := h.svc.Create(r.Context(), &p)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *ProjectHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "project_id")
	var body domain.Project
	if !decodeJSON(w, r, &body) {
		return
	}
	updated, err := h.svc.Update(r.Context(), id, &body)
	if err != nil {
		failErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *ProjectHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "project_id")
	if err := h.svc.SoftDelete(r.Context(), id); err != nil {
		failErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
