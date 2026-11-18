package scenario_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// flaky answers 503 for the first failures requests, then 200.
func flaky(t *testing.T, failures int64) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) <= failures {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func retryStep(url string, mutate func(*domain.ScenarioStep)) *domain.Scenario {
	st := domain.ScenarioStep{Label: "call", Request: &domain.Request{Method: domain.MethodGET, URL: url},
		Assertions: []domain.Assertion{{Type: domain.AssertionStatus, Operator: domain.OperatorEq, Expected: "200"}}}
	mutate(&st)
	return &domain.Scenario{Steps: []domain.ScenarioStep{st}}
}

func runRetry(sc *domain.Scenario) *scenario.Result {
	return scenario.New(executor.New(executor.Options{})).Run(context.Background(), sc, nil)
}

func TestRetryResendsAfterA5xxUntilItSucceeds(t *testing.T) {
	srv, n := flaky(t, 2)
	res := runRetry(retryStep(srv.URL, func(s *domain.ScenarioStep) {
		s.Retry = &domain.RetryPolicy{MaxAttempts: 3, Delay: time.Millisecond}
	}))
	if !res.Passed || res.Steps[0].Attempts != 3 || n.Load() != 3 {
		t.Errorf("passed=%v attempts=%d requests=%d", res.Passed, res.Steps[0].Attempts, n.Load())
	}
}

func TestRetryStopsAtMaxAttemptsAndReportsTheLastResponse(t *testing.T) {
	srv, n := flaky(t, 100)
	res := runRetry(retryStep(srv.URL, func(s *domain.ScenarioStep) { s.Retry = &domain.RetryPolicy{MaxAttempts: 3} }))
	if res.Passed || res.Steps[0].Attempts != 3 || n.Load() != 3 || res.Steps[0].Response.StatusCode != 503 {
		t.Errorf("passed=%v attempts=%d requests=%d", res.Passed, res.Steps[0].Attempts, n.Load())
	}
}

func TestRetryDoesNotResendAFailedAssertion(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n.Add(1); w.WriteHeader(404) }))
	defer srv.Close()
	res := runRetry(retryStep(srv.URL, func(s *domain.ScenarioStep) { s.Retry = &domain.RetryPolicy{MaxAttempts: 5} }))
	if res.Passed || n.Load() != 1 || res.Steps[0].Attempts != 1 {
		t.Errorf("a 4xx or a wrong value is a real result, not a flake: requests=%d", n.Load())
	}
}

func TestRetryResendsAfterATransportError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens any more
	res := runRetry(retryStep(url, func(s *domain.ScenarioStep) { s.Retry = &domain.RetryPolicy{MaxAttempts: 2, Delay: time.Millisecond} }))
	if res.Passed || res.Steps[0].Attempts != 2 || res.Steps[0].Error == "" {
		t.Errorf("%+v", res.Steps[0])
	}
}

func TestNoRetryPolicyMeansOneAttempt(t *testing.T) {
	srv, n := flaky(t, 100)
	res := runRetry(retryStep(srv.URL, func(*domain.ScenarioStep) {}))
	if res.Steps[0].Attempts != 1 || n.Load() != 1 {
		t.Errorf("attempts=%d requests=%d", res.Steps[0].Attempts, n.Load())
	}
}

func TestRetryDelayIsRespectedAndCancelled(t *testing.T) {
	srv, _ := flaky(t, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	scenario.New(executor.New(executor.Options{})).Run(ctx, retryStep(srv.URL, func(s *domain.ScenarioStep) {
		s.Retry = &domain.RetryPolicy{MaxAttempts: 10, Delay: 10 * time.Second}
	}), nil)
	if time.Since(start) > 2*time.Second {
		t.Error("cancelling must interrupt the wait between attempts")
	}
}

func TestStepTimeoutOverridesTheRequestTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer slow.Close()
	res := runRetry(retryStep(slow.URL, func(s *domain.ScenarioStep) {
		s.TimeoutMs = 50
		s.Request.Timeout = time.Minute
	}))
	if res.Passed || !strings.Contains(strings.ToLower(res.Steps[0].Error), "deadline") && !strings.Contains(strings.ToLower(res.Steps[0].Error), "timeout") {
		t.Errorf("a 50 ms step timeout must win over the request's 1 minute: %q", res.Steps[0].Error)
	}
}

func TestRetryAndTimeoutLimitsAreValidated(t *testing.T) {
	sc := retryStep("http://x", func(s *domain.ScenarioStep) {
		s.TimeoutMs = -1
		s.Retry = &domain.RetryPolicy{MaxAttempts: 11, Delay: 2 * time.Minute}
	})
	var msgs []string
	for _, i := range scenario.Validate(sc, "") {
		msgs = append(msgs, i.Message)
	}
	all := strings.Join(msgs, "|")
	for _, want := range []string{"timeoutMs cannot be negative", "maxAttempts is 11", "delayMs is 120000"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing %q in %v", want, msgs)
		}
	}
}
