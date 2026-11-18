package reporting_test

import (
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

func TestConformReportsRedactSecrets(t *testing.T) {
	const secret = "hunter2-secret"
	res := &conformance.ConformResult{
		SpecTitle: "api", SpecVersion: "1", BaseURL: "https://api.example.com/?key=" + secret,
		Operations: []conformance.OperationResult{
			{Method: "GET", Path: "/a", Error: `Get "https://api.example.com/a?key=` + secret + `": refused`},
			{Method: "GET", Path: "/b", StatusCode: 200,
				Violations:        []openapi.ValidationError{{Path: "body", Message: "got " + secret}},
				RequestViolations: []openapi.ValidationError{{Message: "sent " + secret}}},
		},
		Duration: time.Millisecond,
		Secrets:  []string{secret},
	}
	neg := &conformance.NegativeResult{
		SpecTitle: "api", SpecVersion: "1", Secrets: []string{secret},
		Probes: []conformance.NegativeProbeResult{{Description: "probe", Error: "failed for " + secret}},
	}

	outputs := map[string]string{
		"text":     reporting.ConformText(res, nil) + reporting.NegativeText(neg),
		"markdown": string(reporting.ConformMarkdown(res, nil, neg)),
	}
	if b, err := reporting.ConformJSON(res, nil); err == nil {
		outputs["json"] = string(b)
	} else {
		t.Fatal(err)
	}
	if b, err := reporting.ConformJUnit(res); err == nil {
		outputs["junit"] = string(b)
	} else {
		t.Fatal(err)
	}
	if b, err := reporting.ConformHTML(res, nil, neg); err == nil {
		outputs["html"] = string(b)
	} else {
		t.Fatal(err)
	}

	for name, out := range outputs {
		if strings.Contains(out, secret) {
			t.Errorf("%s conform report leaked the secret:\n%s", name, out)
		}
		if !strings.Contains(out, "***") && !strings.Contains(out, `\*\*\*`) { // Markdown escapes the marker
			t.Errorf("%s conform report should show where it was removed", name)
		}
	}
}

func TestConformReportsWithoutSecretsAreUnchanged(t *testing.T) {
	res := makeConformResult(false)
	if out := reporting.ConformText(res, nil); !strings.Contains(out, "missing required field") || strings.Contains(out, "***") {
		t.Errorf("nothing to redact, nothing changes:\n%s", out)
	}
}
