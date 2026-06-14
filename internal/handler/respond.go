package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tifacusil/go-project-management-service/internal/observability"
	"github.com/tifacusil/go-project-management-service/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return false
	}
	return true
}

// failErr maps a service error Kind to its HTTP status; unknown errors → 500 (logged).
func failErr(w http.ResponseWriter, r *http.Request, err error) {
	var se *service.Error
	if errors.As(err, &se) {
		status := http.StatusBadRequest
		if se.Kind == service.KindNotFound {
			status = http.StatusNotFound
		}
		writeError(w, status, se.Msg)
		return
	}
	observability.FromContext(r.Context()).Error("request failed", "event", "handler_error", "err", err)
	writeError(w, http.StatusInternalServerError, "error interno")
}
