package service

import (
	"context"
	"testing"

	"github.com/tifacusil/go-project-management-service/internal/domain"
)

type fakeEquipStore struct {
	inserted []domain.Equipment
}

func (f *fakeEquipStore) FindByProjectID(_ context.Context, pid string) ([]domain.Equipment, error) {
	out := []domain.Equipment{}
	for _, e := range f.inserted {
		if e.ProjectID == pid {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeEquipStore) Insert(_ context.Context, e *domain.Equipment) error {
	f.inserted = append(f.inserted, *e)
	return nil
}

func equipSvc(projExists bool) (*EquipmentService, *fakeEquipStore) {
	st := &fakeEquipStore{}
	pe := fakeProjExists{ids: map[string]bool{}}
	if projExists {
		pe.ids["p1"] = true
	}
	return NewEquipmentService(pe, st), st
}

func strptr(s string) *string { return &s }

func TestEquipmentCreate_ProjectMustExist(t *testing.T) {
	svc, _ := equipSvc(false)
	_, err := svc.Create(context.Background(), "p1", &domain.Equipment{EquipmentID: "e1", Type: strptr("DTU")})
	mustErrKind(t, err, KindNotFound)
}

func TestEquipmentCreate_RejectsBlankID(t *testing.T) {
	svc, _ := equipSvc(true)
	_, err := svc.Create(context.Background(), "p1", &domain.Equipment{Type: strptr("DTU")})
	mustErrKind(t, err, KindInvalid)
}

func TestEquipmentCreate_RejectsBlankType(t *testing.T) {
	svc, _ := equipSvc(true)
	_, err := svc.Create(context.Background(), "p1", &domain.Equipment{EquipmentID: "e1"})
	mustErrKind(t, err, KindInvalid)
}

func TestEquipmentCreate_RejectsBadType(t *testing.T) {
	svc, _ := equipSvc(true)
	_, err := svc.Create(context.Background(), "p1", &domain.Equipment{EquipmentID: "e1", Type: strptr("rocket")})
	mustErrKind(t, err, KindInvalid)
}

func TestEquipmentCreate_DefaultsStatusActive(t *testing.T) {
	svc, st := equipSvc(true)
	got, err := svc.Create(context.Background(), "p1", &domain.Equipment{EquipmentID: "e1", Type: strptr("  battery  ")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.Status != domain.StatusActive {
		t.Errorf("status default wrong: %q", got.Status)
	}
	if got.Type == nil || *got.Type != "battery" {
		t.Errorf("type not trimmed: %v", got.Type)
	}
	if len(st.inserted) != 1 {
		t.Errorf("expected 1 insert, got %d", len(st.inserted))
	}
}
