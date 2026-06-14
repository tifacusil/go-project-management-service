package service

import (
	"context"
	"strings"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// allowedEquipmentTypes mirrors EquipmentService.ALLOWED_TYPES (Java).
var allowedEquipmentTypes = map[string]bool{
	"DTU": true, "solar_panel": true, "battery": true, "chassis": true, "camera": true,
}

type equipmentStore interface {
	FindByProjectID(ctx context.Context, projectID string) ([]domain.Equipment, error)
	Insert(ctx context.Context, e *domain.Equipment) error
}

// EquipmentService mirrors the Java EquipmentService (list + create only).
type EquipmentService struct {
	projects  projectExistence
	equipment equipmentStore
}

func NewEquipmentService(p projectExistence, e equipmentStore) *EquipmentService {
	return &EquipmentService{projects: p, equipment: e}
}

func (s *EquipmentService) ListByProject(ctx context.Context, projectID string) ([]domain.Equipment, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	return s.equipment.FindByProjectID(ctx, projectID)
}

func (s *EquipmentService) Create(ctx context.Context, projectID string, body *domain.Equipment) (*domain.Equipment, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	if body == nil {
		return nil, invalid("Body requerido")
	}
	if body.ProjectID != "" && body.ProjectID != projectID {
		return nil, invalid("project_id del body no coincide con la URL")
	}
	if strings.TrimSpace(body.EquipmentID) == "" {
		return nil, invalid("equipment_id es requerido")
	}
	if body.Type == nil || strings.TrimSpace(*body.Type) == "" {
		return nil, invalid("type es requerido")
	}
	t := strings.TrimSpace(*body.Type)
	if !allowedEquipmentTypes[t] {
		return nil, invalid("type inválido: %s", *body.Type)
	}

	// No dup-id check: baseline EquipmentService omits it (relies on PK).
	status := body.Status
	if status == "" {
		status = domain.StatusActive
	}
	toInsert := &domain.Equipment{
		EquipmentID: body.EquipmentID,
		ProjectID:   projectID,
		Type:        &t,
		Model:       body.Model,
		Serial:      body.Serial,
		Status:      status,
	}
	if err := s.equipment.Insert(ctx, toInsert); err != nil {
		return nil, err
	}
	return toInsert, nil // baseline returns the inserted struct, no re-fetch
}

func (s *EquipmentService) ensureProject(ctx context.Context, projectID string) error {
	n, err := s.projects.CountByID(ctx, projectID)
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound("Proyecto no existe: %s", projectID)
	}
	return nil
}
