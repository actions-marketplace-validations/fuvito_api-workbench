package executor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"sync"
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
	ProxyURL       string
	TLSSkipVerify  bool
	// WithCookies attaches an in-memory cookie jar so Set-Cookie headers are
	// automatically replayed on subsequent requests. Use for scenario runs;
	// leave false for stateless single-request execution.
	WithCookies bool
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

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.TLSSkipVerify}, //nolint:gosec
	}
	if opts.ProxyURL != "" {
		if u, err := url.Parse(opts.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}

	client := &http.Client{Transport: transport}
	if opts.WithCookies {
		jar, _ := cookiejar.New(nil)
		client.Jar = jar
	}

	return &Executor{
		client:         client,
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

	tc := &timingCollector{}
	tracedCtx := httptrace.WithClientTrace(httpReq.Context(), tc.trace())
	httpReq = httpReq.WithContext(tracedCtx)

	start := time.Now()
	httpResp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer httpResp.Body.Close()

	transferStart := time.Now()
	resp, err := buildResponse(httpResp, time.Since(start), e.maxBodySize)
	if err != nil {
		return nil, err
	}
	timing := tc.build(start)
	timing.TransferMs = ms(time.Since(transferStart))
	resp.Timing = timing
	return resp, nil
}

// timingCollector records httptrace timestamps and converts them to domain.Timing.
type timingCollector struct {
	mu              sync.Mutex
	dnsStart        time.Time
	dnsDone         time.Time
	connectStart    time.Time
	connectDone     time.Time
	tlsStart        time.Time
	tlsDone         time.Time
	wroteRequest    time.Time
	firstByte       time.Time
}

func (tc *timingCollector) trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { tc.mu.Lock(); tc.dnsStart = time.Now(); tc.mu.Unlock() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { tc.mu.Lock(); tc.dnsDone = time.Now(); tc.mu.Unlock() },
		ConnectStart:         func(_, _ string) { tc.mu.Lock(); tc.connectStart = time.Now(); tc.mu.Unlock() },
		ConnectDone:          func(_, _ string, _ error) { tc.mu.Lock(); tc.connectDone = time.Now(); tc.mu.Unlock() },
		TLSHandshakeStart:    func() { tc.mu.Lock(); tc.tlsStart = time.Now(); tc.mu.Unlock() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { tc.mu.Lock(); tc.tlsDone = time.Now(); tc.mu.Unlock() },
		WroteRequest:         func(_ httptrace.WroteRequestInfo) { tc.mu.Lock(); tc.wroteRequest = time.Now(); tc.mu.Unlock() },
		GotFirstResponseByte: func() { tc.mu.Lock(); tc.firstByte = time.Now(); tc.mu.Unlock() },
	}
}

func ms(d time.Duration) int64 {
	if d < 0 {
		return 0
	}
	return d.Milliseconds()
}

func (tc *timingCollector) build(start time.Time) domain.Timing {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	t := domain.Timing{}
	if !tc.dnsStart.IsZero() && !tc.dnsDone.IsZero() {
		t.DNSMs = ms(tc.dnsDone.Sub(tc.dnsStart))
	}
	if !tc.connectStart.IsZero() && !tc.connectDone.IsZero() {
		t.ConnectMs = ms(tc.connectDone.Sub(tc.connectStart))
	}
	if !tc.tlsStart.IsZero() && !tc.tlsDone.IsZero() {
		t.TLSMs = ms(tc.tlsDone.Sub(tc.tlsStart))
	}
	ref := tc.wroteRequest
	if ref.IsZero() {
		ref = start
	}
	if !tc.firstByte.IsZero() {
		t.TTFBMs = ms(tc.firstByte.Sub(ref))
	}
	return t
}

func buildHTTPRequest(ctx context.Context, req *domain.Request) (*http.Request, error) {
	var bodyReader io.Reader
	var multipartContentType string

	if req.Body != nil && req.Body.Type == domain.BodyMultipart {
		r, ct, err := buildMultipartBody(req.Body.Content)
		if err != nil {
			return nil, fmt.Errorf("building multipart body: %w", err)
		}
		bodyReader = r
		multipartContentType = ct
	} else if req.Body != nil && req.Body.Content != "" {
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

	if multipartContentType != "" {
		httpReq.Header.Set("Content-Type", multipartContentType)
	} else if req.Body != nil && req.Body.Type != domain.BodyNone && httpReq.Header.Get("Content-Type") == "" {
		if ct := contentTypeFor(req.Body.Type); ct != "" {
			httpReq.Header.Set("Content-Type", ct)
		}
	}

	if req.Auth != nil {
		applyAuth(httpReq, req.Auth)
	}

	return httpReq, nil
}

// multipartField is the JSON encoding used when multipart has file fields.
// Content that starts with '[' is parsed as []multipartField; otherwise the
// legacy url-encoded format (text fields only) is used for backward compat.
type multipartField struct {
	N string `json:"n"` // field name
	T string `json:"t"` // "text" or "file"
	V string `json:"v"` // text value or absolute file path
}

func buildMultipartBody(content string) (io.Reader, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if content != "" {
		if len(content) > 0 && content[0] == '[' {
			// New JSON encoding: supports both text and file fields.
			var fields []multipartField
			if err := json.Unmarshal([]byte(content), &fields); err != nil {
				return nil, "", fmt.Errorf("parsing multipart fields: %w", err)
			}
			for _, f := range fields {
				if f.T == "file" {
					if err := addFileField(mw, f.N, f.V); err != nil {
						return nil, "", err
					}
				} else {
					if err := mw.WriteField(f.N, f.V); err != nil {
						return nil, "", err
					}
				}
			}
		} else {
			// Legacy url-encoded format (text fields only).
			values, err := url.ParseQuery(content)
			if err != nil {
				return nil, "", err
			}
			for name, vals := range values {
				for _, v := range vals {
					if err := mw.WriteField(name, v); err != nil {
						return nil, "", err
					}
				}
			}
		}
	}

	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return &buf, mw.FormDataContentType(), nil
}

func addFileField(mw *multipart.Writer, fieldName, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file %q for multipart field %q: %w", filePath, fieldName, err)
	}
	filename := filepath.Base(filePath)
	ct := mime.TypeByExtension(filepath.Ext(filePath))
	if ct == "" {
		ct = "application/octet-stream"
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, fieldName, filename))
	h.Set("Content-Type", ct)
	part, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
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
