package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/tifacusil/go-project-management-service/docs"
	"github.com/tifacusil/go-project-management-service/internal/domain"
)

// Fakes implement the handler API seams without a DB.

type fakeProjectSvc struct{}

func (fakeProjectSvc) ListAll(context.Context) ([]domain.Project, error) {
	return []domain.Project{{ProjectID: "p1", Name: "P1", Status: "active"}}, nil
}
func (fakeProjectSvc) GetByIDWithSensors(_ context.Context, id string) (*domain.Project, error) {
	return &domain.Project{ProjectID: id, Name: "P1", Status: "active"}, nil
}
func (fakeProjectSvc) Create(_ context.Context, p *domain.Project) (*domain.Project, error) {
	return p, nil
}
func (fakeProjectSvc) Update(_ context.Context, id string, p *domain.Project) (*domain.Project, error) {
	p.ProjectID = id
	return p, nil
}
func (fakeProjectSvc) SoftDelete(context.Context, string) error { return nil }

type fakeSensorSvc struct{}

func (fakeSensorSvc) ListByProject(context.Context, string) ([]domain.Sensor, error) {
	return []domain.Sensor{}, nil
}
func (fakeSensorSvc) GetByID(_ context.Context, pid, sid string) (*domain.Sensor, error) {
	return &domain.Sensor{SensorID: sid, ProjectID: pid, Status: "active"}, nil
}
func (fakeSensorSvc) Create(_ context.Context, pid string, s *domain.Sensor) (*domain.Sensor, error) {
	s.ProjectID = pid
	return s, nil
}
func (fakeSensorSvc) Update(_ context.Context, pid, sid string, s *domain.Sensor) (*domain.Sensor, error) {
	s.ProjectID, s.SensorID = pid, sid
	return s, nil
}
func (fakeSensorSvc) SoftDelete(context.Context, string, string) error { return nil }

type fakeEquipmentSvc struct{}

func (fakeEquipmentSvc) ListByProject(context.Context, string) ([]domain.Equipment, error) {
	return []domain.Equipment{}, nil
}
func (fakeEquipmentSvc) Create(_ context.Context, pid string, e *domain.Equipment) (*domain.Equipment, error) {
	e.ProjectID = pid
	return e, nil
}

// buildRouter mirrors main.go wiring — this is what catches a chi mount overlap
// panic. El servicio sirve en raíz (el gateway nginx reescribe el prefix
// /api/v1/projects); se montan /docs y /health junto a la raíz comodín para
// que el test detecte solapamientos reales de mount.
func buildRouter() chi.Router {
	pj := NewProjectHandler(fakeProjectSvc{}, validator.New(validator.WithRequiredStructEnabled()))
	sn := NewSensorHandler(fakeSensorSvc{})
	eq := NewEquipmentHandler(fakeEquipmentSvc{})

	projects := pj.Routes(func(pr chi.Router) {
		pr.Mount("/sensors", sn.Routes())
		pr.Mount("/equipment", eq.Routes())
	})

	r := chi.NewRouter()
	r.Mount("/docs", docs.Routes())
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Mount("/", projects)
	return r
}

func TestRouting(t *testing.T) {
	srv := httptest.NewServer(buildRouter()) // panics here if mounts overlap
	defer srv.Close()

	cases := []struct {
		method, path string
		want         int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/p1", http.StatusOK},
		{"GET", "/p1/sensors", http.StatusOK},
		{"GET", "/p1/sensors/s1", http.StatusOK},
		{"GET", "/p1/equipment", http.StatusOK},
		// Rutas estáticas ganan sobre el comodín /{project_id}.
		{"GET", "/health", http.StatusOK},
		{"GET", "/docs/openapi.yaml", http.StatusOK},
	}
	for _, c := range cases {
		req, _ := http.NewRequest(c.method, srv.URL+c.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", c.method, c.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != c.want {
			t.Errorf("%s %s: got %d want %d", c.method, c.path, resp.StatusCode, c.want)
		}
	}
}
