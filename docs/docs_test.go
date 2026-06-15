package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRedirectToTrailingSlash(t *testing.T) {
	// Espeja main.go: subrouter montado en /docs. /docs (sin barra) debe
	// redirigir a "docs/" con Location RELATIVO, para que la base de la URL
	// relativa del spec sea /docs/ y se preserve el prefix del gateway.
	root := chi.NewRouter()
	root.Mount("/docs", Routes())
	srv := httptest.NewServer(root)
	defer srv.Close()

	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	t.Run("/docs redirige a docs/ relativo", func(t *testing.T) {
		resp, err := noFollow.Get(srv.URL + "/docs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Fatalf("status = %d, want 301", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "docs/" {
			t.Errorf("Location = %q, want %q (relativo, sin barra inicial)", loc, "docs/")
		}
	})

	t.Run("/docs/ sirve la UI", func(t *testing.T) {
		resp, err := noFollow.Get(srv.URL + "/docs/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
	})
}

func TestRoutes(t *testing.T) {
	srv := httptest.NewServer(Routes())
	defer srv.Close()

	t.Run("UI sirve HTML con Swagger", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("content-type = %q, want text/html", ct)
		}
	})

	t.Run("spec sirve el YAML embebido", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/openapi.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if len(openapiSpec) == 0 {
			t.Fatal("spec embebido vacío")
		}
		if !strings.Contains(string(openapiSpec), "openapi: 3.0.3") {
			t.Error("spec no contiene cabecera openapi 3.0.3")
		}
	})
}
