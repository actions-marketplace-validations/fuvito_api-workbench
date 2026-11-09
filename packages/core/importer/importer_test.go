package importer_test

import (
	"encoding/json"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/importer"
)

// ---- Postman ----

const postmanBasic = `{
  "info": {"name": "My API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "item": [
    {
      "name": "get users",
      "request": {
        "method": "GET",
        "url": {"raw": "https://api.example.com/users", "query": [{"key": "page", "value": "1", "disabled": false}]},
        "header": [{"key": "Accept", "value": "application/json", "disabled": false}]
      }
    }
  ]
}`

func TestParsePostman_Name(t *testing.T) {
	name, _, err := importer.ParsePostmanCollection([]byte(postmanBasic))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "My API" {
		t.Errorf("name: got %q, want %q", name, "My API")
	}
}

func TestParsePostman_RequestCount(t *testing.T) {
	_, reqs, err := importer.ParsePostmanCollection([]byte(postmanBasic))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("want 1 request, got %d", len(reqs))
	}
}

func TestParsePostman_Method(t *testing.T) {
	_, reqs, _ := importer.ParsePostmanCollection([]byte(postmanBasic))
	if reqs[0].Request.Method != domain.MethodGET {
		t.Errorf("method: got %q", reqs[0].Request.Method)
	}
}

func TestParsePostman_URL(t *testing.T) {
	_, reqs, _ := importer.ParsePostmanCollection([]byte(postmanBasic))
	if reqs[0].Request.URL != "https://api.example.com/users" {
		t.Errorf("url: got %q", reqs[0].Request.URL)
	}
}

func TestParsePostman_QueryParams(t *testing.T) {
	_, reqs, _ := importer.ParsePostmanCollection([]byte(postmanBasic))
	req := reqs[0].Request
	if len(req.QueryParams) != 1 || req.QueryParams[0].Name != "page" || req.QueryParams[0].Value != "1" {
		t.Errorf("query params: got %+v", req.QueryParams)
	}
}

func TestParsePostman_Headers(t *testing.T) {
	_, reqs, _ := importer.ParsePostmanCollection([]byte(postmanBasic))
	req := reqs[0].Request
	if len(req.Headers) != 1 || req.Headers[0].Name != "Accept" {
		t.Errorf("headers: got %+v", req.Headers)
	}
}

func TestParsePostman_DisabledQueryParamSkipped(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"GET","url":{"raw":"https://x.com","query":[{"key":"skip","value":"v","disabled":true}]}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	if len(reqs[0].Request.QueryParams) != 0 {
		t.Error("disabled query param should be skipped")
	}
}

func TestParsePostman_DisabledHeaderSkipped(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"GET","url":{"raw":"https://x.com"},"header":[{"key":"X-Skip","value":"v","disabled":true}]}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	if len(reqs[0].Request.Headers) != 0 {
		t.Error("disabled header should be skipped")
	}
}

func TestParsePostman_URLAsString(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"POST","url":"https://api.example.com/items"}}]}`
	_, reqs, err := importer.ParsePostmanCollection([]byte(col))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].Request.URL != "https://api.example.com/items" {
		t.Errorf("url: got %q", reqs[0].Request.URL)
	}
}

func TestParsePostman_Folders(t *testing.T) {
	col := `{
	  "info": {"name": "api"},
	  "item": [
	    {
	      "name": "users",
	      "item": [
	        {"name": "list", "request": {"method": "GET", "url": {"raw": "https://x.com/users"}}},
	        {
	          "name": "admin",
	          "item": [
	            {"name": "create", "request": {"method": "POST", "url": {"raw": "https://x.com/users"}}}
	          ]
	        }
	      ]
	    }
	  ]
	}`
	_, reqs, err := importer.ParsePostmanCollection([]byte(col))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("want 2 requests, got %d", len(reqs))
	}
	if reqs[0].FolderPath != "users" {
		t.Errorf("folderPath: got %q, want %q", reqs[0].FolderPath, "users")
	}
	if reqs[1].FolderPath != "users/admin" {
		t.Errorf("nested folderPath: got %q, want %q", reqs[1].FolderPath, "users/admin")
	}
}

func TestParsePostman_BearerAuth(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"GET","url":{"raw":"https://x.com"},"auth":{"type":"bearer","bearer":[{"key":"token","value":"mytoken"}]}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	auth := reqs[0].Request.Auth
	if auth == nil || auth.Type != domain.AuthBearer {
		t.Fatalf("want bearer auth, got %+v", auth)
	}
	if auth.Bearer.Token != "mytoken" {
		t.Errorf("token: got %q", auth.Bearer.Token)
	}
}

func TestParsePostman_BasicAuth(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"GET","url":{"raw":"https://x.com"},"auth":{"type":"basic","basic":[{"key":"username","value":"usr"},{"key":"password","value":"pwd"}]}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	auth := reqs[0].Request.Auth
	if auth == nil || auth.Type != domain.AuthBasic {
		t.Fatalf("want basic auth, got %+v", auth)
	}
	if auth.Basic.Username != "usr" || auth.Basic.Password != "pwd" {
		t.Errorf("credentials: got %+v", auth.Basic)
	}
}

func TestParsePostman_APIKeyHeaderAuth(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"GET","url":{"raw":"https://x.com"},"auth":{"type":"apikey","apikey":[{"key":"key","value":"X-API-Key"},{"key":"value","value":"secret"},{"key":"in","value":"header"}]}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	auth := reqs[0].Request.Auth
	if auth == nil || auth.Type != domain.AuthAPIKey {
		t.Fatalf("want apikey auth, got %+v", auth)
	}
	if auth.APIKey.Location != domain.APIKeyInHeader {
		t.Errorf("location: got %v", auth.APIKey.Location)
	}
}

func TestParsePostman_APIKeyQueryAuth(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"GET","url":{"raw":"https://x.com"},"auth":{"type":"apikey","apikey":[{"key":"key","value":"token"},{"key":"value","value":"abc"},{"key":"in","value":"query"}]}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	auth := reqs[0].Request.Auth
	if auth == nil || auth.APIKey.Location != domain.APIKeyInQuery {
		t.Errorf("want query location, got %+v", auth)
	}
}

func TestParsePostman_JSONBody(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"POST","url":{"raw":"https://x.com"},"body":{"mode":"raw","raw":"{\"name\":\"alice\"}","options":{"raw":{"language":"json"}}}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	body := reqs[0].Request.Body
	if body == nil || body.Type != domain.BodyJSON {
		t.Fatalf("want JSON body, got %+v", body)
	}
	if body.Content != `{"name":"alice"}` {
		t.Errorf("content: got %q", body.Content)
	}
}

func TestParsePostman_FormBody(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"POST","url":{"raw":"https://x.com"},"body":{"mode":"urlencoded","urlencoded":[{"key":"a","value":"1","disabled":false}]}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	body := reqs[0].Request.Body
	if body == nil || body.Type != domain.BodyFormURLEncoded {
		t.Fatalf("want form body, got %+v", body)
	}
}

func TestParsePostman_InvalidJSON(t *testing.T) {
	_, _, err := importer.ParsePostmanCollection([]byte("{not valid json"))
	if err == nil {
		t.Error("want error for invalid JSON")
	}
}

func TestParsePostman_EmptyName(t *testing.T) {
	col := `{"info":{},"item":[]}`
	name, _, _ := importer.ParsePostmanCollection([]byte(col))
	if name == "" {
		t.Error("want default name when info.name is empty")
	}
}

func TestParsePostman_InvalidMethod(t *testing.T) {
	col := `{"info":{"name":"t"},"item":[{"name":"r","request":{"method":"BOGUS","url":{"raw":"https://x.com"}}}]}`
	_, reqs, _ := importer.ParsePostmanCollection([]byte(col))
	if reqs[0].Request.Method != domain.MethodGET {
		t.Errorf("invalid method should default to GET, got %q", reqs[0].Request.Method)
	}
}

// ---- Insomnia ----

func makeInsomniaExport(resources []map[string]interface{}) []byte {
	exp := map[string]interface{}{
		"_type":           "export",
		"__export_format": 4,
		"resources":       resources,
	}
	data, _ := json.Marshal(exp)
	return data
}

func TestParseInsomnia_Name(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "My Workspace", "parentId": ""},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "get users", "method": "GET", "url": "https://api.example.com/users"},
	})
	name, _, err := importer.ParseInsomniaExport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "My Workspace" {
		t.Errorf("name: got %q", name)
	}
}

func TestParseInsomnia_RequestCount(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api", "parentId": ""},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r1", "method": "GET", "url": "https://x.com/a"},
		{"_id": "req_2", "_type": "request", "parentId": "wrk_1", "name": "r2", "method": "POST", "url": "https://x.com/b"},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	if len(reqs) != 2 {
		t.Fatalf("want 2 requests, got %d", len(reqs))
	}
}

func TestParseInsomnia_Folders(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api", "parentId": ""},
		{"_id": "fld_1", "_type": "request_group", "parentId": "wrk_1", "name": "Users"},
		{"_id": "req_1", "_type": "request", "parentId": "fld_1", "name": "list", "method": "GET", "url": "https://x.com/users"},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	if len(reqs) != 1 {
		t.Fatalf("want 1 request, got %d", len(reqs))
	}
	if reqs[0].FolderPath != "Users" {
		t.Errorf("folderPath: got %q, want %q", reqs[0].FolderPath, "Users")
	}
}

func TestParseInsomnia_Headers(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com",
			"headers": []map[string]interface{}{
				{"name": "Accept", "value": "application/json", "disabled": false},
				{"name": "X-Skip", "value": "v", "disabled": true},
			},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	if len(reqs[0].Request.Headers) != 1 || reqs[0].Request.Headers[0].Name != "Accept" {
		t.Errorf("headers: got %+v", reqs[0].Request.Headers)
	}
}

func TestParseInsomnia_QueryParams(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com",
			"parameters": []map[string]interface{}{
				{"name": "page", "value": "2", "disabled": false},
			},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	params := reqs[0].Request.QueryParams
	if len(params) != 1 || params[0].Name != "page" || params[0].Value != "2" {
		t.Errorf("params: got %+v", params)
	}
}

func TestParseInsomnia_JSONBody(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "POST", "url": "https://x.com",
			"body": map[string]interface{}{"mimeType": "application/json", "text": `{"name":"alice"}`},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	body := reqs[0].Request.Body
	if body == nil || body.Type != domain.BodyJSON {
		t.Fatalf("want JSON body, got %+v", body)
	}
}

func TestParseInsomnia_FormBody(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "POST", "url": "https://x.com",
			"body": map[string]interface{}{
				"mimeType": "application/x-www-form-urlencoded",
				"params":   []map[string]interface{}{{"name": "a", "value": "1"}},
			},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	body := reqs[0].Request.Body
	if body == nil || body.Type != domain.BodyFormURLEncoded {
		t.Fatalf("want form body, got %+v", body)
	}
}

func TestParseInsomnia_RawBody(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "POST", "url": "https://x.com",
			"body": map[string]interface{}{"mimeType": "text/plain", "text": "raw data"},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	body := reqs[0].Request.Body
	if body == nil || body.Type != domain.BodyRaw {
		t.Fatalf("want raw body, got %+v", body)
	}
}

func TestParseInsomnia_BearerAuth(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com",
			"authentication": map[string]interface{}{"type": "bearer", "token": "tok123", "disabled": false},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	auth := reqs[0].Request.Auth
	if auth == nil || auth.Type != domain.AuthBearer || auth.Bearer.Token != "tok123" {
		t.Errorf("auth: got %+v", auth)
	}
}

func TestParseInsomnia_BasicAuth(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com",
			"authentication": map[string]interface{}{"type": "basic", "username": "usr", "password": "pwd", "disabled": false},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	auth := reqs[0].Request.Auth
	if auth == nil || auth.Type != domain.AuthBasic {
		t.Fatalf("want basic auth, got %+v", auth)
	}
	if auth.Basic.Username != "usr" || auth.Basic.Password != "pwd" {
		t.Errorf("credentials: got %+v", auth.Basic)
	}
}

func TestParseInsomnia_APIKeyHeaderAuth(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com",
			"authentication": map[string]interface{}{"type": "apikey", "key": "X-Key", "value": "secret", "addTo": "header", "disabled": false},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	auth := reqs[0].Request.Auth
	if auth == nil || auth.Type != domain.AuthAPIKey || auth.APIKey.Location != domain.APIKeyInHeader {
		t.Errorf("auth: got %+v", auth)
	}
}

func TestParseInsomnia_APIKeyQueryAuth(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com",
			"authentication": map[string]interface{}{"type": "apikey", "key": "k", "value": "v", "addTo": "queryparams", "disabled": false},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	auth := reqs[0].Request.Auth
	if auth == nil || auth.APIKey.Location != domain.APIKeyInQuery {
		t.Errorf("want query location, got %+v", auth)
	}
}

func TestParseInsomnia_DisabledAuthSkipped(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "wrk_1", "_type": "workspace", "name": "api"},
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com",
			"authentication": map[string]interface{}{"type": "bearer", "token": "tok", "disabled": true},
		},
	})
	_, reqs, _ := importer.ParseInsomniaExport(data)
	if reqs[0].Request.Auth != nil {
		t.Error("disabled auth should be skipped")
	}
}

func TestParseInsomnia_InvalidJSON(t *testing.T) {
	_, _, err := importer.ParseInsomniaExport([]byte("{not valid"))
	if err == nil {
		t.Error("want error for invalid JSON")
	}
}

func TestParseInsomnia_DefaultWorkspaceName(t *testing.T) {
	data := makeInsomniaExport([]map[string]interface{}{
		{"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "r", "method": "GET", "url": "https://x.com"},
	})
	name, _, _ := importer.ParseInsomniaExport(data)
	if name == "" {
		t.Error("want default workspace name")
	}
}
