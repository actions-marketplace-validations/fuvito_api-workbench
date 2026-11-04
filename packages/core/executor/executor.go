package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

const (
	defaultTimeout     = 30 * time.Second
	defaultMaxBodySize = 50 * 1024 * 1024 // 50 MB
)

// Executor executes domain.Requests using net/http.
// It is the only place in the codebase that performs HTTP I/O.
// Neither the desktop frontend nor the CLI may duplicate this logic.
type Executor struct {
	client      *http.Client
	defaultTimeout time.Duration
	maxBodySize    int64
}

type Options struct {
	DefaultTimeout time.Duration
	MaxBodySize    int64
}

func New(opts Options) *Executor {
	timeout := opts.DefaultTimeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	maxBody := opts.MaxBodySize
	if maxBody == 0 {
		maxBody = defaultMaxBodySize
	}
	return &Executor{
		client:         &http.Client{},
		defaultTimeout: timeout,
		maxBodySize:    maxBody,
	}
}

// Execute sends the request and returns the response.
// The caller is responsible for variable interpolation before calling Execute.
// ctx can be used to cancel the request externally; req.Timeout further constrains it.
func (e *Executor) Execute(ctx context.Context, req *domain.Request) (*domain.Response, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	timeout := e.defaultTimeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := buildHTTPRequest(timeoutCtx, req)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	start := time.Now()
	httpResp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer httpResp.Body.Close()

	return buildResponse(httpResp, time.Since(start), e.maxBodySize)
}

func buildHTTPRequest(ctx context.Context, req *domain.Request) (*http.Request, error) {
	var bodyReader io.Reader
	if req.Body != nil && req.Body.Content != "" {
		bodyReader = bytes.NewBufferString(req.Body.Content)
	}

	httpReq, err := http.NewRequestWithContext(ctx, string(req.Method), req.URL, bodyReader)
	if err != nil {
		return nil, err
	}

	q := httpReq.URL.Query()
	for _, p := range req.QueryParams {
		q.Add(p.Name, p.Value)
	}
	httpReq.URL.RawQuery = q.Encode()

	for _, h := range req.Headers {
		httpReq.Header.Set(h.Name, h.Value)
	}

	if req.Body != nil && req.Body.Type != domain.BodyNone && httpReq.Header.Get("Content-Type") == "" {
		if ct := contentTypeFor(req.Body.Type); ct != "" {
			httpReq.Header.Set("Content-Type", ct)
		}
	}

	if req.Auth != nil {
		applyAuth(httpReq, req.Auth)
	}

	return httpReq, nil
}

func buildResponse(resp *http.Response, duration time.Duration, maxBodySize int64) (*domain.Response, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	headers := make([]domain.Header, 0, len(resp.Header))
	for name, vals := range resp.Header {
		for _, v := range vals {
			headers = append(headers, domain.Header{Name: name, Value: v})
		}
	}

	return &domain.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       body,
		Duration:   duration,
		Size:       int64(len(body)),
	}, nil
}

func contentTypeFor(t domain.BodyType) string {
	switch t {
	case domain.BodyJSON:
		return "application/json"
	case domain.BodyFormURLEncoded:
		return "application/x-www-form-urlencoded"
	default:
		return ""
	}
}

func applyAuth(req *http.Request, auth *domain.Auth) {
	switch auth.Type {
	case domain.AuthBearer:
		if auth.Bearer != nil && auth.Bearer.Token != "" {
			req.Header.Set("Authorization", "Bearer "+auth.Bearer.Token)
		}
	case domain.AuthBasic:
		if auth.Basic != nil {
			req.SetBasicAuth(auth.Basic.Username, auth.Basic.Password)
		}
	case domain.AuthAPIKey:
		if auth.APIKey != nil && auth.APIKey.Key != "" {
			switch auth.APIKey.Location {
			case domain.APIKeyInHeader:
				req.Header.Set(auth.APIKey.Key, auth.APIKey.Value)
			case domain.APIKeyInQuery:
				q := req.URL.Query()
				q.Set(auth.APIKey.Key, auth.APIKey.Value)
				req.URL.RawQuery = q.Encode()
			}
		}
	}
}
