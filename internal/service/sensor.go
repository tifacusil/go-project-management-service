package service

import (
	"context"
	"errors"
	"strings"

	"github.com/tifacusil/go-project-management-service/internal/domain"
	"github.com/tifacusil/go-project-management-service/internal/repository"
)

// allowedSensorTypes mirrors SensorService.ALLOWED_TYPES (Java).
var allowedSensorTypes = map[string]bool{
	"gnss": true, "rainfall": true, "strain": true,
	"soil_pressure": true, "deep_displacement": true, "camera": true,
}

// projectExistence: only the check sensor/equipment services need.
type projectExistence interface {
	CountByID(ctx context.Context, id string) (int, error)
}

type sensorStore interface {
	FindByProjectID(ctx context.Context, projectID string) ([]domain.Sensor, error)
	FindByIDAndProjectID(ctx context.Context, projectID, sensorID string) (*domain.Sensor, error)
	CountByID(ctx context.Context, sensorID string) (int, error)
	Insert(ctx context.Context, s *domain.Sensor) error
	Update(ctx context.Context, s *domain.Sensor) error
	SoftDelete(ctx context.Context, projectID, sensorID string) error
}

// SensorService mirrors the Java SensorService.
type SensorService struct {
	projects projectExistence
	sensors  sensorStore
}

func NewSensorService(p projectExistence, s sensorStore) *SensorService {
	return &SensorService{projects: p, sensors: s}
}

func (s *SensorService) ListByProject(ctx context.Context, projectID string) ([]domain.Sensor, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	return s.sensors.FindByProjectID(ctx, projectID)
}

func (s *SensorService) GetByID(ctx context.Context, projectID, sensorID string) (*domain.Sensor, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	sensor, err := s.sensors.FindByIDAndProjectID(ctx, projectID, sensorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound("Sensor no existe: %s", sensorID)
		}
		return nil, err
	}
	return sensor, nil
}

func (s *SensorService) Create(ctx context.Context, projectID string, body *domain.Sensor) (*domain.Sensor, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	if body == nil {
		return nil, invalid("Body requerido")
	}
	if body.ProjectID != "" && body.ProjectID != projectID {
		return nil, invalid("project_id del body no coincide con la URL")
	}
	if strings.TrimSpace(body.SensorID) == "" {
		return nil, invalid("sensor_id es requerido")
	}
	if strings.TrimSpace(body.SensorType) == "" {
		return nil, invalid("sensor_type es requerido")
	}
	t := strings.TrimSpace(body.SensorType)
	if !allowedSensorTypes[t] {
		return nil, invalid("sensor_type inválido: %s", body.SensorType)
	}

	n, err := s.sensors.CountByID(ctx, body.SensorID)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, conflict("sensor_id ya existe: %s", body.SensorID)
	}

	toInsert := &domain.Sensor{
		SensorID:     body.SensorID,
		ProjectID:    projectID,
		SensorType:   t,
		Model:        body.Model,
		InstallLat:   body.InstallLat,
		InstallLng:   body.InstallLng,
		InstallDepth: body.InstallDepth,
		Status:       body.Status,
		InstalledAt:  body.InstalledAt,
	}
	if err := s.sensors.Insert(ctx, toInsert); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, projectID, body.SensorID)
}

func (s *SensorService) Update(ctx context.Context, projectID, sensorID string, body *domain.Sensor) (*domain.Sensor, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	if body == nil {
		return nil, invalid("Body requerido")
	}
	if body.ProjectID != "" && body.ProjectID != projectID {
		return nil, invalid("project_id del body no coincide con la URL")
	}
	if body.SensorID != "" && body.SensorID != sensorID {
		return nil, invalid("sensor_id del body no coincide con la URL")
	}

	existing, err := s.sensors.FindByIDAndProjectID(ctx, projectID, sensorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound("Sensor no existe: %s", sensorID)
		}
		return nil, err
	}

	newType := existing.SensorType
	if strings.TrimSpace(body.SensorType) != "" {
		newType = strings.TrimSpace(body.SensorType)
	}
	if newType != "" && !allowedSensorTypes[newType] {
		return nil, invalid("sensor_type inválido: %s", body.SensorType)
	}

	merged := &domain.Sensor{
		ProjectID:    projectID,
		SensorID:     sensorID,
		SensorType:   newType,
		Model:        pickStr(body.Model, existing.Model),
		InstallLat:   pickF64(body.InstallLat, existing.InstallLat),
		InstallLng:   pickF64(body.InstallLng, existing.InstallLng),
		InstallDepth: pickF64(body.InstallDepth, existing.InstallDepth),
		Status:       existing.Status,
		InstalledAt:  pickTime(body.InstalledAt, existing.InstalledAt),
	}
	if body.Status != "" {
		merged.Status = body.Status
	}

	if err := s.sensors.Update(ctx, merged); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, projectID, sensorID)
}

func (s *SensorService) SoftDelete(ctx context.Context, projectID, sensorID string) error {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return err
	}
	if _, err := s.sensors.FindByIDAndProjectID(ctx, projectID, sensorID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return notFound("Sensor no existe: %s", sensorID)
		}
		return err
	}
	return s.sensors.SoftDelete(ctx, projectID, sensorID)
}

func (s *SensorService) ensureProject(ctx context.Context, projectID string) error {
	n, err := s.projects.CountByID(ctx, projectID)
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound("Proyecto no existe: %s", projectID)
	}
	return nil
}
