package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// SensorRepository = SQL access for sensors. Mirrors SensorMapper.
type SensorRepository struct {
	db *sqlx.DB
}

func NewSensorRepository(db *sqlx.DB) *SensorRepository {
	return &SensorRepository{db: db}
}

const sensorCols = `sensor_id, project_id, sensor_type, model, install_lat, install_lng, install_depth, status, installed_at`

func (r *SensorRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.Sensor, error) {
	q := `SELECT ` + sensorCols + ` FROM sensors WHERE project_id = ? ORDER BY sensor_id`
	sensors := []domain.Sensor{}
	if err := r.db.SelectContext(ctx, &sensors, q, projectID); err != nil {
		return nil, fmt.Errorf("find sensors for project %s: %w", projectID, err)
	}
	return sensors, nil
}

func (r *SensorRepository) FindByIDAndProjectID(ctx context.Context, projectID, sensorID string) (*domain.Sensor, error) {
	q := `SELECT ` + sensorCols + ` FROM sensors WHERE project_id = ? AND sensor_id = ?`
	var s domain.Sensor
	if err := r.db.GetContext(ctx, &s, q, projectID, sensorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find sensor %s/%s: %w", projectID, sensorID, err)
	}
	return &s, nil
}

func (r *SensorRepository) CountAll(ctx context.Context) (int, error) {
	var n int
	if err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM sensors`); err != nil {
		return 0, fmt.Errorf("count all sensors: %w", err)
	}
	return n, nil
}

func (r *SensorRepository) CountByID(ctx context.Context, sensorID string) (int, error) {
	var n int
	q := `SELECT COUNT(*) FROM sensors WHERE sensor_id = ?`
	if err := r.db.GetContext(ctx, &n, q, sensorID); err != nil {
		return 0, fmt.Errorf("count sensor %s: %w", sensorID, err)
	}
	return n, nil
}

func (r *SensorRepository) CountByProjectIDAndStatus(ctx context.Context, projectID, status string) (int, error) {
	var n int
	q := `SELECT COUNT(*) FROM sensors WHERE project_id = ? AND status = ?`
	if err := r.db.GetContext(ctx, &n, q, projectID, status); err != nil {
		return 0, fmt.Errorf("count sensors %s/%s: %w", projectID, status, err)
	}
	return n, nil
}

func (r *SensorRepository) Insert(ctx context.Context, s *domain.Sensor) error {
	q := `INSERT INTO sensors (sensor_id, project_id, sensor_type, model, install_lat, install_lng, install_depth, status, installed_at)
	      VALUES (:sensor_id, :project_id, :sensor_type, :model, :install_lat, :install_lng, :install_depth,
	              COALESCE(NULLIF(:status, ''), 'active'), :installed_at)`
	if _, err := r.db.NamedExecContext(ctx, q, s); err != nil {
		return fmt.Errorf("insert sensor %s: %w", s.SensorID, err)
	}
	return nil
}

func (r *SensorRepository) Update(ctx context.Context, s *domain.Sensor) error {
	q := `UPDATE sensors
	      SET sensor_type = COALESCE(NULLIF(:sensor_type, ''), sensor_type),
	          model = :model,
	          install_lat = :install_lat,
	          install_lng = :install_lng,
	          install_depth = :install_depth,
	          status = COALESCE(NULLIF(:status, ''), status),
	          installed_at = :installed_at
	      WHERE sensor_id = :sensor_id AND project_id = :project_id`
	res, err := r.db.NamedExecContext(ctx, q, s)
	if err != nil {
		return fmt.Errorf("update sensor %s: %w", s.SensorID, err)
	}
	return affectedOrNotFound(res)
}

func (r *SensorRepository) SoftDelete(ctx context.Context, projectID, sensorID string) error {
	q := `UPDATE sensors SET status = 'inactive' WHERE sensor_id = ? AND project_id = ?`
	res, err := r.db.ExecContext(ctx, q, sensorID, projectID)
	if err != nil {
		return fmt.Errorf("soft delete sensor %s/%s: %w", projectID, sensorID, err)
	}
	return affectedOrNotFound(res)
}
