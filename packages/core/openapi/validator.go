package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/fuvito/api-workbench/packages/core/domain"
)

// ValidationError is a single schema violation.
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Path != "" {
		return e.Path + ": " + e.Message
	}
	return e.Message
}

// ValidateResponse checks that resp conforms to the declared response schema
// for the given operationID and HTTP status code.
// Returns nil if valid, a list of violations otherwise.
func ValidateResponse(s *Spec, operationID string, resp *domain.Response) []ValidationError {
	op := findOperation(s.doc, operationID)
	if op == nil {
		return []ValidationError{{Message: fmt.Sprintf("operation %q not found in spec", operationID)}}
	}

	statusStr := strconv.Itoa(resp.StatusCode)
	respRef := op.Responses.Value(statusStr)
	if respRef == nil {
		respRef = op.Responses.Value("default")
	}
	if respRef == nil || respRef.Value == nil {
		return nil // no schema declared — nothing to validate
	}

	body := resp.BodyString()
	if body == "" {
		return nil
	}

	ct := contentType(resp)
	mediaType := respRef.Value.Content.Get(ct)
	if mediaType == nil {
		mediaType = respRef.Value.Content.Get("application/json")
	}
	if mediaType == nil || mediaType.Schema == nil {
		return nil
	}

	schema := mediaType.Schema.Value
	if schema == nil {
		return nil
	}

	var data interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return []ValidationError{{Message: "response body is not valid JSON: " + err.Error()}}
	}

	if err := schema.VisitJSON(data); err != nil {
		return parseSchemaErrors(err)
	}
	return nil
}

// ValidateResponseByPath validates a response against the spec using the HTTP method and path
// directly, without needing an operationID.
func ValidateResponseByPath(s *Spec, method, path string, statusCode int, resp *domain.Response) []ValidationError {
	router, err := gorillamux.NewRouter(s.doc)
	if err != nil {
		return []ValidationError{{Message: "building router: " + err.Error()}}
	}

	httpReq, err := http.NewRequest(method, path, nil)
	if err != nil {
		return []ValidationError{{Message: "building request: " + err.Error()}}
	}

	route, _, err := router.FindRoute(httpReq)
	if err != nil {
		return nil // path not in spec — skip
	}

	body := []byte(resp.BodyString())
	ct := contentType(resp)

	httpResp := &http.Response{
		StatusCode: resp.StatusCode,
		Header:     make(http.Header),
		Body:       nopCloser(body),
	}
	httpResp.Header.Set("Content-Type", ct)

	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: httpReq,
			Route:   route,
		},
		Status: statusCode,
		Header: httpResp.Header,
		Body:   nopCloser(body),
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true,
		},
	}

	if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
		if me, ok := err.(*openapi3filter.ResponseError); ok {
			return parseSchemaErrors(me.Err)
		}
		return []ValidationError{{Message: err.Error()}}
	}
	return nil
}

// ValidateRequest checks that req conforms to the spec's declared request schema
// for the matching operation. Returns nil if valid or if the path is not in the spec.
func ValidateRequest(s *Spec, req *domain.Request) []ValidationError {
	router, err := gorillamux.NewRouter(s.doc)
	if err != nil {
		return []ValidationError{{Message: "building router: " + err.Error()}}
	}

	rawURL := req.URL
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), string(req.Method), rawURL, nil)
	if err != nil {
		return []ValidationError{{Message: "building request: " + err.Error()}}
	}

	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for _, p := range req.QueryParams {
			if p.Name != "" {
				q.Set(p.Name, p.Value)
			}
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	for _, h := range req.Headers {
		if h.Name != "" {
			httpReq.Header.Set(h.Name, h.Value)
		}
	}

	if req.Body != nil && req.Body.Content != "" {
		ct := bodyContentType(req.Body.Type)
		httpReq.Header.Set("Content-Type", ct)
		b := []byte(req.Body.Content)
		httpReq.Body = io.NopCloser(bytes.NewReader(b))
		httpReq.ContentLength = int64(len(b))
	}

	route, pathParams, err := router.FindRoute(httpReq)
	if err != nil {
		return nil // path not in spec — skip
	}

	input := &openapi3filter.RequestValidationInput{
		Request:    httpReq,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			// Skip security-scheme checks — the conformance runner handles auth
			// via env-variable interpolation, not spec-level security validators.
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}

	if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
		if re, ok := err.(*openapi3filter.RequestError); ok {
			return parseSchemaErrors(re.Err)
		}
		return []ValidationError{{Message: err.Error()}}
	}
	return nil
}

func bodyContentType(t domain.BodyType) string {
	switch t {
	case domain.BodyJSON:
		return "application/json"
	case domain.BodyFormURLEncoded:
		return "application/x-www-form-urlencoded"
	case domain.BodyMultipart:
		return "multipart/form-data"
	default:
		return "text/plain"
	}
}

func findOperation(doc *openapi3.T, operationID string) *openapi3.Operation {
	for _, item := range doc.Paths.Map() {
		for _, op := range []*openapi3.Operation{
			item.Get, item.Post, item.Put, item.Patch,
			item.Delete, item.Head, item.Options,
		} {
			if op != nil && op.OperationID == operationID {
				return op
			}
		}
	}
	return nil
}

func contentType(resp *domain.Response) string {
	for _, h := range resp.Headers {
		if strings.EqualFold(h.Name, "content-type") {
			return strings.SplitN(h.Value, ";", 2)[0]
		}
	}
	return "application/json"
}

func parseSchemaErrors(err error) []ValidationError {
	if err == nil {
		return nil
	}
	var errs []ValidationError
	// openapi3 returns a SchemaError or a list of them
	type schemaErr interface {
		Unwrap() []error
	}
	if multi, ok := err.(schemaErr); ok {
		for _, e := range multi.Unwrap() {
			errs = append(errs, ValidationError{Message: e.Error()})
		}
	} else {
		errs = append(errs, ValidationError{Message: err.Error()})
	}
	return errs
}

type nopCloserReader struct{ *strings.Reader }

func (nopCloserReader) Close() error { return nil }

func nopCloser(data []byte) *nopCloserReader {
	return &nopCloserReader{strings.NewReader(string(data))}
}

// ExecutedRequest is a (method, full URL) pair from an executed scenario step.
type ExecutedRequest struct {
	Method string
	URL    string
}

// CoveredOperation identifies a spec operation (method + path pattern) exercised or missed.
type CoveredOperation struct {
	Method string
	Path   string
}

func (o CoveredOperation) String() string { return o.Method + " " + o.Path }

// ScenarioCoverageResult is the coverage computed from matching executed requests
// against an OpenAPI spec's declared operations.
type ScenarioCoverageResult struct {
	Covered   int
	Total     int
	Percent   float64
	Exercised []CoveredOperation
	Missed    []CoveredOperation
}

// ScenarioCoverage matches each executed request against the spec's declared operations
// and returns a coverage summary.
func (s *Spec) ScenarioCoverage(executed []ExecutedRequest) *ScenarioCoverageResult {
	router, err := gorillamux.NewRouter(s.doc)
	if err != nil {
		return nil
	}

	// Build the full set of spec operations as a set.
	type opKey struct{ method, path string }
	all := make(map[opKey]struct{})
	for path, item := range s.doc.Paths.Map() {
		for method := range item.Operations() {
			all[opKey{strings.ToUpper(method), path}] = struct{}{}
		}
	}

	// Use the spec's first server URL as the base for path matching.
	// This normalises executed URLs so FindRoute succeeds regardless of
	// which actual host the requests were sent to (e.g. an httptest server
	// on a random port vs. the host declared in the spec's servers list).
	specBase := ""
	if len(s.doc.Servers) > 0 {
		specBase = strings.TrimRight(s.doc.Servers[0].URL, "/")
	}

	exercised := make(map[opKey]struct{})
	for _, ex := range executed {
		matchURL := ex.URL
		if specBase != "" {
			if u, err := url.Parse(ex.URL); err == nil && u.Path != "" {
				matchURL = specBase + u.Path
			}
		}
		req, err := http.NewRequest(strings.ToUpper(ex.Method), matchURL, nil)
		if err != nil {
			continue
		}
		route, _, err := router.FindRoute(req)
		if err != nil {
			continue
		}
		exercised[opKey{strings.ToUpper(ex.Method), route.Path}] = struct{}{}
	}

	cr := &ScenarioCoverageResult{Total: len(all)}
	for k := range exercised {
		if _, ok := all[k]; ok {
			cr.Covered++
			cr.Exercised = append(cr.Exercised, CoveredOperation{Method: k.method, Path: k.path})
		}
	}
	for k := range all {
		if _, ok := exercised[k]; !ok {
			cr.Missed = append(cr.Missed, CoveredOperation{Method: k.method, Path: k.path})
		}
	}
	if cr.Total > 0 {
		cr.Percent = float64(cr.Covered) / float64(cr.Total) * 100
	}
	return cr
}
