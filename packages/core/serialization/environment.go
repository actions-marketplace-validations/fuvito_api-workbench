package serialization

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"gopkg.in/yaml.v3"
)

// EnvironmentFile is the on-disk representation of an Environment.
type EnvironmentFile struct {
	SchemaVersion string   `yaml:"schemaVersion" json:"schemaVersion"`
	Name          string   `yaml:"name" json:"name"`
	Variables     []EnvVar `yaml:"variables" json:"variables"`
}

type EnvVar struct {
	Name   string `yaml:"name" json:"name"`
	Value  string `yaml:"value" json:"value"`
	Secret bool   `yaml:"secret,omitempty" json:"secret,omitempty"`
}

// ParseEnvironment decodes an environment file from YAML or JSON bytes.
func ParseEnvironment(data []byte, format string) (*domain.Environment, error) {
	var ef EnvironmentFile
	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &ef); err != nil {
			return nil, fmt.Errorf("parsing yaml: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, &ef); err != nil {
			return nil, fmt.Errorf("parsing json: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown format %q; expected yaml or json", format)
	}
	return envFileToDomain(&ef), nil
}

// ParseEnvironmentFile detects format from the file extension.
func ParseEnvironmentFile(data []byte, filename string) (*domain.Environment, error) {
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	return ParseEnvironment(data, ext)
}

// WriteEnvironment encodes a domain.Environment to YAML or JSON.
func WriteEnvironment(env *domain.Environment, format string) ([]byte, error) {
	ef := envDomainToFile(env)
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return yaml.Marshal(ef)
	case "json":
		return json.MarshalIndent(ef, "", "  ")
	default:
		return nil, fmt.Errorf("unknown format %q; expected yaml or json", format)
	}
}

func envFileToDomain(ef *EnvironmentFile) *domain.Environment {
	env := domain.NewEnvironment(ef.Name)
	for _, v := range ef.Variables {
		env.Variables = append(env.Variables, domain.EnvironmentVariable{
			Name:   v.Name,
			Value:  v.Value,
			Secret: v.Secret,
		})
	}
	return env
}

func envDomainToFile(env *domain.Environment) *EnvironmentFile {
	ef := &EnvironmentFile{
		SchemaVersion: "1",
		Name:          env.Name,
	}
	for _, v := range env.Variables {
		ef.Variables = append(ef.Variables, EnvVar{
			Name:   v.Name,
			Value:  v.Value,
			Secret: v.Secret,
		})
	}
	return ef
}
