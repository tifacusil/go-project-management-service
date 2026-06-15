//go:build integration

// Integration tests (CLAUDE.md §3.4): real MySQL via Testcontainers, schema
// applied with goose. Run with:  go test -tags=integration ./internal/repository/
// Requires Docker. Excluded from the default `go test ./...`.
package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/tifacusil/go-project-management-service/internal/domain"
	"github.com/tifacusil/go-project-management-service/internal/repository"
)

// setupMySQL spins an ephemeral MySQL, applies migrations, returns a live sqlx.DB.
func setupMySQL(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("project_management"),
		tcmysql.WithUsername("test"),
		tcmysql.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "multiStatements=true")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	goose.SetDialect("mysql")
	if err := goose.Up(db.DB, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestIntegration_ProjectCRUD(t *testing.T) {
	db := setupMySQL(t)
	ctx := context.Background()
	repo := repository.NewProjectRepository(db)

	// Insert → defaults applied (status active, created_at now).
	lat, lng := -12.04, -77.04
	p := &domain.Project{ProjectID: "shangtan", Name: "Shangtan", LocationLat: &lat, LocationLng: &lng}
	if err := repo.Insert(ctx, p); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := repo.FindByID(ctx, "shangtan")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Name != "Shangtan" || got.Status != domain.StatusActive {
		t.Errorf("unexpected after insert: %+v", got)
	}
	if got.CreatedAt.IsZero() || got.CreatedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("created_at not defaulted: %v", got.CreatedAt)
	}
	if got.LocationLat == nil || *got.LocationLat != lat {
		t.Errorf("lat round-trip wrong: %v", got.LocationLat)
	}

	// Count.
	if n, _ := repo.CountByID(ctx, "shangtan"); n != 1 {
		t.Errorf("count = %d, want 1", n)
	}

	// Update name, keep status.
	p.Name = "Shangtan Renombrado"
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = repo.FindByID(ctx, "shangtan")
	if got.Name != "Shangtan Renombrado" {
		t.Errorf("update not persisted: %q", got.Name)
	}

	// Soft delete → status inactive.
	if err := repo.SoftDelete(ctx, "shangtan"); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	got, _ = repo.FindByID(ctx, "shangtan")
	if got.Status != domain.StatusInactive {
		t.Errorf("not soft-deleted: %q", got.Status)
	}

	// Missing → ErrNotFound.
	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestIntegration_SensorAndCounts(t *testing.T) {
	db := setupMySQL(t)
	ctx := context.Background()
	projects := repository.NewProjectRepository(db)
	sensors := repository.NewSensorRepository(db)

	if err := projects.Insert(ctx, &domain.Project{ProjectID: "p1", Name: "P1"}); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	gnss := "gnss"
	s := &domain.Sensor{SensorID: "s1", ProjectID: "p1", SensorType: gnss}
	if err := sensors.Insert(ctx, s); err != nil {
		t.Fatalf("insert sensor: %v", err)
	}

	list, err := sensors.FindByProjectID(ctx, "p1")
	if err != nil || len(list) != 1 {
		t.Fatalf("find sensors: len=%d err=%v", len(list), err)
	}

	active, err := sensors.CountByProjectIDAndStatus(ctx, "p1", domain.StatusActive)
	if err != nil || active != 1 {
		t.Fatalf("active count = %d err=%v, want 1", active, err)
	}

	if err := sensors.SoftDelete(ctx, "p1", "s1"); err != nil {
		t.Fatalf("soft delete sensor: %v", err)
	}
	active, _ = sensors.CountByProjectIDAndStatus(ctx, "p1", domain.StatusActive)
	if active != 0 {
		t.Errorf("active after delete = %d, want 0", active)
	}
}
