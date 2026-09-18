package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Output wrappers for consistent json output in handlers

// M is an ad-hoc JSON object.
type M map[string]any

type APIError struct {
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Message
}

type ResultOutput struct {
	Result string `json:"result"`
}

// writeJSON encodes v as the response body with the given status code.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("writing json response", "error", err)
	}
}

func outputJSON(w http.ResponseWriter, o any) {
	writeJSON(w, http.StatusOK, o)
}

func resultJSON(w http.ResponseWriter, result string) {
	writeJSON(w, http.StatusOK, ResultOutput{Result: result})
}

func conflictError(w http.ResponseWriter, err error) {
	errJSON(w, http.StatusConflict, err)
}

func internalServerError(w http.ResponseWriter, err error) {
	errJSON(w, http.StatusInternalServerError, err)
}

func badPayloadError(w http.ResponseWriter, err error) {
	e := fmt.Errorf("bad json request: %w", err)
	errJSON(w, http.StatusBadRequest, e)
}

func badQueryError(w http.ResponseWriter, err error) {
	errJSON(w, http.StatusBadRequest, fmt.Errorf("bad query parameter: %w", err))
}

func notFoundError(w http.ResponseWriter, err error) {
	errJSON(w, http.StatusNotFound, err)
}

func badIntError(w http.ResponseWriter, r *http.Request) {
	s := r.PathValue("id")
	errJSON(w, http.StatusBadRequest, fmt.Errorf("%s is not a valid integer", s))
}

func errJSON(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, APIError{Message: err.Error()})
}
