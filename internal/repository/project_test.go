package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// Unit tests (CLAUDE.md §3.1): repository logic isolated from a real MySQL via sqlmock.
//
// TODO(SE-practice): missing test layers for this service (CLAUDE.md §3):
//   - contract tests: validate domain structs against contracts/openapi/*.yaml
//   - CDCT (pact-go): producer/consumer compatibility
//   - integration: Testcontainers MySQL for the critical CRUD flow

func newMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = mockDB.Close() })
	return sqlx.NewDb(mockDB, "mysql"), mock
}

func TestFindByID_NotFound(t *testing.T) {
	db, mock := newMock(t)
	repo := NewProjectRepository(db)

	// No rows → sql.ErrNoRows → repo maps to ErrNotFound.
	mock.ExpectQuery("FROM projects WHERE project_id").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindByID(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSoftDelete_NoRowsIsNotFound(t *testing.T) {
	db, mock := newMock(t)
	repo := NewProjectRepository(db)

	// 0 rows affected → ErrNotFound (project did not exist).
	mock.ExpectExec("UPDATE projects SET status = 'inactive'").
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.SoftDelete(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFindAll_OrdersByCreatedAtDesc(t *testing.T) {
	db, mock := newMock(t)
	repo := NewProjectRepository(db)

	rows := sqlmock.NewRows([]string{"project_id", "name", "location_lat", "location_lng", "status", "created_at"})
	mock.ExpectQuery("ORDER BY created_at DESC").WillReturnRows(rows)

	if _, err := repo.FindAll(context.Background()); err != nil {
		t.Fatalf("find all: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
