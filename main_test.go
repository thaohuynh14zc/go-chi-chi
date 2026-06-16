package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanicRecovery(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)

	r := SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	// Assert response status code is 500 Internal Server Error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Assert response headers contain X-Request-Id
	reqID := rec.Header().Get("X-Request-Id")
	if reqID == "" {
		t.Error("expected X-Request-Id header to be present")
	}

	// Assert captured log output contains the panic message and the correlation_id
	logOutput := buf.String()
	if !strings.Contains(logOutput, "simulated database connection failure") {
		t.Errorf("expected log output to contain panic message, got: %s", logOutput)
	}

	// Parse log output to verify correlation_id
	var logMap map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logMap)
	if err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	corrID, ok := logMap["correlation_id"].(string)
	if !ok || corrID != reqID {
		t.Errorf("expected correlation_id to be %q, got %q", reqID, corrID)
	}
}
