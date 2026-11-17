package serialization_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

var formats = []string{"yaml", "yml", "json"}

func fullRequest() *domain.Request {
	return &domain.Request{
		Title:       "full",
		Method:      domain.MethodPOST,
		URL:         "https://x.test/a",
		Headers:     []domain.Header{{Name: "X-A", Value: "1"}},
		QueryParams: []domain.QueryParam{{Name: "q", Value: "v"}},
		Body:        &domain.RequestBody{Type: domain.BodyJSON, Content: `{"a":1}`},
		Timeout:     1500 * time.Millisecond,
	}
}

func TestRequest_RoundTripAllAuthTypes(t *testing.T) {
	auths := map[string]*domain.Auth{
		"bearer": {Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: "t"}},
		"basic":  {Type: domain.AuthBasic, Basic: &domain.BasicAuth{Username: "u", Password: "p"}},
		"apikey": {Type: domain.AuthAPIKey, APIKey: &domain.APIKeyAuth{Key: "k", Value: "v", Location: domain.APIKeyInHeader}},
		"none":   {Type: domain.AuthNone},
	}
	for name, auth := range auths {
		for _, f := range formats {
			t.Run(name+"/"+f, func(t *testing.T) {
				in := fullRequest()
				in.Auth = auth
				data, err := serialization.WriteRequest(in, f)
				if err != nil {
					t.Fatal(err)
				}
				out, err := serialization.ParseRequest(data, f)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(out.Auth, auth) {
					t.Errorf("auth lost: %+v vs %+v", out.Auth, auth)
				}
				if out.Timeout != in.Timeout || out.Title != in.Title || out.Body.Content != in.Body.Content || len(out.Headers) != 1 || len(out.QueryParams) != 1 {
					t.Errorf("request lost data: %+v", out)
				}
			})
		}
	}
}

func TestRequest_ParseErrors(t *testing.T) {
	cases := map[string]struct{ data, format, want string }{
		"unknown format":  {"{}", "toml", "unknown format"},
		"bad yaml":        {"method: [", "yaml", "parsing yaml"},
		"bad json":        {"{", "json", "parsing json"},
		"missing method":  {"url: http://x", "yaml", "method is required"},
		"missing url":     {"method: GET", "yaml", "url is required"},
		"invalid timeout": {"method: GET\nurl: http://x\ntimeout: soon", "yaml", "invalid timeout"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := serialization.ParseRequest([]byte(c.data), c.format)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want containing %q", err, c.want)
			}
		})
	}
}

func TestRequest_MethodIsNormalisedAndFormatsAreCaseInsensitive(t *testing.T) {
	r, err := serialization.ParseRequest([]byte("method: post\nurl: http://x"), "YAML")
	if err != nil || r.Method != domain.MethodPOST {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := serialization.WriteRequest(r, "toml"); err == nil {
		t.Error("unknown write format")
	}
}

func TestEnvironment_RoundTripAndErrors(t *testing.T) {
	env := domain.NewEnvironment("local")
	env.Variables = []domain.EnvironmentVariable{{Name: "host", Value: "h"}, {Name: "tok", Value: "s", Secret: true}}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			data, err := serialization.WriteEnvironment(env, f)
			if err != nil {
				t.Fatal(err)
			}
			out, err := serialization.ParseEnvironment(data, f)
			if err != nil {
				t.Fatal(err)
			}
			if out.Name != "local" || len(out.Variables) != 2 || out.Variables[0].Secret || !out.Variables[1].Secret {
				t.Errorf("round trip: %+v", out)
			}
		})
	}

	if _, err := serialization.ParseEnvironment([]byte("{}"), "toml"); err == nil {
		t.Error("unknown format")
	}
	if _, err := serialization.ParseEnvironment([]byte("name: ["), "yaml"); err == nil {
		t.Error("bad yaml")
	}
	if _, err := serialization.ParseEnvironment([]byte("{"), "json"); err == nil {
		t.Error("bad json")
	}
	if _, err := serialization.WriteEnvironment(env, "toml"); err == nil {
		t.Error("unknown write format")
	}
}

func TestParseEnvironmentFile_DetectsFormatFromExtension(t *testing.T) {
	if e, err := serialization.ParseEnvironmentFile([]byte(`{"name":"j"}`), "dir/env.json"); err != nil || e.Name != "j" {
		t.Errorf("json: %+v %v", e, err)
	}
	if e, err := serialization.ParseEnvironmentFile([]byte("name: y"), "env.yml"); err != nil || e.Name != "y" {
		t.Errorf("yml: %+v %v", e, err)
	}
	if _, err := serialization.ParseEnvironmentFile([]byte("name: y"), "env.txt"); err == nil {
		t.Error("unknown extension should fail")
	}
}

func TestScenario_RoundTripAllStepFields(t *testing.T) {
	off := false
	in := &domain.Scenario{
		Title: "s", Description: "d", SpecFile: "spec.yaml",
		Steps: []domain.ScenarioStep{{
			Label: "one", Notes: "n", Disabled: true, RequestRef: "ref.yaml",
			FailPolicy: domain.FailPolicyContinue, TimeoutMs: 900, ValidateSpec: &off,
			Retry:       &domain.RetryPolicy{MaxAttempts: 3, Delay: 250 * time.Millisecond},
			Request:     fullRequest(),
			Assertions:  []domain.Assertion{{Type: domain.AssertionJSONPath, Name: "a", Path: "$.id", Operator: domain.OperatorEq, Expected: "1", Schema: ""}, {Type: domain.AssertionSchema, Schema: "getUser"}},
			Extractions: []domain.Extraction{{Name: "id", Source: domain.ExtractJSONPath, Path: "$.id"}},
		}},
	}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			data, err := serialization.WriteScenario(in, f)
			if err != nil {
				t.Fatal(err)
			}
			out, err := serialization.ParseScenario(data, f)
			if err != nil {
				t.Fatal(err)
			}
			st := out.Steps[0]
			if out.SpecFile != "spec.yaml" || out.Description != "d" || st.Notes != "n" || !st.Disabled || st.RequestRef != "ref.yaml" ||
				st.FailPolicy != domain.FailPolicyContinue || st.TimeoutMs != 900 || st.ValidateSpec == nil || *st.ValidateSpec {
				t.Errorf("step fields lost: %+v", st)
			}
			if st.Retry == nil || st.Retry.MaxAttempts != 3 || st.Retry.Delay != 250*time.Millisecond {
				t.Errorf("retry lost: %+v", st.Retry)
			}
			if len(st.Assertions) != 2 || st.Assertions[0].Name != "a" || st.Assertions[1].Schema != "getUser" {
				t.Errorf("assertions lost: %+v", st.Assertions)
			}
			if len(st.Extractions) != 1 || st.Extractions[0].Path != "$.id" || st.Request == nil || st.Request.Timeout != 1500*time.Millisecond {
				t.Errorf("extractions/request lost: %+v", st)
			}
		})
	}
}

func TestScenario_RetryWithZeroAttemptsIsDropped(t *testing.T) {
	in := &domain.Scenario{Title: "s", Steps: []domain.ScenarioStep{{Label: "a", Retry: &domain.RetryPolicy{MaxAttempts: 0, Delay: time.Second}}}}
	data, _ := serialization.WriteScenario(in, "yaml")
	if strings.Contains(string(data), "retry") {
		t.Errorf("zero-attempt retry should not be written:\n%s", data)
	}
}

func TestScenario_ParseErrors(t *testing.T) {
	if _, err := serialization.ParseScenario([]byte("{}"), "toml"); err == nil {
		t.Error("unknown format")
	}
	if _, err := serialization.ParseScenario([]byte("steps: ["), "yaml"); err == nil {
		t.Error("bad yaml")
	}
	if _, err := serialization.ParseScenario([]byte("{"), "json"); err == nil {
		t.Error("bad json")
	}
	_, err := serialization.ParseScenario([]byte("steps:\n  - label: broken\n    request:\n      method: GET\n"), "yaml")
	if err == nil || !strings.Contains(err.Error(), `step "broken"`) {
		t.Errorf("step errors should name the step: %v", err)
	}
	if _, err := serialization.WriteScenario(&domain.Scenario{}, "toml"); err == nil {
		t.Error("unknown write format")
	}
}

func TestParseScenarioFile_DetectsFormatFromExtension(t *testing.T) {
	if s, err := serialization.ParseScenarioFile([]byte(`{"title":"j","steps":[]}`), "x.json"); err != nil || s.Title != "j" {
		t.Errorf("json: %+v %v", s, err)
	}
	if s, err := serialization.ParseScenarioFile([]byte("title: y"), "x.yaml"); err != nil || s.Title != "y" {
		t.Errorf("yaml: %+v %v", s, err)
	}
	if _, err := serialization.ParseScenarioFile([]byte("title: y"), "x"); err == nil {
		t.Error("missing extension should fail")
	}
}
