package openapi_test

import (
	"fmt"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

func TestProbeVersions(t *testing.T) {
	specs := map[string]string{
		"OAS 3.0.3":  `{"openapi":"3.0.3","info":{"title":"T","version":"1"},"paths":{"/ping":{"get":{"operationId":"ping","responses":{"200":{"description":"ok"}}}}}}`,
		"OAS 3.1.0":  `{"openapi":"3.1.0","info":{"title":"T","version":"1"},"paths":{"/ping":{"get":{"operationId":"ping","responses":{"200":{"description":"ok"}}}}}}`,
		"Swagger1.x": `{"swaggerVersion":"1.2","basePath":"http://example.com","apis":[]}`,
		"Unknown":    `{"openapi":"99.0.0","info":{"title":"T","version":"1"},"paths":{}}`,
	}
	for name, data := range specs {
		is2 := openapi.IsSwagger2([]byte(data))
		spec, err := openapi.ParseBytes([]byte(data), "json")
		fmt.Printf("%-12s IsSwagger2=%-5v parseErr=%-5v spec=%v\n", name, is2, err != nil, spec != nil)
	}
}
