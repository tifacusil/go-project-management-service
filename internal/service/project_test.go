package service

import (
	"context"
	"errors"
	"testing"

	"github.com/tifacusil/go-project-management-service/internal/domain"
	"github.com/tifacusil/go-project-management-service/internal/repository"
)

// fakeProjectRepo is an in-memory stand-in for the project repository.
type fakeProjectRepo struct {
	store      map[string]*domain.Project
	insertErr  error
	updateErr  error
	lastInsert *domain.Project
	lastUpdate *domain.Project
}

func newFakeProjectRepo() *fakeProjectRepo {
	return &fakeProjectRepo{store: map[string]*domain.Project{}}
}

func (f *fakeProjectRepo) FindAll(context.Context) ([]domain.Project, error) {
	out := []domain.Project{}
	for _, p := range f.store {
		out = append(out, *p)
	}
	return out, nil
}
func (f *fakeProjectRepo) FindByID(_ context.Context, id string) (*domain.Project, error) {
	p, ok := f.store[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (f *fakeProjectRepo) CountByID(_ context.Context, id string) (int, error) {
	if _, ok := f.store[id]; ok {
		return 1, nil
	}
	return 0, nil
}
func (f *fakeProjectRepo) Insert(_ context.Context, p *domain.Project) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	cp := *p
	f.lastInsert = &cp
	f.store[p.ProjectID] = &cp
	return nil
}
func (f *fakeProjectRepo) Update(_ context.Context, p *domain.Project) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	cp := *p
	f.lastUpdate = &cp
	f.store[p.ProjectID] = &cp
	return nil
}
func (f *fakeProjectRepo) SoftDelete(_ context.Context, id string) error {
	p := f.store[id]
	p.Status = domain.StatusInactive
	return nil
}

// fakeSensorRepo covers the two reads ProjectService needs.
type fakeSensorRepo struct {
	byProject   map[string][]domain.Sensor
	activeCount map[string]int
}

func newFakeSensorRepo() *fakeSensorRepo {
	return &fakeSensorRepo{byProject: map[string][]domain.Sensor{}, activeCount: map[string]int{}}
}

func (f *fakeSensorRepo) FindByProjectID(_ context.Context, pid string) ([]domain.Sensor, error) {
	return f.byProject[pid], nil
}
func (f *fakeSensorRepo) CountByProjectIDAndStatus(_ context.Context, pid, _ string) (int, error) {
	return f.activeCount[pid], nil
}

func mustErrKind(t *testing.T, err error, want Kind) {
	t.Helper()
	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("want *service.Error kind=%d, got %T: %v", want, err, err)
	}
	if se.Kind != want {
		t.Fatalf("want kind=%d, got kind=%d (%s)", want, se.Kind, se.Msg)
	}
}

func TestProjectCreate_RejectsBlankID(t *testing.T) {
	svc := NewProjectService(newFakeProjectRepo(), newFakeSensorRepo())
	_, err := svc.Create(context.Background(), &domain.Project{Name: "x"})
	mustErrKind(t, err, KindInvalid)
}

func TestProjectCreate_RejectsBlankName(t *testing.T) {
	svc := NewProjectService(newFakeProjectRepo(), newFakeSensorRepo())
	_, err := svc.Create(context.Background(), &domain.Project{ProjectID: "p1"})
	mustErrKind(t, err, KindInvalid)
}

func TestProjectCreate_RejectsDuplicate(t *testing.T) {
	pr := newFakeProjectRepo()
	pr.store["p1"] = &domain.Project{ProjectID: "p1", Name: "exists"}
	svc := NewProjectService(pr, newFakeSensorRepo())

	_, err := svc.Create(context.Background(), &domain.Project{ProjectID: "p1", Name: "dup"})
	mustErrKind(t, err, KindConflict)
}

func TestProjectCreate_OK(t *testing.T) {
	pr := newFakeProjectRepo()
	svc := NewProjectService(pr, newFakeSensorRepo())

	got, err := svc.Create(context.Background(), &domain.Project{ProjectID: "p1", Name: "new"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.ProjectID != "p1" || got.Name != "new" {
		t.Fatalf("unexpected project: %+v", got)
	}
}

func TestProjectGet_NotFound(t *testing.T) {
	svc := NewProjectService(newFakeProjectRepo(), newFakeSensorRepo())
	_, err := svc.GetByIDWithSensors(context.Background(), "nope")
	mustErrKind(t, err, KindNotFound)
}

func TestProjectUpdate_MergeKeepsPreviousOnNil(t *testing.T) {
	pr := newFakeProjectRepo()
	lat := 1.5
	pr.store["p1"] = &domain.Project{ProjectID: "p1", Name: "old", LocationLat: &lat, Status: "active"}
	svc := NewProjectService(pr, newFakeSensorRepo())

	// Body sets only Name; lat/status must survive.
	_, err := svc.Update(context.Background(), "p1", &domain.Project{Name: "renamed"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got := pr.lastUpdate
	if got.Name != "renamed" {
		t.Errorf("name not applied: %q", got.Name)
	}
	if got.LocationLat == nil || *got.LocationLat != 1.5 {
		t.Errorf("lat not preserved: %v", got.LocationLat)
	}
	if got.Status != "active" {
		t.Errorf("status not preserved: %q", got.Status)
	}
}

func TestProjectUpdate_RejectsIDMismatch(t *testing.T) {
	pr := newFakeProjectRepo()
	pr.store["p1"] = &domain.Project{ProjectID: "p1", Name: "x"}
	svc := NewProjectService(pr, newFakeSensorRepo())

	_, err := svc.Update(context.Background(), "p1", &domain.Project{ProjectID: "other", Name: "y"})
	mustErrKind(t, err, KindInvalid)
}

func TestProjectDelete_BlockedByActiveSensors(t *testing.T) {
	pr := newFakeProjectRepo()
	pr.store["p1"] = &domain.Project{ProjectID: "p1", Name: "x"}
	sr := newFakeSensorRepo()
	sr.activeCount["p1"] = 2
	svc := NewProjectService(pr, sr)

	err := svc.SoftDelete(context.Background(), "p1")
	mustErrKind(t, err, KindInvalid)
}

func TestProjectDelete_OKWhenNoActiveSensors(t *testing.T) {
	pr := newFakeProjectRepo()
	pr.store["p1"] = &domain.Project{ProjectID: "p1", Name: "x", Status: "active"}
	svc := NewProjectService(pr, newFakeSensorRepo())

	if err := svc.SoftDelete(context.Background(), "p1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if pr.store["p1"].Status != domain.StatusInactive {
		t.Errorf("not soft-deleted: %q", pr.store["p1"].Status)
	}
}
