// Package seed carga datos iniciales (puerto de DataInitializer.java).
// Idempotente: solo siembra si la tabla está vacía. Inserta vía repos (no
// service) para saltar validaciones de negocio, igual que el baseline usa mappers.
package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

type projectStore interface {
	CountAll(ctx context.Context) (int, error)
	CountByID(ctx context.Context, id string) (int, error)
	Insert(ctx context.Context, p *domain.Project) error
}

type sensorStore interface {
	CountAll(ctx context.Context) (int, error)
	Insert(ctx context.Context, s *domain.Sensor) error
}

// Run siembra proyectos y sensores si están vacíos. Errores NO crashean el
// arranque: se devuelven para que el caller los loguee como warning (paridad Java).
func Run(ctx context.Context, projects projectStore, sensors sensorStore, log *slog.Logger) error {
	if err := seedProjects(ctx, projects, log); err != nil {
		return err
	}
	return seedSensors(ctx, projects, sensors, log)
}

func seedProjects(ctx context.Context, projects projectStore, log *slog.Logger) error {
	n, err := projects.CountAll(ctx)
	if err != nil {
		return fmt.Errorf("count projects: %w", err)
	}
	if n > 0 {
		return nil
	}
	for _, p := range seedProjectList() {
		if err := projects.Insert(ctx, &p); err != nil {
			return fmt.Errorf("seed project %s: %w", p.ProjectID, err)
		}
	}
	log.Info("seeded projects", "event", "seed", "count", len(seedProjectList()))
	return nil
}

func seedSensors(ctx context.Context, projects projectStore, sensors sensorStore, log *slog.Logger) error {
	n, err := sensors.CountAll(ctx)
	if err != nil {
		return fmt.Errorf("count sensors: %w", err)
	}
	if n > 0 {
		return nil
	}
	// Asegurar que los proyectos referenciados existan (por si projects no estaba vacío
	// pero sí faltaban estos). Idempotente por id.
	for _, p := range seedProjectList() {
		exists, err := projects.CountByID(ctx, p.ProjectID)
		if err != nil {
			return fmt.Errorf("check project %s: %w", p.ProjectID, err)
		}
		if exists == 0 {
			if err := projects.Insert(ctx, &p); err != nil {
				return fmt.Errorf("seed project %s: %w", p.ProjectID, err)
			}
		}
	}
	list := seedSensorList()
	for _, s := range list {
		if err := sensors.Insert(ctx, &s); err != nil {
			return fmt.Errorf("seed sensor %s: %w", s.SensorID, err)
		}
	}
	log.Info("seeded sensors", "event", "seed", "count", len(list))
	return nil
}

func seedProjectList() []domain.Project {
	return []domain.Project{
		{ProjectID: "shangtan", Name: "Shangtan Landslide", LocationLat: f(23.45), LocationLng: f(110.12), Status: domain.StatusActive},
		{ProjectID: "lingzhan", Name: "Lingzhan Landslide", LocationLat: f(24.12), LocationLng: f(106.56), Status: domain.StatusActive},
	}
}

func seedSensorList() []domain.Sensor {
	return []domain.Sensor{
		sensor("GXZM-DB01", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-DB02", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-DB03", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-DB04", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-DB05", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-DB06", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-DB07", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-DB08", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-BASE", "shangtan", "gnss", "Huace H3"),
		sensor("GXZM-RG01", "shangtan", "rainfall", "JFZ-01 digital rain gauge"),
		sensor("GXZM-CAM01", "shangtan", "camera", "Hikvision IDS-2DF843715X-AFW/SP"),
		sensor("GXZM-CAM02", "shangtan", "camera", "Hikvision IDS-2DF843715X-AFW/SP"),
		sensor("GXZM-CAM03", "shangtan", "camera", "Hikvision IDS-2DF843715X-AFW/SP"),
		sensor("LZ-DB01", "lingzhan", "gnss", "Huace H3"),
		sensor("LZ-DB02", "lingzhan", "gnss", "Huace H3"),
		sensor("LZ-DB03", "lingzhan", "gnss", "Huace H3"),
		sensor("LZ-BASE", "lingzhan", "gnss", "Huace H3"),
	}
}

func sensor(id, projectID, sensorType, model string) domain.Sensor {
	m := model
	return domain.Sensor{
		SensorID:   id,
		ProjectID:  projectID,
		SensorType: sensorType,
		Model:      &m,
		Status:     domain.StatusActive,
	}
}

func f(v float64) *float64 { return &v }
