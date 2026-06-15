package seed

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

type fakeProjects struct {
	count    int
	byID     map[string]bool
	inserted []domain.Project
}

func (f *fakeProjects) CountAll(context.Context) (int, error) { return f.count, nil }
func (f *fakeProjects) CountByID(_ context.Context, id string) (int, error) {
	if f.byID[id] {
		return 1, nil
	}
	return 0, nil
}
func (f *fakeProjects) Insert(_ context.Context, p *domain.Project) error {
	f.inserted = append(f.inserted, *p)
	if f.byID == nil {
		f.byID = map[string]bool{}
	}
	f.byID[p.ProjectID] = true
	return nil
}

type fakeSensors struct {
	count    int
	inserted []domain.Sensor
}

func (f *fakeSensors) CountAll(context.Context) (int, error) { return f.count, nil }
func (f *fakeSensors) Insert(_ context.Context, s *domain.Sensor) error {
	f.inserted = append(f.inserted, *s)
	return nil
}

func nopLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestSeed_EmptyDB_InsertsAll(t *testing.T) {
	p := &fakeProjects{}
	s := &fakeSensors{}
	if err := Run(context.Background(), p, s, nopLog()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if len(p.inserted) != 2 {
		t.Errorf("projects insertados = %d, want 2", len(p.inserted))
	}
	if len(s.inserted) != 17 {
		t.Errorf("sensors insertados = %d, want 17", len(s.inserted))
	}
}

func TestSeed_NonEmpty_NoOp(t *testing.T) {
	p := &fakeProjects{count: 5}
	s := &fakeSensors{count: 9}
	if err := Run(context.Background(), p, s, nopLog()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if len(p.inserted) != 0 || len(s.inserted) != 0 {
		t.Errorf("no debería insertar nada: projects=%d sensors=%d", len(p.inserted), len(s.inserted))
	}
}

func TestSeed_ProjectsExistButSensorsEmpty_SeedsSensorsOnly(t *testing.T) {
	// projects ya poblados; sensors vacíos → siembra sensores, no duplica proyectos.
	p := &fakeProjects{count: 2, byID: map[string]bool{"shangtan": true, "lingzhan": true}}
	s := &fakeSensors{}
	if err := Run(context.Background(), p, s, nopLog()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if len(p.inserted) != 0 {
		t.Errorf("no debía insertar proyectos (ya existen), insertó %d", len(p.inserted))
	}
	if len(s.inserted) != 17 {
		t.Errorf("sensors = %d, want 17", len(s.inserted))
	}
}
