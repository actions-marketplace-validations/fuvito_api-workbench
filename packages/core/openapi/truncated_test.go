package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// F-019: a body cut at the read limit is said to be cut, not reported as invalid JSON.
func TestValidateResponse_BodyCutAtTheReadLimit(t *testing.T) {
	spec, err := openapi.ParseBytes([]byte(serversSpec), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	cut := &domain.Response{
		StatusCode: 200, Truncated: true, ContentLength: 51 << 20,
		Headers: []domain.Header{{Name: "Content-Type", Value: "application/json"}},
		Body:    []byte(`{"name": "abc`),
	}
	errs := openapi.ValidateResponseByPath(spec, "GET", "/cases/x", 200, cut)
	if len(errs) != 1 || errs[0].Kind != openapi.KindBodyTooLarge || !strings.Contains(errs[0].Message, "cut off") || !strings.Contains(errs[0].Message, "51 MB") {
		t.Errorf("got %+v", errs)
	}
	cut.Headers = []domain.Header{{Name: "Content-Type", Value: "application/octet-stream"}}
	if errs := openapi.ValidateResponseByPath(spec, "GET", "/cases/x", 200, cut); len(errs) == 1 && errs[0].Kind == openapi.KindBodyTooLarge {
		t.Error("a binary download cut at the limit is not compared anyway")
	}
}
