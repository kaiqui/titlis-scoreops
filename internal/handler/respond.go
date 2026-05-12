package handler

import (
	"encoding/json"
	"net/http"

	"github.com/titlis/scoreops/internal/domain"
)

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

func handleDomainError(w http.ResponseWriter, err error) {
	if domain.IsValidation(err) {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	switch err {
	case domain.ErrNotFound:
		respondError(w, http.StatusNotFound, "not found")
	case domain.ErrConflict:
		respondError(w, http.StatusConflict, "já existe um registro com esses dados")
	default:
		respondError(w, http.StatusInternalServerError, "internal error")
	}
}
