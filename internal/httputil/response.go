package httputil

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// RespondJSON sends a JSON response with the given status code.
// Logs encoding errors to avoid silent failures.
func RespondJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR: failed to encode JSON response: %v", err)
	}
}

// RespondError sends a JSON error response with the given message and status code.
func RespondError(w http.ResponseWriter, message string, statusCode int) {
	RespondJSON(w, ErrorResponse{Error: message}, statusCode)
}

// RespondErrorWithCode sends a JSON error response with a machine-readable error code.
func RespondErrorWithCode(w http.ResponseWriter, message string, code string, statusCode int) {
	RespondJSON(w, ErrorResponse{Error: message, Code: code}, statusCode)
}
