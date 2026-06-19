package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

const maxRequestBodyBytes = 1 << 20

type unsupportedMediaTypeError struct {
	contentType string
}

func (e *unsupportedMediaTypeError) Error() string {
	return fmt.Sprintf("Content-Type %q is not supported; use application/json", e.contentType)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	contentType := r.Header.Get("Content-Type")
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || mediaType != "application/json" {
			return &unsupportedMediaTypeError{contentType: contentType}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("body must contain a single JSON value")
	}
	return nil
}

func writeJSONReadError(w http.ResponseWriter, err error) {
	var unsupportedMediaType *unsupportedMediaTypeError
	if errors.As(err, &unsupportedMediaType) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorBody(unsupportedMediaType.Error()))
		return
	}

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorBody("request body is too large"))
		return
	}

	writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON: "+err.Error()))
}

func errorBody(msg string) map[string]string {
	return map[string]string{"error": msg}
}
