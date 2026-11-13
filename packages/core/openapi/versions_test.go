package openapi_test

// Cross-version compatibility tests.
//
// Covers: Swagger 1.x, Swagger 2.0 (JSON + YAML), OpenAPI 3.0.x, OpenAPI 3.1.x.
// Each fixture exercises multiple HTTP methods, path/query/header/body parameters,
// security schemes, $ref resolution, formData, and array bodies so regressions
// surface quickly regardless of which code path is exercised.

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// ---- Fixtures ----------------------------------------------------------------

// Minimal but realistic Swagger 2.0 JSON spec.
// Covers: GET/POST/PUT/DELETE, path param, query param, required header,
// $ref body, array body, formData (text + file), apiKey + oauth2 security.
const sw2JSON = `{
  "swagger": "2.0",
  "info": {"title": "Bookstore", "version": "2.1"},
  "host": "api.books.io",
  "basePath": "/v1",
  "schemes": ["https"],
  "securityDefinitions": {
    "ApiKeyAuth": {"type": "apiKey", "name": "X-API-Key", "in": "header"},
    "OAuth2":     {"type": "oauth2", "flow": "implicit", "authorizationUrl": "https://auth.books.io/oauth"}
  },
  "definitions": {
    "Book": {
      "type": "object",
      "required": ["title", "isbn"],
      "properties": {
        "id":     {"type": "integer"},
        "title":  {"type": "string"},
        "isbn":   {"type": "string"},
        "author": {"type": "string"}
      }
    },
    "Review": {
      "type": "object",
      "properties": {
        "rating": {"type": "integer"},
        "text":   {"type": "string"}
      }
    }
  },
  "paths": {
    "/books": {
      "get": {
        "operationId": "listBooks",
        "summary": "List books",
        "tags": ["books"],
        "parameters": [
          {"name": "genre",    "in": "query",  "required": false, "type": "string"},
          {"name": "limit",    "in": "query",  "required": true,  "type": "integer"},
          {"name": "X-Locale","in": "header", "required": true,  "type": "string"}
        ]
      },
      "post": {
        "operationId": "createBook",
        "summary": "Create a book",
        "tags": ["books"],
        "parameters": [
          {"name": "body", "in": "body", "required": true,
           "schema": {"$ref": "#/definitions/Book"}}
        ],
        "security": [{"ApiKeyAuth": []}]
      }
    },
    "/books/{bookId}": {
      "get": {
        "operationId": "getBook",
        "summary": "Get book by ID",
        "tags": ["books"],
        "parameters": [
          {"name": "bookId", "in": "path", "required": true, "type": "integer"}
        ]
      },
      "put": {
        "operationId": "updateBook",
        "summary": "Update a book",
        "tags": ["books"],
        "parameters": [
          {"name": "bookId", "in": "path", "required": true, "type": "integer"},
          {"name": "body",   "in": "body", "required": true,
           "schema": {"$ref": "#/definitions/Book"}}
        ],
        "security": [{"OAuth2": ["write"]}]
      },
      "delete": {
        "operationId": "deleteBook",
        "summary": "Delete a book",
        "tags": ["books"],
        "parameters": [
          {"name": "bookId", "in": "path", "required": true, "type": "integer"}
        ],
        "security": [{"ApiKeyAuth": []}]
      }
    },
    "/books/{bookId}/cover": {
      "post": {
        "operationId": "uploadCover",
        "summary": "Upload cover image",
        "tags": ["books"],
        "parameters": [
          {"name": "bookId", "in": "path",     "required": true,  "type": "integer"},
          {"name": "caption","in": "formData", "required": false, "type": "string"},
          {"name": "image",  "in": "formData", "required": true,  "type": "file"}
        ],
        "security": [{"ApiKeyAuth": []}]
      }
    },
    "/books/search": {
      "post": {
        "operationId": "searchBooks",
        "summary": "Full-text search",
        "tags": ["books"],
        "parameters": [
          {"name": "q",    "in": "formData", "required": true,  "type": "string"},
          {"name": "page", "in": "formData", "required": false, "type": "integer"}
        ]
      }
    },
    "/books/bulk": {
      "post": {
        "operationId": "bulkCreateBooks",
        "summary": "Create books in bulk",
        "tags": ["books"],
        "parameters": [
          {"name": "body", "in": "body", "required": true,
           "schema": {"type": "array", "items": {"$ref": "#/definitions/Book"}}}
        ],
        "security": [{"ApiKeyAuth": []}]
      }
    },
    "/books/{bookId}/reviews": {
      "get": {
        "operationId": "listReviews",
        "summary": "List reviews for a book",
        "tags": ["reviews"],
        "parameters": [
          {"name": "bookId", "in": "path",  "required": true,  "type": "integer"},
          {"name": "sort",   "in": "query", "required": false, "type": "string"}
        ]
      },
      "post": {
        "operationId": "createReview",
        "summary": "Post a review",
        "tags": ["reviews"],
        "parameters": [
          {"name": "bookId", "in": "path", "required": true, "type": "integer"},
          {"name": "body",   "in": "body", "required": true,
           "schema": {"$ref": "#/definitions/Review"}}
        ],
        "security": [{"OAuth2": ["reviews:write"]}]
      }
    }
  }
}`

// Same spec content in YAML format — verifies the YAML parsing path.
const sw2YAML = `swagger: "2.0"
info:
  title: Bookstore YAML
  version: "2.1"
host: api.books.io
basePath: /v1
schemes:
  - https
securityDefinitions:
  ApiKeyAuth:
    type: apiKey
    name: X-API-Key
    in: header
definitions:
  Book:
    type: object
    required:
      - title
      - isbn
    properties:
      id:
        type: integer
      title:
        type: string
      isbn:
        type: string
      author:
        type: string
paths:
  /books:
    get:
      operationId: listBooks
      summary: List books
      tags:
        - books
      parameters:
        - name: limit
          in: query
          required: true
          type: integer
    post:
      operationId: createBook
      summary: Create a book
      tags:
        - books
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: "#/definitions/Book"
      security:
        - ApiKeyAuth: []
  /books/{bookId}:
    get:
      operationId: getBook
      summary: Get book by ID
      tags:
        - books
      parameters:
        - name: bookId
          in: path
          required: true
          type: integer
    delete:
      operationId: deleteBook
      summary: Delete a book
      tags:
        - books
      parameters:
        - name: bookId
          in: path
          required: true
          type: integer
      security:
        - ApiKeyAuth: []
`

// Minimal OpenAPI 3.0.3 spec exercising GET/POST/PUT/DELETE, path params,
// query params, JSON body with inline schema and $ref, Bearer auth.
const oas30JSON = `{
  "openapi": "3.0.3",
  "info": {"title": "Widget API", "version": "3.0"},
  "servers": [{"url": "https://widgets.example.com/api"}],
  "components": {
    "schemas": {
      "Widget": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "id":    {"type": "integer"},
          "name":  {"type": "string"},
          "color": {"type": "string"}
        }
      }
    },
    "securitySchemes": {
      "bearerAuth": {"type": "http", "scheme": "bearer"}
    }
  },
  "paths": {
    "/widgets": {
      "get": {
        "operationId": "listWidgets",
        "summary": "List all widgets",
        "tags": ["widgets"],
        "parameters": [
          {"name": "color", "in": "query", "schema": {"type": "string"}},
          {"name": "limit", "in": "query", "required": true, "schema": {"type": "integer"}}
        ],
        "responses": {"200": {"description": "ok"}}
      },
      "post": {
        "operationId": "createWidget",
        "summary": "Create a widget",
        "tags": ["widgets"],
        "security": [{"bearerAuth": []}],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/Widget"}
            }
          }
        },
        "responses": {"201": {"description": "created"}}
      }
    },
    "/widgets/{id}": {
      "get": {
        "operationId": "getWidget",
        "summary": "Get widget by ID",
        "tags": ["widgets"],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}
        ],
        "responses": {"200": {"description": "ok"}}
      },
      "put": {
        "operationId": "updateWidget",
        "summary": "Replace a widget",
        "tags": ["widgets"],
        "security": [{"bearerAuth": []}],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/Widget"}
            }
          }
        },
        "responses": {"200": {"description": "ok"}}
      },
      "delete": {
        "operationId": "deleteWidget",
        "summary": "Delete a widget",
        "tags": ["widgets"],
        "security": [{"bearerAuth": []}],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}
        ],
        "responses": {"204": {"description": "deleted"}}
      }
    },
    "/widgets/{id}/clone": {
      "post": {
        "operationId": "cloneWidget",
        "summary": "Clone a widget",
        "tags": ["widgets"],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}
        ],
        "responses": {"200": {"description": "ok"}}
      }
    }
  }
}`

// OpenAPI 3.1.0 spec — same shape as the 3.0 spec but with the 3.1 version field.
// 3.1 made "paths" optional and uses full JSON Schema; we verify kin-openapi
// handles the version without error.
const oas31JSON = `{
  "openapi": "3.1.0",
  "info": {"title": "Gadget API", "version": "3.1"},
  "paths": {
    "/gadgets": {
      "get": {
        "operationId": "listGadgets",
        "summary": "List gadgets",
        "tags": ["gadgets"],
        "parameters": [
          {"name": "type", "in": "query", "schema": {"type": "string"}}
        ],
        "responses": {"200": {"description": "ok"}}
      },
      "post": {
        "operationId": "createGadget",
        "summary": "Create gadget",
        "tags": ["gadgets"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["model"],
                "properties": {
                  "model":  {"type": "string"},
                  "serial": {"type": "string"}
                }
              }
            }
          }
        },
        "responses": {"201": {"description": "created"}}
      }
    },
    "/gadgets/{id}": {
      "get": {
        "operationId": "getGadget",
        "summary": "Get gadget",
        "tags": ["gadgets"],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {"200": {"description": "ok"}}
      },
      "patch": {
        "operationId": "patchGadget",
        "summary": "Patch gadget",
        "tags": ["gadgets"],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "model": {"type": "string"}
                }
              }
            }
          }
        },
        "responses": {"200": {"description": "ok"}}
      },
      "delete": {
        "operationId": "deleteGadget",
        "summary": "Delete gadget",
        "tags": ["gadgets"],
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {"204": {"description": "deleted"}}
      }
    }
  }
}`

// Swagger 1.2 format — not supported; must return an error from ParseBytes.
const sw1JSON = `{
  "swaggerVersion": "1.2",
  "basePath": "http://example.com/api",
  "apis": [
    {"path": "/books", "operations": [{"httpMethod": "GET", "nickname": "listBooks"}]}
  ]
}`

// ---- IsSwagger2 detection ----------------------------------------------------

func TestIsSwagger2_AllVersions(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"swagger 2.0 JSON",        sw2JSON,                                    true},
		{"swagger 2.0 YAML",        sw2YAML,                                    true},
		{"swagger 2.0 single-quote","swagger: '2.0'\ninfo:\n  title: T\n  version: \"1\"\npaths: {}", true},
		{"openapi 3.0.3",           oas30JSON,                                  false},
		{"openapi 3.1.0",           oas31JSON,                                  false},
		{"swagger 1.x",             sw1JSON,                                    false},
		{"empty object",            "{}",                                       false},
		{"random text",             "not a spec",                               false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := openapi.IsSwagger2([]byte(tc.data))
			if got != tc.want {
				t.Errorf("IsSwagger2 = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---- Swagger 2.0 JSON --------------------------------------------------------

func TestSW2JSON_ParsesInfo(t *testing.T) {
	title, version, _, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Bookstore" {
		t.Errorf("title: got %q, want %q", title, "Bookstore")
	}
	if version != "2.1" {
		t.Errorf("version: got %q, want %q", version, "2.1")
	}
}

func TestSW2JSON_OperationCount(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// /books:GET,POST  /books/{id}:GET,PUT,DELETE  /books/{id}/cover:POST
	// /books/search:POST  /books/bulk:POST  /books/{id}/reviews:GET,POST = 10
	if len(reqs) != 10 {
		t.Errorf("operation count: got %d, want 10", len(reqs))
	}
}

func TestSW2JSON_PathParams(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "getBook" {
			if !strings.Contains(r.Request.URL, "{{bookId}}") {
				t.Errorf("URL %q missing {{bookId}} placeholder", r.Request.URL)
			}
			// Ensure no unreplaced single-brace form remains after the double-brace substitution.
			if strings.Contains(strings.ReplaceAll(r.Request.URL, "{{bookId}}", ""), "{bookId}") {
				t.Errorf("URL %q still has raw {bookId}", r.Request.URL)
			}
			return
		}
	}
	t.Error("getBook not found")
}

func TestSW2JSON_RequiredQueryParams(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "listBooks" {
			// "limit" is required; "genre" is not
			hasLimit := false
			for _, qp := range r.Request.QueryParams {
				if qp.Name == "limit" {
					hasLimit = true
				}
				if qp.Name == "genre" {
					t.Error("optional query param 'genre' should not be pre-populated")
				}
			}
			if !hasLimit {
				t.Error("required query param 'limit' missing")
			}
			return
		}
	}
	t.Error("listBooks not found")
}

func TestSW2JSON_RequiredHeaderParam(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "listBooks" {
			for _, h := range r.Request.Headers {
				if h.Name == "X-Locale" {
					return
				}
			}
			t.Error("required header X-Locale not found")
			return
		}
	}
	t.Error("listBooks not found")
}

func TestSW2JSON_RefBodyStub(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "createBook" {
			if r.Request.Body == nil || r.Request.Body.Type != domain.BodyJSON {
				t.Fatalf("expected JSON body, got %v", r.Request.Body)
			}
			for _, field := range []string{`"title"`, `"isbn"`, `"author"`, `"id"`} {
				if !strings.Contains(r.Request.Body.Content, field) {
					t.Errorf("body stub missing field %s:\n%s", field, r.Request.Body.Content)
				}
			}
			// Verify required fields come first (title, isbn before author, id)
			titleIdx := strings.Index(r.Request.Body.Content, `"title"`)
			authorIdx := strings.Index(r.Request.Body.Content, `"author"`)
			if titleIdx > authorIdx {
				t.Error("required fields should appear before optional fields")
			}
			return
		}
	}
	t.Error("createBook not found")
}

func TestSW2JSON_ArrayBodyStub(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "bulkCreateBooks" {
			if r.Request.Body == nil || r.Request.Body.Type != domain.BodyJSON {
				t.Fatalf("expected JSON body, got %v", r.Request.Body)
			}
			content := strings.TrimSpace(r.Request.Body.Content)
			if !strings.HasPrefix(content, "[") || !strings.HasSuffix(content, "]") {
				t.Errorf("expected array body, got: %s", content)
			}
			if !strings.Contains(content, `"title"`) {
				t.Errorf("array body missing Book fields: %s", content)
			}
			return
		}
	}
	t.Error("bulkCreateBooks not found")
}

func TestSW2JSON_FormURLEncoded(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "searchBooks" {
			if r.Request.Body == nil {
				t.Fatal("expected body, got nil")
			}
			if r.Request.Body.Type != domain.BodyFormURLEncoded {
				t.Errorf("body type: got %q, want form-urlencoded", r.Request.Body.Type)
			}
			if !strings.Contains(r.Request.Body.Content, "q=") {
				t.Errorf("form body missing q: %s", r.Request.Body.Content)
			}
			return
		}
	}
	t.Error("searchBooks not found")
}

func TestSW2JSON_MultipartFileUpload(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "uploadCover" {
			if r.Request.Body == nil {
				t.Fatal("expected body, got nil")
			}
			if r.Request.Body.Type != domain.BodyMultipart {
				t.Errorf("body type: got %q, want multipart", r.Request.Body.Type)
			}
			// Multipart JSON format: {"n":"image","t":"file",...}
			if !strings.Contains(r.Request.Body.Content, `"file"`) {
				t.Errorf("multipart stub missing file field: %s", r.Request.Body.Content)
			}
			return
		}
	}
	t.Error("uploadCover not found")
}

func TestSW2JSON_APIKeyAuth(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "createBook" {
			if r.Request.Auth == nil || r.Request.Auth.Type != domain.AuthAPIKey {
				t.Errorf("expected apikey auth, got %v", r.Request.Auth)
			}
			if r.Request.Auth.APIKey.Key != "X-API-Key" {
				t.Errorf("apikey name: got %q, want %q", r.Request.Auth.APIKey.Key, "X-API-Key")
			}
			return
		}
	}
	t.Error("createBook not found")
}

func TestSW2JSON_OAuthBearer(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "updateBook" {
			if r.Request.Auth == nil || r.Request.Auth.Type != domain.AuthBearer {
				t.Errorf("expected bearer auth, got %v", r.Request.Auth)
			}
			return
		}
	}
	t.Error("updateBook not found")
}

func TestSW2JSON_TagsOnRequests(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "createReview" {
			if len(r.Tags) == 0 || r.Tags[0] != "reviews" {
				t.Errorf("tags: got %v, want [reviews]", r.Tags)
			}
			return
		}
	}
	t.Error("createReview not found")
}

func TestSW2JSON_PathField(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "listReviews" {
			if r.Path != "/books/{bookId}/reviews" {
				t.Errorf("Path: got %q, want %q", r.Path, "/books/{bookId}/reviews")
			}
			return
		}
	}
	t.Error("listReviews not found")
}

func TestSW2JSON_BaseURLFromSpec(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if !strings.HasPrefix(r.Request.URL, "https://api.books.io/v1/") {
			t.Errorf("URL %q does not start with expected base", r.Request.URL)
		}
	}
}

func TestSW2JSON_BaseURLOverride(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2JSON), "json", "https://staging.books.io")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if !strings.HasPrefix(r.Request.URL, "https://staging.books.io/") {
			t.Errorf("URL %q not overridden to staging", r.Request.URL)
		}
	}
}

// ---- Swagger 2.0 YAML --------------------------------------------------------

func TestSW2YAML_DetectedAndParsed(t *testing.T) {
	if !openapi.IsSwagger2([]byte(sw2YAML)) {
		t.Fatal("YAML Swagger 2.0 not detected by IsSwagger2")
	}
	title, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2YAML), "yaml", "")
	if err != nil {
		t.Fatalf("ParseSwagger2Bytes YAML: %v", err)
	}
	if title != "Bookstore YAML" {
		t.Errorf("title: got %q", title)
	}
	if len(reqs) != 4 { // GET /books, POST /books, GET /books/{id}, DELETE /books/{id}
		t.Errorf("operation count: got %d, want 4", len(reqs))
	}
}

func TestSW2YAML_PathParamAndRefBody(t *testing.T) {
	_, _, reqs, err := openapi.ParseSwagger2Bytes([]byte(sw2YAML), "yaml", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range reqs {
		if r.OperationID == "createBook" {
			if r.Request.Body == nil || r.Request.Body.Type != domain.BodyJSON {
				t.Errorf("expected JSON body, got %v", r.Request.Body)
			}
			if !strings.Contains(r.Request.Body.Content, `"title"`) {
				t.Errorf("body stub missing title: %s", r.Request.Body.Content)
			}
			return
		}
	}
	t.Error("createBook not found in YAML parse")
}

// ---- OpenAPI 3.0.x -----------------------------------------------------------

func TestOAS30_DetectionAndInfo(t *testing.T) {
	if openapi.IsSwagger2([]byte(oas30JSON)) {
		t.Error("OAS3.0 should not be detected as Swagger 2.0")
	}
	spec, err := openapi.ParseBytes([]byte(oas30JSON), "json")
	if err != nil {
		t.Fatalf("ParseBytes OAS3.0: %v", err)
	}
	title, version := spec.Info()
	if title != "Widget API" {
		t.Errorf("title: got %q", title)
	}
	if version != "3.0" {
		t.Errorf("version: got %q", version)
	}
}

func TestOAS30_OperationCount(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(oas30JSON), "json")
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	reqs, err := openapi.GenerateRequests(spec, "")
	if err != nil {
		t.Fatalf("GenerateRequests: %v", err)
	}
	// /widgets: GET, POST  /widgets/{id}: GET, PUT, DELETE  /widgets/{id}/clone: POST = 6
	if len(reqs) != 6 {
		t.Errorf("operation count: got %d, want 6", len(reqs))
	}
}

func TestOAS30_PathParams(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas30JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if r.OperationID == "getWidget" {
			if !strings.Contains(r.Request.URL, "{{id}}") {
				t.Errorf("URL %q missing {{id}} placeholder", r.Request.URL)
			}
			return
		}
	}
	t.Error("getWidget not found")
}

func TestOAS30_RequiredQueryParam(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas30JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if r.OperationID == "listWidgets" {
			hasLimit := false
			for _, qp := range r.Request.QueryParams {
				if qp.Name == "limit" {
					hasLimit = true
				}
				if qp.Name == "color" {
					t.Error("optional query param 'color' should not be pre-populated")
				}
			}
			if !hasLimit {
				t.Error("required query param 'limit' missing")
			}
			return
		}
	}
	t.Error("listWidgets not found")
}

func TestOAS30_JSONBodyStub(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas30JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if r.OperationID == "createWidget" {
			if r.Request.Body == nil || r.Request.Body.Type != domain.BodyJSON {
				t.Fatalf("expected JSON body, got %v", r.Request.Body)
			}
			for _, field := range []string{`"name"`, `"color"`, `"id"`} {
				if !strings.Contains(r.Request.Body.Content, field) {
					t.Errorf("body stub missing field %s", field)
				}
			}
			return
		}
	}
	t.Error("createWidget not found")
}

func TestOAS30_BearerAuth(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas30JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if r.OperationID == "deleteWidget" {
			if r.Request.Auth == nil || r.Request.Auth.Type != domain.AuthBearer {
				t.Errorf("expected bearer auth, got %v", r.Request.Auth)
			}
			return
		}
	}
	t.Error("deleteWidget not found")
}

func TestOAS30_BaseURL(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas30JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if !strings.HasPrefix(r.Request.URL, "https://widgets.example.com/api/") {
			t.Errorf("URL %q does not use spec server", r.Request.URL)
		}
	}
}

func TestOAS30_BaseURLOverride(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas30JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "https://dev.example.com")
	for _, r := range reqs {
		if !strings.HasPrefix(r.Request.URL, "https://dev.example.com/") {
			t.Errorf("URL %q not overridden", r.Request.URL)
		}
	}
}

func TestOAS30_PathField(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas30JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if r.OperationID == "updateWidget" {
			if r.Path != "/widgets/{id}" {
				t.Errorf("Path: got %q, want %q", r.Path, "/widgets/{id}")
			}
			return
		}
	}
	t.Error("updateWidget not found")
}

// ---- OpenAPI 3.1.x -----------------------------------------------------------

func TestOAS31_DetectionAndInfo(t *testing.T) {
	if openapi.IsSwagger2([]byte(oas31JSON)) {
		t.Error("OAS3.1 should not be detected as Swagger 2.0")
	}
	spec, err := openapi.ParseBytes([]byte(oas31JSON), "json")
	if err != nil {
		t.Fatalf("ParseBytes OAS3.1: %v", err)
	}
	title, version := spec.Info()
	if title != "Gadget API" {
		t.Errorf("title: got %q", title)
	}
	if version != "3.1" {
		t.Errorf("version: got %q", version)
	}
}

func TestOAS31_AllMethods(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(oas31JSON), "json")
	if err != nil {
		t.Fatalf("ParseBytes OAS3.1: %v", err)
	}
	reqs, err := openapi.GenerateRequests(spec, "")
	if err != nil {
		t.Fatalf("GenerateRequests: %v", err)
	}
	// /gadgets: GET, POST  /gadgets/{id}: GET, PATCH, DELETE = 5
	if len(reqs) != 5 {
		t.Errorf("operation count: got %d, want 5", len(reqs))
	}
	methods := make(map[string]bool)
	for _, r := range reqs {
		methods[string(r.Request.Method)] = true
	}
	for _, m := range []string{"GET", "POST", "PATCH", "DELETE"} {
		if !methods[m] {
			t.Errorf("method %s not found in generated requests", m)
		}
	}
}

func TestOAS31_InlineBodyStub(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas31JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if r.OperationID == "createGadget" {
			if r.Request.Body == nil || r.Request.Body.Type != domain.BodyJSON {
				t.Fatalf("expected JSON body, got %v", r.Request.Body)
			}
			if !strings.Contains(r.Request.Body.Content, `"model"`) {
				t.Errorf("body stub missing model field: %s", r.Request.Body.Content)
			}
			return
		}
	}
	t.Error("createGadget not found")
}

func TestOAS31_PathParam(t *testing.T) {
	spec, _ := openapi.ParseBytes([]byte(oas31JSON), "json")
	reqs, _ := openapi.GenerateRequests(spec, "")
	for _, r := range reqs {
		if r.OperationID == "getGadget" {
			if !strings.Contains(r.Request.URL, "{{id}}") {
				t.Errorf("URL %q missing {{id}} placeholder", r.Request.URL)
			}
			return
		}
	}
	t.Error("getGadget not found")
}

// ---- Swagger 1.x and unknown versions ----------------------------------------

func TestSwagger1x_ReturnsError(t *testing.T) {
	if openapi.IsSwagger2([]byte(sw1JSON)) {
		t.Error("Swagger 1.x should not be detected as Swagger 2.0")
	}
	_, err := openapi.ParseBytes([]byte(sw1JSON), "json")
	if err == nil {
		t.Error("expected error for Swagger 1.x, got nil")
	}
}

func TestUnknownVersion_KinOpenAPIPermissive(t *testing.T) {
	// kin-openapi does not reject unknown openapi: version strings.
	// This test documents the current behaviour so any future change is explicit.
	unknown := `{"openapi":"99.0.0","info":{"title":"T","version":"1"},"paths":{}}`
	if openapi.IsSwagger2([]byte(unknown)) {
		t.Error("unknown version should not match IsSwagger2")
	}
	// ParseBytes currently succeeds for unknown versions (kin-openapi is permissive).
	spec, err := openapi.ParseBytes([]byte(unknown), "json")
	_ = spec
	t.Logf("unknown version parse: err=%v", err)
	// We do not assert pass/fail here — just ensure it doesn't panic.
}

func TestGarbage_ReturnsError(t *testing.T) {
	for _, bad := range []string{"not json", "{{{", "", "   "} {
		_, err := openapi.ParseBytes([]byte(bad), "json")
		if err == nil {
			t.Errorf("expected error for garbage input %q, got nil", bad)
		}
	}
}
