package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// --- Postman Collection v2.1 JSON structures ---

type postmanCollection struct {
	Info     postmanInfo       `json:"info"`
	Item     []postmanItem     `json:"item"`
	Variable []postmanVariable `json:"variable"`
}

type postmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item"`
	Request *postmanRequest `json:"request"`
}

type postmanRequest struct {
	Method string          `json:"method"`
	URL    postmanURL      `json:"url"`
	Header []postmanHeader `json:"header"`
	Body   *postmanBody    `json:"body"`
	Auth   *postmanAuth    `json:"auth"`
}

// postmanURL is either a string or a structured object in the wild.
type postmanURL struct {
	Raw      string      `json:"raw"`
	Query    []postmanKV `json:"query"`
	Variable []postmanKV `json:"variable"`
}

func (u *postmanURL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		u.Raw = s
		return nil
	}
	type alias postmanURL
	var obj alias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*u = postmanURL(obj)
	return nil
}

type postmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type postmanKV struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type postmanBody struct {
	Mode       string            `json:"mode"`
	Raw        string            `json:"raw"`
	Options    *postmanBodyOpts  `json:"options"`
	URLEncoded []postmanKV       `json:"urlencoded"`
}

type postmanBodyOpts struct {
	Raw *postmanBodyOptsRaw `json:"raw"`
}

type postmanBodyOptsRaw struct {
	Language string `json:"language"`
}

type postmanAuth struct {
	Type   string      `json:"type"`
	Bearer []postmanKV `json:"bearer"`
	Basic  []postmanKV `json:"basic"`
	APIKey []postmanKV `json:"apikey"`
}

type postmanVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ImportedRequest is a domain.Request with the workspace-relative folder path.
type ImportedRequest struct {
	FolderPath string // slash-separated relative path; empty = workspace root
	Request    *domain.Request
}

// ParsePostmanCollection parses a Postman Collection v2.1 JSON file.
// Returns the collection name and a flat list of requests with their folder paths.
func ParsePostmanCollection(data []byte) (name string, requests []ImportedRequest, err error) {
	var col postmanCollection
	if jsonErr := json.Unmarshal(data, &col); jsonErr != nil {
		return "", nil, fmt.Errorf("parsing postman collection: %w", jsonErr)
	}
	if col.Info.Name == "" {
		col.Info.Name = "postman-import"
	}
	return col.Info.Name, flattenPostmanItems(col.Item, ""), nil
}

func flattenPostmanItems(items []postmanItem, folderPath string) []ImportedRequest {
	var result []ImportedRequest
	for _, item := range items {
		if item.Request == nil && len(item.Item) > 0 {
			// folder
			sub := item.Name
			if folderPath != "" {
				sub = folderPath + "/" + item.Name
			}
			result = append(result, flattenPostmanItems(item.Item, sub)...)
		} else if item.Request != nil {
			req := convertPostmanRequest(item)
			result = append(result, ImportedRequest{FolderPath: folderPath, Request: req})
		}
	}
	return result
}

func convertPostmanRequest(item postmanItem) *domain.Request {
	pr := item.Request
	req := &domain.Request{
		SchemaVersion: "1",
		Title:         item.Name,
		Method:        normaliseMethod(pr.Method),
	}

	// URL — strip inline query string; use structured query array instead
	raw := pr.URL.Raw
	if idx := strings.Index(raw, "?"); idx != -1 {
		raw = raw[:idx]
	}
	req.URL = raw

	for _, qp := range pr.URL.Query {
		if !qp.Disabled {
			req.QueryParams = append(req.QueryParams, domain.QueryParam{Name: qp.Key, Value: qp.Value})
		}
	}

	for _, h := range pr.Header {
		if !h.Disabled && h.Key != "" {
			req.Headers = append(req.Headers, domain.Header{Name: h.Key, Value: h.Value})
		}
	}

	if pr.Body != nil {
		req.Body = convertPostmanBody(pr.Body)
	}

	if pr.Auth != nil {
		req.Auth = convertPostmanAuth(pr.Auth)
	}

	return req
}

func convertPostmanBody(b *postmanBody) *domain.RequestBody {
	switch b.Mode {
	case "raw":
		bt := domain.BodyRaw
		if b.Options != nil && b.Options.Raw != nil && b.Options.Raw.Language == "json" {
			bt = domain.BodyJSON
		}
		return &domain.RequestBody{Type: bt, Content: b.Raw}
	case "urlencoded":
		var pairs []string
		for _, kv := range b.URLEncoded {
			if !kv.Disabled {
				pairs = append(pairs, kv.Key+"="+kv.Value)
			}
		}
		return &domain.RequestBody{Type: domain.BodyFormURLEncoded, Content: strings.Join(pairs, "&")}
	}
	return nil
}

func convertPostmanAuth(a *postmanAuth) *domain.Auth {
	switch strings.ToLower(a.Type) {
	case "bearer":
		token := postmanKVFind(a.Bearer, "token")
		return &domain.Auth{Type: domain.AuthBearer, Bearer: &domain.BearerAuth{Token: token}}
	case "basic":
		return &domain.Auth{
			Type: domain.AuthBasic,
			Basic: &domain.BasicAuth{
				Username: postmanKVFind(a.Basic, "username"),
				Password: postmanKVFind(a.Basic, "password"),
			},
		}
	case "apikey":
		loc := domain.APIKeyInHeader
		if strings.ToLower(postmanKVFind(a.APIKey, "in")) == "query" {
			loc = domain.APIKeyInQuery
		}
		return &domain.Auth{
			Type: domain.AuthAPIKey,
			APIKey: &domain.APIKeyAuth{
				Key:      postmanKVFind(a.APIKey, "key"),
				Value:    postmanKVFind(a.APIKey, "value"),
				Location: loc,
			},
		}
	}
	return nil
}

func postmanKVFind(kvs []postmanKV, key string) string {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

func normaliseMethod(m string) domain.HTTPMethod {
	method := domain.HTTPMethod(strings.ToUpper(m))
	if !method.IsValid() {
		return domain.MethodGET
	}
	return method
}
