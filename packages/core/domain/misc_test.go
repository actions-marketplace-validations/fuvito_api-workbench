package domain_test

import (
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

func TestTimingTotalMs(t *testing.T) {
	if got := (domain.Timing{DNSMs: 1, ConnectMs: 2, TLSMs: 3, TTFBMs: 4, TransferMs: 5}).TotalMs(); got != 15 {
		t.Errorf("got %d", got)
	}
	if (domain.Timing{}).TotalMs() != 0 {
		t.Error("zero timing")
	}
}

func TestResponseHelpers(t *testing.T) {
	r := &domain.Response{StatusCode: 204, Body: []byte("héllo"), Duration: time.Second}
	if r.BodyString() != "héllo" {
		t.Errorf("BodyString: %q", r.BodyString())
	}
	for code, want := range map[int]bool{199: false, 200: true, 299: true, 300: false, 404: false} {
		if (&domain.Response{StatusCode: code}).IsSuccess() != want {
			t.Errorf("IsSuccess(%d) should be %v", code, want)
		}
	}
	if (&domain.Response{}).BodyString() != "" {
		t.Error("nil body should be empty string")
	}
}

func TestDefaultSettings(t *testing.T) {
	s := domain.DefaultSettings()
	if s.Theme != "dark" || s.FontSize != 13 || s.DefaultTimeoutMs != 30000 {
		t.Errorf("defaults: %+v", s)
	}
	if s.ProxyURL != "" || s.TLSSkipVerify || s.TelemetryEnabled {
		t.Errorf("privacy-sensitive options must default off: %+v", s)
	}
}

func TestNewWorkspace(t *testing.T) {
	w := domain.NewWorkspace("proj")
	if w.Name != "proj" || w.SchemaVersion != domain.CurrentSchemaVersion {
		t.Errorf("%+v", w)
	}
}
