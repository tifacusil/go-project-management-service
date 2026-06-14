package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// ErrNotFound = no row matched. Service maps to 404.
var ErrNotFound = errors.New("not found")

// ProjectRepository = SQL access for projects. Mirrors ProjectMapper.
type ProjectRepository struct {
	db *sqlx.DB
}

func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

const projectCols = `project_id, name, location_lat, location_lng, status, created_at`

func (r *ProjectRepository) FindAll(ctx context.Context) ([]domain.Project, error) {
	q := `SELECT ` + projectCols + ` FROM projects ORDER BY created_at DESC`
	projects := []domain.Project{}
	if err := r.db.SelectContext(ctx, &projects, q); err != nil {
		return nil, fmt.Errorf("find all projects: %w", err)
	}
	return projects, nil
}

func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*domain.Project, error) {
	q := `SELECT ` + projectCols + ` FROM projects WHERE project_id = ?`
	var p domain.Project
	if err := r.db.GetContext(ctx, &p, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find project %s: %w", id, err)
	}
	return &p, nil
}

func (r *ProjectRepository) CountByID(ctx context.Context, id string) (int, error) {
	var n int
	q := `SELECT COUNT(*) FROM projects WHERE project_id = ?`
	if err := r.db.GetContext(ctx, &n, q, id); err != nil {
		return 0, fmt.Errorf("count project %s: %w", id, err)
	}
	return n, nil
}

func (r *ProjectRepository) Insert(ctx context.Context, p *domain.Project) error {
	q := `INSERT INTO projects (project_id, name, location_lat, location_lng, status)
	      VALUES (:project_id, :name, :location_lat, :location_lng, COALESCE(NULLIF(:status, ''), 'active'))`
	if _, err := r.db.NamedExecContext(ctx, q, p); err != nil {
		return fmt.Errorf("insert project %s: %w", p.ProjectID, err)
	}
	return nil
}

func (r *ProjectRepository) Update(ctx context.Context, p *domain.Project) error {
	q := `UPDATE projects
	      SET name = :name,
	          location_lat = :location_lat,
	          location_lng = :location_lng,
	          status = COALESCE(NULLIF(:status, ''), status)
	      WHERE project_id = :project_id`
	res, err := r.db.NamedExecContext(ctx, q, p)
	if err != nil {
		return fmt.Errorf("update project %s: %w", p.ProjectID, err)
	}
	return affectedOrNotFound(res)
}

func (r *ProjectRepository) SoftDelete(ctx context.Context, id string) error {
	q := `UPDATE projects SET status = 'inactive' WHERE project_id = ?`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("soft delete project %s: %w", id, err)
	}
	return affectedOrNotFound(res)
}

func affectedOrNotFound(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
