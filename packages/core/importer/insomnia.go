package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// --- Insomnia v4 export JSON structures ---

type insomniaExport struct {
	Type      string              `json:"_type"`
	Format    int                 `json:"__export_format"`
	Resources []insomniaResource  `json:"resources"`
}

type insomniaResource struct {
	ID       string `json:"_id"`
	Type     string `json:"_type"`
	ParentID string `json:"parentId"`
	Name     string `json:"name"`

	// request fields
	Method         string           `json:"method"`
	URL            string           `json:"url"`
	Headers        []insomniaHeader `json:"headers"`
	Parameters     []insomniaParam  `json:"parameters"`
	Body           *insomniaBody    `json:"body"`
	Authentication *insomniaAuth    `json:"authentication"`
}

type insomniaHeader struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type insomniaParam struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type insomniaBody struct {
	MimeType string              `json:"mimeType"`
	Text     string              `json:"text"`
	Params   []insomniaBodyParam `json:"params"`
}

type insomniaBodyParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type insomniaAuth struct {
	Type     string `json:"type"`
	Disabled bool   `json:"disabled"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	AddTo    string `json:"addTo"`
}

// ParseInsomniaExport parses an Insomnia v4 export JSON file.
// Returns the workspace name and a flat list of requests with their folder paths.
func ParseInsomniaExport(data []byte) (name string, requests []ImportedRequest, err error) {
	var exp insomniaExport
	if jsonErr := json.Unmarshal(data, &exp); jsonErr != nil {
		return "", nil, fmt.Errorf("parsing insomnia export: %w", jsonErr)
	}

	// Index all resources by ID
	byID := make(map[string]insomniaResource, len(exp.Resources))
	for _, r := range exp.Resources {
		byID[r.ID] = r
	}

	// Workspace display name
	wsName := "insomnia-import"
	for _, r := range exp.Resources {
		if r.Type == "workspace" {
			wsName = r.Name
			break
		}
	}

	// Memoised folder-path builder (stops at workspace or unknown parent)
	cache := make(map[string]string)
	var folderPath func(id string) string
	folderPath = func(id string) string {
		if p, ok := cache[id]; ok {
			return p
		}
		r, ok := byID[id]
		if !ok || r.Type == "workspace" {
			cache[id] = ""
			return ""
		}
		parent := folderPath(r.ParentID)
		var p string
		if parent == "" {
			p = sanitiseSegment(r.Name)
		} else {
			p = parent + "/" + sanitiseSegment(r.Name)
		}
		cache[id] = p
		return p
	}

	for _, r := range exp.Resources {
		if r.Type != "request" {
			continue
		}
		fp := folderPath(r.ParentID)
		req := convertInsomniaRequest(r)
		requests = append(requests, ImportedRequest{FolderPath: fp, Request: req})
	}

	return wsName, requests, nil
}

func convertInsomniaRequest(r insomniaResource) *domain.Request {
	req := &domain.Request{
		SchemaVersion: "1",
		Title:         r.Name,
		Method:        normaliseMethod(r.Method),
		URL:           r.URL,
	}

	for _, h := range r.Headers {
		if !h.Disabled && h.Name != "" {
			req.Headers = append(req.Headers, domain.Header{Name: h.Name, Value: h.Value})
		}
	}

	for _, p := range r.Parameters {
		if !p.Disabled && p.Name != "" {
			req.QueryParams = append(req.QueryParams, domain.QueryParam{Name: p.Name, Value: p.Value})
		}
	}

	if r.Body != nil {
		req.Body = convertInsomniaBody(r.Body)
	}

	if r.Authentication != nil && !r.Authentication.Disabled {
		req.Auth = convertInsomniaAuth(r.Authentication)
	}

	return req
}

func convertInsomniaBody(b *insomniaBody) *domain.RequestBody {
	switch {
	case strings.Contains(b.MimeType, "json"):
		return &domain.RequestBody{Type: domain.BodyJSON, Content: b.Text}
	case strings.Contains(b.MimeType, "form-urlencoded"):
		var pairs []string
		for _, p := range b.Params {
			pairs = append(pairs, p.Name+"="+p.Value)
		}
		return &domain.RequestBody{Type: domain.BodyFormURLEncoded, Content: strings.Join(pairs, "&")}
	case b.Text != "":
		return &domain.RequestBody{Type: domain.BodyRaw, Content: b.Text}
	}
	return nil
}

func convertInsomniaAuth(a *insomniaAuth) *domain.Auth {
	switch strings.ToLower(a.Type) {
	case "bearer":
		return &domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: a.Token}}
	case "basic":
		return &domain.Auth{
			Type: domain.AuthBasic,
			Basic: &domain.BasicAuth{Username: a.Username, Password: a.Password},
		}
	case "apikey":
		loc := domain.APIKeyInHeader
		if strings.ToLower(a.AddTo) == "queryparams" {
			loc = domain.APIKeyInQuery
		}
		return &domain.Auth{
			Type: domain.AuthAPIKey,
			APIKey: &domain.APIKeyAuth{Key: a.Key, Value: a.Value, Location: loc},
		}
	}
	return nil
}

func sanitiseSegment(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "folder"
	}
	return b.String()
}
