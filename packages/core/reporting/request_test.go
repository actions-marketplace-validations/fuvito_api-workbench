package reporting_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

func makeResp(status int, statusText string, ms int64, body string, ct string) *domain.Response {
	return &domain.Response{
		StatusCode: status,
		Status:     statusText,
		Headers:    []domain.Header{{Name: "Content-Type", Value: ct}},
		Body:       []byte(body),
		Duration:   time.Duration(ms) * time.Millisecond,
		Size:       int64(len(body)),
	}
}

func TestTextRequest_Success(t *testing.T) {
	resp := makeResp(200, "200 OK", 42, `{"id":1}`, "application/json")
	out := reporting.TextRequest("GET", "https://api.example.com/users/1", resp, nil)

	if !strings.Contains(out, "GET https://api.example.com/users/1") {
		t.Error("missing method + URL line")
	}
	if !strings.Contains(out, "200 OK") {
		t.Error("missing status")
	}
	if !strings.Contains(out, "42ms") {
		t.Error("missing duration")
	}
	if !strings.Contains(out, `"id": 1`) {
		t.Error("body not pretty-printed")
	}
}

func TestTextRequest_Error(t *testing.T) {
	out := reporting.TextRequest("GET", "https://api.example.com/down", nil, errors.New("connection refused"))

	if !strings.Contains(out, "GET https://api.example.com/down") {
		t.Error("missing method + URL line")
	}
	if !strings.Contains(out, "ERROR:") {
		t.Error("missing ERROR label")
	}
	if !strings.Contains(out, "connection refused") {
		t.Error("missing error message")
	}
}

func TestTextRequest_NonJSONBody(t *testing.T) {
	resp := makeResp(200, "200 OK", 10, "plain text response", "text/plain")
	out := reporting.TextRequest("GET", "https://api.example.com/health", resp, nil)

	if !strings.Contains(out, "plain text response") {
		t.Error("missing plain text body")
	}
}

func TestJSONRequest_Success(t *testing.T) {
	resp := makeResp(201, "201 Created", 55, `{"created":true}`, "application/json")
	raw, err := reporting.JSONRequest("POST", "https://api.example.com/items", resp, nil)
	if err != nil {
		t.Fatalf("JSONRequest error: %v", err)
	}

	var report reporting.JSONRequestReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if report.Method != "POST" {
		t.Errorf("method: got %q", report.Method)
	}
	if report.StatusCode != 201 {
		t.Errorf("statusCode: got %d", report.StatusCode)
	}
	if report.DurationMs != 55 {
		t.Errorf("durationMs: got %d", report.DurationMs)
	}
	if report.Error != "" {
		t.Errorf("unexpected error field: %q", report.Error)
	}
}

func TestJSONRequest_Error(t *testing.T) {
	raw, err := reporting.JSONRequest("GET", "https://api.example.com/down", nil, errors.New("timeout"))
	if err != nil {
		t.Fatalf("JSONRequest error: %v", err)
	}

	var report reporting.JSONRequestReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if report.Error != "timeout" {
		t.Errorf("error field: got %q", report.Error)
	}
	if report.StatusCode != 0 {
		t.Errorf("statusCode should be 0 on error, got %d", report.StatusCode)
	}
}
