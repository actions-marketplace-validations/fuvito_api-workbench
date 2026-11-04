package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
