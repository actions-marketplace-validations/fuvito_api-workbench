package serialization

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"gopkg.in/yaml.v3"
)

// ScenarioFile is the on-disk representation of a Scenario.
type ScenarioFile struct {
	SchemaVersion string      `yaml:"schemaVersion" json:"schemaVersion"`
	Name          string      `yaml:"name" json:"name"`
	Description   string      `yaml:"description,omitempty" json:"description,omitempty"`
	Steps         []StepFile  `yaml:"steps" json:"steps"`
}

type StepFile struct {
	Name        string           `yaml:"name" json:"name"`
	RequestRef  string           `yaml:"requestRef,omitempty" json:"requestRef,omitempty"`
	Request     *RequestFile     `yaml:"request,omitempty" json:"request,omitempty"`
	Assertions  []AssertionFile  `yaml:"assertions,omitempty" json:"assertions,omitempty"`
	Extractions []ExtractionFile `yaml:"extractions,omitempty" json:"extractions,omitempty"`
	FailPolicy  string           `yaml:"failPolicy,omitempty" json:"failPolicy,omitempty"`
}

type AssertionFile struct {
	Type     string `yaml:"type" json:"type"`
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	Path     string `yaml:"path,omitempty" json:"path,omitempty"`
	Operator string `yaml:"operator,omitempty" json:"operator,omitempty"`
	Expected any    `yaml:"expected" json:"expected"`
	Schema   string `yaml:"schema,omitempty" json:"schema,omitempty"`
}

type ExtractionFile struct {
	Name   string `yaml:"name" json:"name"`
	Source string `yaml:"source" json:"source"`
	Path   string `yaml:"path,omitempty" json:"path,omitempty"`
}

// ParseScenario decodes a scenario file from YAML or JSON bytes.
func ParseScenario(data []byte, format string) (*domain.Scenario, error) {
	var sf ScenarioFile
	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &sf); err != nil {
			return nil, fmt.Errorf("parsing yaml: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, &sf); err != nil {
			return nil, fmt.Errorf("parsing json: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}
	return scenarioFileToDomain(&sf)
}

// ParseScenarioFile detects format from the file extension.
func ParseScenarioFile(data []byte, filename string) (*domain.Scenario, error) {
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	return ParseScenario(data, ext)
}

// WriteScenario encodes a domain.Scenario to YAML or JSON.
func WriteScenario(s *domain.Scenario, format string) ([]byte, error) {
	sf := scenarioDomainToFile(s)
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return yaml.Marshal(sf)
	case "json":
		return json.MarshalIndent(sf, "", "  ")
	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}
}

func scenarioFileToDomain(sf *ScenarioFile) (*domain.Scenario, error) {
	s := &domain.Scenario{
		SchemaVersion: sf.SchemaVersion,
		Name:          sf.Name,
		Description:   sf.Description,
	}
	for _, step := range sf.Steps {
		ds, err := stepFileToDomain(&step)
		if err != nil {
			return nil, fmt.Errorf("step %q: %w", step.Name, err)
		}
		s.Steps = append(s.Steps, *ds)
	}
	return s, nil
}

func stepFileToDomain(sf *StepFile) (*domain.ScenarioStep, error) {
	step := &domain.ScenarioStep{
		Name:       sf.Name,
		RequestRef: sf.RequestRef,
		FailPolicy: domain.FailPolicy(sf.FailPolicy),
	}
	if sf.Request != nil {
		req, err := requestFileToDomain(sf.Request)
		if err != nil {
			return nil, err
		}
		step.Request = req
	}
	for _, a := range sf.Assertions {
		step.Assertions = append(step.Assertions, domain.Assertion{
			Type:     domain.AssertionType(a.Type),
			Name:     a.Name,
			Path:     a.Path,
			Operator: domain.Operator(a.Operator),
			Expected: a.Expected,
			Schema:   a.Schema,
		})
	}
	for _, e := range sf.Extractions {
		step.Extractions = append(step.Extractions, domain.Extraction{
			Name:   e.Name,
			Source: domain.ExtractionSource(e.Source),
			Path:   e.Path,
		})
	}
	return step, nil
}

func scenarioDomainToFile(s *domain.Scenario) *ScenarioFile {
	sf := &ScenarioFile{
		SchemaVersion: "1",
		Name:          s.Name,
		Description:   s.Description,
	}
	for _, step := range s.Steps {
		sf.Steps = append(sf.Steps, stepDomainToFile(&step))
	}
	return sf
}

func stepDomainToFile(step *domain.ScenarioStep) StepFile {
	sf := StepFile{
		Name:       step.Name,
		RequestRef: step.RequestRef,
		FailPolicy: string(step.FailPolicy),
	}
	if step.Request != nil {
		sf.Request = requestDomainToFile(step.Request)
	}
	for _, a := range step.Assertions {
		sf.Assertions = append(sf.Assertions, AssertionFile{
			Type:     string(a.Type),
			Name:     a.Name,
			Path:     a.Path,
			Operator: string(a.Operator),
			Expected: a.Expected,
			Schema:   a.Schema,
		})
	}
	for _, e := range step.Extractions {
		sf.Extractions = append(sf.Extractions, ExtractionFile{
			Name:   e.Name,
			Source: string(e.Source),
			Path:   e.Path,
		})
	}
	return sf
}
