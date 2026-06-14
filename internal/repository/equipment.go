package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// EquipmentRepository = SQL access for equipment. Mirrors EquipmentMapper
// (list + create only — baseline exposes no update/delete for equipment).
type EquipmentRepository struct {
	db *sqlx.DB
}

func NewEquipmentRepository(db *sqlx.DB) *EquipmentRepository {
	return &EquipmentRepository{db: db}
}

const equipmentCols = `equipment_id, project_id, type, model, serial, status`

func (r *EquipmentRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.Equipment, error) {
	q := `SELECT ` + equipmentCols + ` FROM equipment WHERE project_id = ? ORDER BY equipment_id`
	items := []domain.Equipment{}
	if err := r.db.SelectContext(ctx, &items, q, projectID); err != nil {
		return nil, fmt.Errorf("find equipment for project %s: %w", projectID, err)
	}
	return items, nil
}

func (r *EquipmentRepository) Insert(ctx context.Context, e *domain.Equipment) error {
	q := `INSERT INTO equipment (equipment_id, project_id, type, model, serial, status)
	      VALUES (:equipment_id, :project_id, :type, :model, :serial, COALESCE(NULLIF(:status, ''), 'active'))`
	if _, err := r.db.NamedExecContext(ctx, q, e); err != nil {
		return fmt.Errorf("insert equipment %s: %w", e.EquipmentID, err)
	}
	return nil
}
