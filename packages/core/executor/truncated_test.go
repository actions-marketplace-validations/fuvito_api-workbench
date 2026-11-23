package executor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
)

func TestExecute_FlagsABodyCutAtTheReadLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", 100)))
	}))
	defer srv.Close()
	exec := executor.New(executor.Options{MaxBodySize: 10})
	resp, err := exec.Execute(context.Background(), &domain.Request{Method: "GET", URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Truncated || len(resp.Body) != 10 || resp.ContentLength != 100 {
		t.Errorf("truncated=%v len=%d contentLength=%d", resp.Truncated, len(resp.Body), resp.ContentLength)
	}
}
