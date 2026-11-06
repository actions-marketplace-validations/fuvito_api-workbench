package serialization

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"gopkg.in/yaml.v3"
)

// RequestFile is the on-disk representation of a Request.
// It maps 1-to-1 with the domain type but carries YAML/JSON tags.
type RequestFile struct {
	SchemaVersion string     `yaml:"schemaVersion" json:"schemaVersion"`
	Title         string     `yaml:"title,omitempty" json:"title,omitempty"`
	Description   string     `yaml:"description,omitempty" json:"description,omitempty"`
	Method        string     `yaml:"method" json:"method"`
	URL           string     `yaml:"url" json:"url"`
	Headers       []KVPair   `yaml:"headers,omitempty" json:"headers,omitempty"`
	QueryParams   []KVPair   `yaml:"queryParams,omitempty" json:"queryParams,omitempty"`
	Body          *BodyFile  `yaml:"body,omitempty" json:"body,omitempty"`
	Auth          *AuthFile  `yaml:"auth,omitempty" json:"auth,omitempty"`
	Timeout       string     `yaml:"timeout,omitempty" json:"timeout,omitempty"` // e.g. "30s"
}

type KVPair struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

type BodyFile struct {
	Type    string `yaml:"type" json:"type"`
	Content string `yaml:"content" json:"content"`
}

type AuthFile struct {
	Type     string `yaml:"type" json:"type"`
	Token    string `yaml:"token,omitempty" json:"token,omitempty"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	Key      string `yaml:"key,omitempty" json:"key,omitempty"`
	Value    string `yaml:"value,omitempty" json:"value,omitempty"`
	Location string `yaml:"location,omitempty" json:"location,omitempty"`
}

// ParseRequest decodes a request file from YAML or JSON bytes.
// format must be "yaml", "yml", or "json".
func ParseRequest(data []byte, format string) (*domain.Request, error) {
	var rf RequestFile
	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &rf); err != nil {
			return nil, fmt.Errorf("parsing yaml: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, &rf); err != nil {
			return nil, fmt.Errorf("parsing json: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown format %q; expected yaml or json", format)
	}
	return requestFileToDomain(&rf)
}

// ParseRequestFile detects format from the file extension.
func ParseRequestFile(data []byte, filename string) (*domain.Request, error) {
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	return ParseRequest(data, ext)
}

// WriteRequest encodes a domain.Request to YAML or JSON.
func WriteRequest(req *domain.Request, format string) ([]byte, error) {
	rf := requestDomainToFile(req)
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return yaml.Marshal(rf)
	case "json":
		return json.MarshalIndent(rf, "", "  ")
	default:
		return nil, fmt.Errorf("unknown format %q; expected yaml or json", format)
	}
}

func requestFileToDomain(rf *RequestFile) (*domain.Request, error) {
	if rf.Method == "" {
		return nil, errors.New("method is required")
	}
	if rf.URL == "" {
		return nil, errors.New("url is required")
	}

	req := &domain.Request{
		SchemaVersion: rf.SchemaVersion,
		Title:         rf.Title,
		Description:   rf.Description,
		Method:        domain.HTTPMethod(strings.ToUpper(rf.Method)),
		URL:           rf.URL,
	}

	for _, h := range rf.Headers {
		req.Headers = append(req.Headers, domain.Header{Name: h.Name, Value: h.Value})
	}
	for _, p := range rf.QueryParams {
		req.QueryParams = append(req.QueryParams, domain.QueryParam{Name: p.Name, Value: p.Value})
	}
	if rf.Body != nil {
		req.Body = &domain.RequestBody{Type: domain.BodyType(rf.Body.Type), Content: rf.Body.Content}
	}
	if rf.Auth != nil {
		req.Auth = authFileToDomain(rf.Auth)
	}
	if rf.Timeout != "" {
		d, err := time.ParseDuration(rf.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", rf.Timeout, err)
		}
		req.Timeout = d
	}

	return req, nil
}

func authFileToDomain(a *AuthFile) *domain.Auth {
	auth := &domain.Auth{Type: domain.AuthType(a.Type)}
	switch auth.Type {
	case domain.AuthBearer:
		auth.Bearer = &domain.BearerAuth{Token: a.Token}
	case domain.AuthBasic:
		auth.Basic = &domain.BasicAuth{Username: a.Username, Password: a.Password}
	case domain.AuthAPIKey:
		auth.APIKey = &domain.APIKeyAuth{Key: a.Key, Value: a.Value, Location: domain.APIKeyLocation(a.Location)}
	}
	return auth
}

func requestDomainToFile(req *domain.Request) *RequestFile {
	rf := &RequestFile{
		SchemaVersion: "1",
		Title:         req.Title,
		Description:   req.Description,
		Method:        string(req.Method),
		URL:           req.URL,
	}
	for _, h := range req.Headers {
		rf.Headers = append(rf.Headers, KVPair{Name: h.Name, Value: h.Value})
	}
	for _, p := range req.QueryParams {
		rf.QueryParams = append(rf.QueryParams, KVPair{Name: p.Name, Value: p.Value})
	}
	if req.Body != nil {
		rf.Body = &BodyFile{Type: string(req.Body.Type), Content: req.Body.Content}
	}
	if req.Auth != nil {
		rf.Auth = authDomainToFile(req.Auth)
	}
	if req.Timeout > 0 {
		rf.Timeout = req.Timeout.String()
	}
	return rf
}

func authDomainToFile(a *domain.Auth) *AuthFile {
	af := &AuthFile{Type: string(a.Type)}
	if a.Bearer != nil {
		af.Token = a.Bearer.Token
	}
	if a.Basic != nil {
		af.Username = a.Basic.Username
		af.Password = a.Basic.Password
	}
	if a.APIKey != nil {
		af.Key = a.APIKey.Key
		af.Value = a.APIKey.Value
		af.Location = string(a.APIKey.Location)
	}
	return af
}
