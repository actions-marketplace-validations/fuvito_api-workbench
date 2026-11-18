package serialization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"gopkg.in/yaml.v3"
)

// BundleFile is the on-disk representation of a bundle: one document that carries
// several environments, reusable requests and scenarios, typically produced by an
// AI assistant and imported into a workspace in one step.
//
// Every item reuses the normal file format, plus a "file" key that names the file it
// will be saved as (without folder prefix or extension requirements). Bundles are
// decoded strictly: an unknown field is an error, so typos and misplaced fields in
// generated output are caught at import time instead of being silently ignored.
type BundleFile struct {
	SchemaVersion string                  `yaml:"schemaVersion" json:"schemaVersion"`
	Title         string                  `yaml:"title,omitempty" json:"title,omitempty"`
	Description   string                  `yaml:"description,omitempty" json:"description,omitempty"`
	SpecFile      string                  `yaml:"specFile,omitempty" json:"specFile,omitempty"`
	Environments  []BundleEnvironmentFile `yaml:"environments,omitempty" json:"environments,omitempty"`
	Requests      []BundleRequestFile     `yaml:"requests,omitempty" json:"requests,omitempty"`
	Scenarios     []BundleScenarioFile    `yaml:"scenarios,omitempty" json:"scenarios,omitempty"`
}

// BundleEnvironmentFile is an environment inside a bundle.
type BundleEnvironmentFile struct {
	File      string   `yaml:"file,omitempty" json:"file,omitempty"`
	Name      string   `yaml:"name" json:"name"`
	Variables []EnvVar `yaml:"variables" json:"variables"`
}

// BundleRequestFile is a reusable request inside a bundle. Scenarios refer to it with
// requestRef: <file>.
type BundleRequestFile struct {
	File        string    `yaml:"file" json:"file"`
	Title       string    `yaml:"title,omitempty" json:"title,omitempty"`
	Method      string    `yaml:"method" json:"method"`
	URL         string    `yaml:"url" json:"url"`
	Headers     []KVPair  `yaml:"headers,omitempty" json:"headers,omitempty"`
	QueryParams []KVPair  `yaml:"queryParams,omitempty" json:"queryParams,omitempty"`
	Body        *BodyFile `yaml:"body,omitempty" json:"body,omitempty"`
	Auth        *AuthFile `yaml:"auth,omitempty" json:"auth,omitempty"`
	Timeout     string    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// BundleScenarioFile is a scenario inside a bundle.
type BundleScenarioFile struct {
	File        string     `yaml:"file,omitempty" json:"file,omitempty"`
	Title       string     `yaml:"title" json:"title"`
	Description string     `yaml:"description,omitempty" json:"description,omitempty"`
	SpecFile    string     `yaml:"specFile,omitempty" json:"specFile,omitempty"`
	Variables   []EnvVar   `yaml:"variables,omitempty" json:"variables,omitempty"`
	Steps       []StepFile `yaml:"steps" json:"steps"`
}

// Bundle is the parsed, domain-level form of a BundleFile.
type Bundle struct {
	SchemaVersion string
	Title         string
	Description   string
	// SpecFile is the default OpenAPI spec (path or URL) for scenarios that do not set their own.
	SpecFile     string
	Environments []BundleEnvironment
	Requests     []BundleRequest
	Scenarios    []BundleScenario
}

type BundleEnvironment struct {
	File        string // optional; derived from the name when empty
	Environment *domain.Environment
}

type BundleRequest struct {
	File    string
	Request *domain.Request
}

type BundleScenario struct {
	File     string // optional; derived from the title when empty
	Scenario *domain.Scenario
}

// ParseBundle decodes a bundle from YAML or JSON bytes. format is "yaml", "yml" or "json".
func ParseBundle(data []byte, format string) (*Bundle, error) {
	var bf BundleFile
	switch strings.ToLower(format) {
	case "yaml", "yml":
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&bf); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errors.New("bundle is empty")
			}
			return nil, fmt.Errorf("parsing yaml: %s", friendlyDecodeError(err))
		}
	case "json":
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&bf); err != nil {
			return nil, fmt.Errorf("parsing json: %s", friendlyDecodeError(err))
		}
	default:
		return nil, fmt.Errorf("unknown format %q; expected yaml or json", format)
	}
	return bundleFileToDomain(&bf)
}

func bundleFileToDomain(bf *BundleFile) (*Bundle, error) {
	b := &Bundle{
		SchemaVersion: bf.SchemaVersion,
		Title:         bf.Title,
		Description:   bf.Description,
		SpecFile:      bf.SpecFile,
	}
	for _, e := range bf.Environments {
		ef := EnvironmentFile{SchemaVersion: "1", Name: e.Name, Variables: e.Variables}
		b.Environments = append(b.Environments, BundleEnvironment{File: e.File, Environment: envFileToDomain(&ef)})
	}
	for i, r := range bf.Requests {
		rf := RequestFile{
			SchemaVersion: "1", Title: r.Title, Method: r.Method, URL: r.URL, Headers: r.Headers,
			QueryParams: r.QueryParams, Body: r.Body, Auth: r.Auth, Timeout: r.Timeout,
		}
		req, err := requestFileToDomain(&rf)
		if err != nil {
			return nil, fmt.Errorf("requests[%d] %q: %w", i, r.File, err)
		}
		b.Requests = append(b.Requests, BundleRequest{File: r.File, Request: req})
	}
	for i, s := range bf.Scenarios {
		sf := ScenarioFile{SchemaVersion: "1", Title: s.Title, Description: s.Description, SpecFile: s.SpecFile, Variables: s.Variables, Steps: s.Steps}
		sc, err := scenarioFileToDomain(&sf)
		if err != nil {
			return nil, fmt.Errorf("scenarios[%d] %q: %w", i, s.Title, err)
		}
		b.Scenarios = append(b.Scenarios, BundleScenario{File: s.File, Scenario: sc})
	}
	return b, nil
}

// WriteBundle encodes a bundle to YAML or JSON. Items are written in the order given.
func WriteBundle(b *Bundle, format string) ([]byte, error) {
	bf := bundleDomainToFile(b)
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return yaml.Marshal(bf)
	case "json":
		return json.MarshalIndent(bf, "", "  ")
	default:
		return nil, fmt.Errorf("unknown format %q; expected yaml or json", format)
	}
}

func bundleDomainToFile(b *Bundle) *BundleFile {
	version := b.SchemaVersion
	if version == "" {
		version = "1"
	}
	bf := &BundleFile{SchemaVersion: version, Title: b.Title, Description: b.Description, SpecFile: b.SpecFile}
	for _, e := range b.Environments {
		ef := envDomainToFile(e.Environment)
		bf.Environments = append(bf.Environments, BundleEnvironmentFile{File: e.File, Name: ef.Name, Variables: ef.Variables})
	}
	for _, r := range b.Requests {
		rf := requestDomainToFile(r.Request)
		bf.Requests = append(bf.Requests, BundleRequestFile{
			File: r.File, Title: rf.Title, Method: rf.Method, URL: rf.URL, Headers: rf.Headers,
			QueryParams: rf.QueryParams, Body: rf.Body, Auth: rf.Auth, Timeout: rf.Timeout,
		})
	}
	for _, s := range b.Scenarios {
		sf := scenarioDomainToFile(s.Scenario)
		bf.Scenarios = append(bf.Scenarios, BundleScenarioFile{
			File: s.File, Title: sf.Title, Description: sf.Description, SpecFile: sf.SpecFile, Variables: sf.Variables, Steps: sf.Steps,
		})
	}
	return bf
}

var unknownFieldRe = regexp.MustCompile(`field (\S+) not found in type \S+`)

// friendlyDecodeError rewrites decoder messages so authors of bundle files (often an AI
// assistant) see `unknown field "timeoutMs"` rather than Go type names.
func friendlyDecodeError(err error) string {
	return unknownFieldRe.ReplaceAllString(err.Error(), `unknown field "$1"`)
}
