package serialization_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

func TestFollowRedirectsRoundTrip(t *testing.T) {
	req, err := serialization.ParseRequest([]byte("schemaVersion: \"1\"\nmethod: GET\nurl: http://x/old\nfollowRedirects: false\n"), "yaml")
	if err != nil || !req.NoFollowRedirects {
		t.Fatalf("parse: %v %+v", err, req)
	}
	out, err := serialization.WriteRequest(req, "yaml")
	if err != nil || !strings.Contains(string(out), "followRedirects: false") {
		t.Errorf("write: %v\n%s", err, out)
	}

	def, _ := serialization.ParseRequest([]byte("schemaVersion: \"1\"\nmethod: GET\nurl: http://x\n"), "yaml")
	if def.NoFollowRedirects {
		t.Error("the default is to follow redirects")
	}
	out, _ = serialization.WriteRequest(def, "yaml")
	if strings.Contains(string(out), "followRedirects") {
		t.Errorf("the default is not written:\n%s", out)
	}
	on, _ := serialization.ParseRequest([]byte("schemaVersion: \"1\"\nmethod: GET\nurl: http://x\nfollowRedirects: true\n"), "yaml")
	if on.NoFollowRedirects {
		t.Error("followRedirects: true is the default")
	}
}

func TestFollowRedirectsInScenarioStepAndBundle(t *testing.T) {
	sc, err := serialization.ParseScenario([]byte(`schemaVersion: "1"
title: S
steps:
  - label: a
    request: {method: GET, url: "http://x/old", followRedirects: false}
`), "yaml")
	if err != nil || !sc.Steps[0].Request.NoFollowRedirects {
		t.Fatalf("scenario step: %v", err)
	}
	out, err := serialization.WriteScenario(sc, "yaml")
	if err != nil || !strings.Contains(string(out), "followRedirects: false") {
		t.Errorf("scenario write: %v\n%s", err, out)
	}

	b, err := serialization.ParseBundle([]byte(`schemaVersion: "1"
requests:
  - {file: old, method: GET, url: "http://x/old", followRedirects: false}
`), "yaml")
	if err != nil || !b.Requests[0].Request.NoFollowRedirects {
		t.Fatalf("bundle: %v", err)
	}
	var _ domain.Request = *b.Requests[0].Request
	out, err = serialization.WriteBundle(b, "yaml")
	if err != nil || !strings.Contains(string(out), "followRedirects: false") {
		t.Errorf("bundle write: %v\n%s", err, out)
	}
}
