package service

import (
	"context"
	"testing"

	"github.com/tifacusil/go-project-management-service/internal/domain"
	"github.com/tifacusil/go-project-management-service/internal/repository"
)

// fakeProjExists implements projectExistence.
type fakeProjExists struct{ ids map[string]bool }

func (f fakeProjExists) CountByID(_ context.Context, id string) (int, error) {
	if f.ids[id] {
		return 1, nil
	}
	return 0, nil
}

// fakeSensorStore implements sensorStore.
type fakeSensorStore struct {
	store      map[string]*domain.Sensor // key sensor_id
	lastUpdate *domain.Sensor
}

func newFakeSensorStore() *fakeSensorStore {
	return &fakeSensorStore{store: map[string]*domain.Sensor{}}
}

func (f *fakeSensorStore) FindByProjectID(_ context.Context, pid string) ([]domain.Sensor, error) {
	out := []domain.Sensor{}
	for _, s := range f.store {
		if s.ProjectID == pid {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (f *fakeSensorStore) FindByIDAndProjectID(_ context.Context, pid, sid string) (*domain.Sensor, error) {
	s, ok := f.store[sid]
	if !ok || s.ProjectID != pid {
		return nil, repository.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (f *fakeSensorStore) CountByID(_ context.Context, sid string) (int, error) {
	if _, ok := f.store[sid]; ok {
		return 1, nil
	}
	return 0, nil
}
func (f *fakeSensorStore) Insert(_ context.Context, s *domain.Sensor) error {
	cp := *s
	f.store[s.SensorID] = &cp
	return nil
}
func (f *fakeSensorStore) Update(_ context.Context, s *domain.Sensor) error {
	cp := *s
	f.lastUpdate = &cp
	f.store[s.SensorID] = &cp
	return nil
}
func (f *fakeSensorStore) SoftDelete(_ context.Context, pid, sid string) error {
	f.store[sid].Status = domain.StatusInactive
	return nil
}

func sensorSvc(projExists bool) (*SensorService, *fakeSensorStore) {
	st := newFakeSensorStore()
	pe := fakeProjExists{ids: map[string]bool{}}
	if projExists {
		pe.ids["p1"] = true
	}
	return NewSensorService(pe, st), st
}

func TestSensorCreate_ProjectMustExist(t *testing.T) {
	svc, _ := sensorSvc(false)
	_, err := svc.Create(context.Background(), "p1", &domain.Sensor{SensorID: "s1", SensorType: "gnss"})
	mustErrKind(t, err, KindNotFound)
}

func TestSensorCreate_RejectsBadType(t *testing.T) {
	svc, _ := sensorSvc(true)
	_, err := svc.Create(context.Background(), "p1", &domain.Sensor{SensorID: "s1", SensorType: "lidar"})
	mustErrKind(t, err, KindInvalid)
}

func TestSensorCreate_RejectsBlankType(t *testing.T) {
	svc, _ := sensorSvc(true)
	_, err := svc.Create(context.Background(), "p1", &domain.Sensor{SensorID: "s1"})
	mustErrKind(t, err, KindInvalid)
}

func TestSensorCreate_RejectsDuplicate(t *testing.T) {
	svc, st := sensorSvc(true)
	st.store["s1"] = &domain.Sensor{SensorID: "s1", ProjectID: "p1", SensorType: "gnss"}
	_, err := svc.Create(context.Background(), "p1", &domain.Sensor{SensorID: "s1", SensorType: "rainfall"})
	mustErrKind(t, err, KindConflict)
}

func TestSensorCreate_RejectsBodyURLMismatch(t *testing.T) {
	svc, _ := sensorSvc(true)
	_, err := svc.Create(context.Background(), "p1", &domain.Sensor{SensorID: "s1", ProjectID: "other", SensorType: "gnss"})
	mustErrKind(t, err, KindInvalid)
}

func TestSensorCreate_OKNormalizesType(t *testing.T) {
	svc, st := sensorSvc(true)
	_, err := svc.Create(context.Background(), "p1", &domain.Sensor{SensorID: "s1", SensorType: "  strain  "})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if st.store["s1"].SensorType != "strain" {
		t.Errorf("type not trimmed: %q", st.store["s1"].SensorType)
	}
}

func TestSensorUpdate_MergeKeepsPreviousModel(t *testing.T) {
	svc, st := sensorSvc(true)
	model := "X100"
	st.store["s1"] = &domain.Sensor{SensorID: "s1", ProjectID: "p1", SensorType: "gnss", Model: &model, Status: "active"}

	// Body changes nothing relevant → model + type must survive.
	_, err := svc.Update(context.Background(), "p1", "s1", &domain.Sensor{})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if st.lastUpdate.Model == nil || *st.lastUpdate.Model != "X100" {
		t.Errorf("model not preserved: %v", st.lastUpdate.Model)
	}
	if st.lastUpdate.SensorType != "gnss" {
		t.Errorf("type not preserved: %q", st.lastUpdate.SensorType)
	}
}

func TestSensorUpdate_RejectsBadNewType(t *testing.T) {
	svc, st := sensorSvc(true)
	st.store["s1"] = &domain.Sensor{SensorID: "s1", ProjectID: "p1", SensorType: "gnss"}
	_, err := svc.Update(context.Background(), "p1", "s1", &domain.Sensor{SensorType: "lidar"})
	mustErrKind(t, err, KindInvalid)
}

func TestSensorGet_NotFound(t *testing.T) {
	svc, _ := sensorSvc(true)
	_, err := svc.GetByID(context.Background(), "p1", "ghost")
	mustErrKind(t, err, KindNotFound)
}
