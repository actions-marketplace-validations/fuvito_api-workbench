package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/history"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// ---- Wails-serializable I/O types ----

type HeaderIO struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type QueryParamIO struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type BodyIO struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type AuthIO struct {
	Type     string `json:"type"`
	Token    string `json:"token"`    // bearer
	Username string `json:"username"` // basic
	Password string `json:"password"` // basic
	Key      string `json:"key"`      // apikey
	Value    string `json:"value"`    // apikey
	Location string `json:"location"` // apikey: header|query
}

type RequestInput struct {
	Method      string         `json:"method"`
	URL         string         `json:"url"`
	Headers     []HeaderIO     `json:"headers"`
	QueryParams []QueryParamIO `json:"queryParams"`
	Body        *BodyIO        `json:"body"`
	Auth        *AuthIO        `json:"auth"`
	TimeoutMs   int64          `json:"timeoutMs"` // 0 means use executor default
}

type TimingIO struct {
	DNSMs      int64 `json:"dnsMs"`
	ConnectMs  int64 `json:"connectMs"`
	TLSMs      int64 `json:"tlsMs"`
	TTFBMs     int64 `json:"ttfbMs"`
	TransferMs int64 `json:"transferMs"`
}

type ResponseOutput struct {
	StatusCode int        `json:"statusCode"`
	Status     string     `json:"status"`
	Headers    []HeaderIO `json:"headers"`
	Body       string     `json:"body"`
	DurationMs int64      `json:"durationMs"`
	Size       int64      `json:"size"`
	Timing     TimingIO   `json:"timing"`
	Error      string     `json:"error,omitempty"`
}

type HistoryEntryIO struct {
	ID          int64      `json:"id"`
	ExecutedAt  string     `json:"executedAt"`
	Method      string     `json:"method"`
	URL         string     `json:"url"`
	StatusCode  int        `json:"statusCode"`
	StatusText  string     `json:"statusText"`
	DurationMs  int64      `json:"durationMs"`
	SizeBytes   int64      `json:"sizeBytes"`
	Headers     []HeaderIO `json:"headers"`
	BodyType    string     `json:"bodyType"`
	BodyContent string     `json:"bodyContent"`
}

// ---- Service ----

type RequestService struct {
	ctx        context.Context
	exec       *executor.Executor
	store      *history.Store
	workspace  *WorkspaceService
	settings   *SettingsService
	mu         sync.Mutex
	cancelLast context.CancelFunc
	execGen    uint64
}

func NewRequestService(ws *WorkspaceService, ss *SettingsService) *RequestService {
	store, err := openHistory()
	if err != nil {
		log.Printf("history store unavailable: %v", err)
	}
	return &RequestService{
		ctx:       context.Background(),
		exec:      buildExecutor(ss),
		store:     store,
		workspace: ws,
		settings:  ss,
	}
}

func buildExecutor(ss *SettingsService) *executor.Executor {
	return buildExecutorOpts(ss, false)
}

func buildExecutorWithCookies(ss *SettingsService) *executor.Executor {
	return buildExecutorOpts(ss, true)
}

func buildExecutorOpts(ss *SettingsService, withCookies bool) *executor.Executor {
	if ss == nil {
		return executor.New(executor.Options{WithCookies: withCookies})
	}
	cfg := ss.GetSettings()
	opts := executor.Options{
		TLSSkipVerify: cfg.TLSSkipVerify,
		ProxyURL:      cfg.ProxyURL,
		WithCookies:   withCookies,
	}
	if cfg.DefaultTimeoutMs > 0 {
		opts.DefaultTimeout = time.Duration(cfg.DefaultTimeoutMs) * time.Millisecond
	}
	return executor.New(opts)
}

func openHistory() (*history.Store, error) {
	path, err := history.DefaultPath()
	if err != nil {
		return nil, err
	}
	return history.Open(path)
}

func (s *RequestService) setContext(ctx context.Context) {
	s.ctx = ctx
}

// RebuildExecutor recreates the HTTP client with fresh settings (called after SaveSettings).
func (s *RequestService) RebuildExecutor() {
	s.exec = buildExecutor(s.settings)
}

func (s *RequestService) close() {
	if s.store != nil {
		s.store.Close()
	}
}

// Execute is the Wails-bound method called by the frontend.
// Network errors and validation failures are returned in ResponseOutput.Error
// rather than as Go errors, so the frontend always receives a consistent type.
func (s *RequestService) Execute(input RequestInput) ResponseOutput {
	req, err := inputToDomain(input)
	if err != nil {
		return ResponseOutput{Error: err.Error()}
	}

	var env *domain.Environment
	if s.workspace != nil {
		env = s.workspace.ActiveEnvironment()
	}
	resolver := variables.NewResolver(env)
	resolved := resolver.InterpolateRequest(req)

	// Per-request cancellable context so CancelRequest() can abort in-flight calls.
	ctx, cancel := context.WithCancel(s.ctx)
	s.mu.Lock()
	if s.cancelLast != nil {
		s.cancelLast()
	}
	s.cancelLast = cancel
	s.execGen++
	gen := s.execGen
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		if s.execGen == gen {
			s.cancelLast = nil
		}
		s.mu.Unlock()
	}()

	resp, err := s.exec.Execute(ctx, resolved)
	if err != nil {
		return ResponseOutput{Error: err.Error()}
	}

	out := domainToOutput(resp)
	s.recordHistory(input, out)
	return out
}

// GetHistory returns the most recent history entries (up to 100).
func (s *RequestService) GetHistory() []HistoryEntryIO {
	if s.store == nil {
		return nil
	}
	wsPath := ""
	if s.workspace != nil {
		wsPath = s.workspace.GetWorkspacePath()
	}
	entries, err := s.store.Recent(wsPath, 100)
	if err != nil {
		log.Printf("reading history: %v", err)
		return nil
	}
	out := make([]HistoryEntryIO, len(entries))
	for i, e := range entries {
		headers := make([]HeaderIO, len(e.Headers))
		for j, h := range e.Headers {
			headers[j] = HeaderIO{Name: h.Name, Value: h.Value}
		}
		out[i] = HistoryEntryIO{
			ID:          e.ID,
			ExecutedAt:  e.ExecutedAt.Format(time.RFC3339),
			Method:      e.Method,
			URL:         e.URL,
			StatusCode:  e.StatusCode,
			StatusText:  e.StatusText,
			DurationMs:  e.DurationMs,
			SizeBytes:   e.SizeBytes,
			Headers:     headers,
			BodyType:    e.BodyType,
			BodyContent: e.BodyContent,
		}
	}
	return out
}

// CancelRequest aborts the currently in-flight Execute call, if any.
func (s *RequestService) CancelRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelLast != nil {
		s.cancelLast()
		s.cancelLast = nil
	}
}

// ClearHistory deletes all entries from the request history.
func (s *RequestService) ClearHistory() {
	if s.store == nil {
		return
	}
	wsPath := ""
	if s.workspace != nil {
		wsPath = s.workspace.GetWorkspacePath()
	}
	if err := s.store.Clear(wsPath); err != nil {
		log.Printf("clearing history: %v", err)
	}
}

func (s *RequestService) recordHistory(input RequestInput, out ResponseOutput) {
	if s.store == nil || out.Error != "" {
		return
	}
	headers := make([]history.Header, len(input.Headers))
	for i, h := range input.Headers {
		headers[i] = history.Header{Name: h.Name, Value: h.Value}
	}
	bodyType, bodyContent := "none", ""
	if input.Body != nil {
		bodyType = input.Body.Type
		bodyContent = input.Body.Content
	}
	wsPath := ""
	if s.workspace != nil {
		wsPath = s.workspace.GetWorkspacePath()
	}
	if _, err := s.store.Record(wsPath, input.Method, input.URL, out.StatusCode, out.Status, out.DurationMs, out.Size, headers, bodyType, bodyContent); err != nil {
		log.Printf("writing history: %v", err)
	}
}

// ---- mapping helpers ----

func inputToDomain(in RequestInput) (*domain.Request, error) {
	req := &domain.Request{
		Method: domain.HTTPMethod(in.Method),
		URL:    in.URL,
	}

	for _, h := range in.Headers {
		req.Headers = append(req.Headers, domain.Header{Name: h.Name, Value: h.Value})
	}
	for _, p := range in.QueryParams {
		req.QueryParams = append(req.QueryParams, domain.QueryParam{Name: p.Name, Value: p.Value})
	}
	if in.Body != nil {
		req.Body = &domain.RequestBody{
			Type:    domain.BodyType(in.Body.Type),
			Content: in.Body.Content,
		}
	}
	if in.Auth != nil {
		req.Auth = inputAuthToDomain(in.Auth)
	}
	if in.TimeoutMs > 0 {
		req.Timeout = time.Duration(in.TimeoutMs) * time.Millisecond
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}
	return req, nil
}

func inputAuthToDomain(a *AuthIO) *domain.Auth {
	auth := &domain.Auth{Type: domain.AuthType(a.Type)}
	switch auth.Type {
	case domain.AuthBearer:
		auth.Bearer = &domain.BearerAuth{Token: a.Token}
	case domain.AuthBasic:
		auth.Basic = &domain.BasicAuth{Username: a.Username, Password: a.Password}
	case domain.AuthAPIKey:
		auth.APIKey = &domain.APIKeyAuth{
			Key:      a.Key,
			Value:    a.Value,
			Location: domain.APIKeyLocation(a.Location),
		}
	}
	return auth
}

func domainToOutput(r *domain.Response) ResponseOutput {
	headers := make([]HeaderIO, 0, len(r.Headers))
	for _, h := range r.Headers {
		headers = append(headers, HeaderIO{Name: h.Name, Value: h.Value})
	}
	return ResponseOutput{
		StatusCode: r.StatusCode,
		Status:     r.Status,
		Headers:    headers,
		Body:       r.BodyString(),
		DurationMs: r.Duration.Milliseconds(),
		Size:       r.Size,
		Timing: TimingIO{
			DNSMs:      r.Timing.DNSMs,
			ConnectMs:  r.Timing.ConnectMs,
			TLSMs:      r.Timing.TLSMs,
			TTFBMs:     r.Timing.TTFBMs,
			TransferMs: r.Timing.TransferMs,
		},
	}
}
