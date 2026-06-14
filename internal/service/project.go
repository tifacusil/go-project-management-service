// Package service holds business logic — validations + soft delete rules live
// here, not in handlers or SQL (CLAUDE.md: lógica de negocio en capa de servicio).
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/tifacusil/go-project-management-service/internal/domain"
	"github.com/tifacusil/go-project-management-service/internal/repository"
)

// projectRepo / sensorRepo: only the methods this service uses (testable seams).
type projectRepo interface {
	FindAll(ctx context.Context) ([]domain.Project, error)
	FindByID(ctx context.Context, id string) (*domain.Project, error)
	CountByID(ctx context.Context, id string) (int, error)
	Insert(ctx context.Context, p *domain.Project) error
	Update(ctx context.Context, p *domain.Project) error
	SoftDelete(ctx context.Context, id string) error
}

type sensorRepo interface {
	FindByProjectID(ctx context.Context, projectID string) ([]domain.Sensor, error)
	CountByProjectIDAndStatus(ctx context.Context, projectID, status string) (int, error)
}

// ProjectService mirrors the Java ProjectService.
type ProjectService struct {
	projects projectRepo
	sensors  sensorRepo
}

func NewProjectService(p projectRepo, s sensorRepo) *ProjectService {
	return &ProjectService{projects: p, sensors: s}
}

func (s *ProjectService) ListAll(ctx context.Context) ([]domain.Project, error) {
	return s.projects.FindAll(ctx)
}

// GetByIDWithSensors loads the project and attaches its sensors. 404 if missing.
func (s *ProjectService) GetByIDWithSensors(ctx context.Context, id string) (*domain.Project, error) {
	p, err := s.projects.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound("Proyecto no existe: %s", id)
		}
		return nil, err
	}
	sensors, err := s.sensors.FindByProjectID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Sensors = sensors
	return p, nil
}

// Create validates, rejects duplicate IDs, inserts, returns the stored project.
func (s *ProjectService) Create(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	if p == nil {
		return nil, invalid("Body requerido")
	}
	if strings.TrimSpace(p.ProjectID) == "" {
		return nil, invalid("project_id es requerido")
	}
	if strings.TrimSpace(p.Name) == "" {
		return nil, invalid("name es requerido")
	}

	n, err := s.projects.CountByID(ctx, p.ProjectID)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, conflict("project_id ya existe: %s", p.ProjectID)
	}

	if err := s.projects.Insert(ctx, p); err != nil {
		return nil, err
	}
	return s.GetByIDWithSensors(ctx, p.ProjectID)
}

// Update merges body over existing (nil fields keep prior value), then persists.
func (s *ProjectService) Update(ctx context.Context, id string, body *domain.Project) (*domain.Project, error) {
	if body == nil {
		return nil, invalid("Body requerido")
	}
	if body.ProjectID != "" && body.ProjectID != id {
		return nil, invalid("project_id del body no coincide con la URL")
	}

	existing, err := s.projects.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound("Proyecto no existe: %s", id)
		}
		return nil, err
	}

	merged := &domain.Project{
		ProjectID:   id,
		Name:        existing.Name,
		LocationLat: existing.LocationLat,
		LocationLng: existing.LocationLng,
		Status:      existing.Status,
	}
	if strings.TrimSpace(body.Name) != "" {
		merged.Name = body.Name
	}
	if body.LocationLat != nil {
		merged.LocationLat = body.LocationLat
	}
	if body.LocationLng != nil {
		merged.LocationLng = body.LocationLng
	}
	if body.Status != "" {
		merged.Status = body.Status
	}

	if err := s.projects.Update(ctx, merged); err != nil {
		return nil, err
	}
	return s.GetByIDWithSensors(ctx, id)
}

// SoftDelete sets status=inactive. Blocks if any sensor is still active.
func (s *ProjectService) SoftDelete(ctx context.Context, id string) error {
	if _, err := s.projects.FindByID(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return notFound("Proyecto no existe: %s", id)
		}
		return err
	}

	active, err := s.sensors.CountByProjectIDAndStatus(ctx, id, domain.StatusActive)
	if err != nil {
		return err
	}
	if active > 0 {
		return invalid("No se permite eliminar el proyecto con sensores activos")
	}

	return s.projects.SoftDelete(ctx, id)
}
