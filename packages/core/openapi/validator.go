package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// Kinds of violation a response check can report. An empty Kind is a violation this package does
// not classify (the message says what it is).
const (
	KindBody             = "body"               // the body does not match the declared schema
	KindUndeclaredStatus = "undeclared-status"  // the status code is not declared for the operation
	KindContentType      = "wrong-content-type" // the response Content-Type is not one the spec declares
	KindMissingHeader    = "missing-header"     // a header the spec requires was not sent
	KindBadHeader        = "bad-header"         // a declared header has a value of the wrong type
	KindRequestParameter = "request-parameter"  // a path, query or header value in the request does not fit its parameter
	KindRequestBody      = "request-body"       // the request body does not fit the spec
	KindBodyTooLarge     = "body-too-large"     // the response is longer than the read limit and was cut off
	KindSlowResponse     = "slow-response"      // the response took longer than the limit given for the run
	KindBodyEmpty        = "body-empty"         // the response body is empty where the spec declares a schema
	KindBodyNotJSON      = "body-not-json"      // the response body cannot be parsed as JSON
	KindBodyBranch       = "body-branch"        // the body fits none (Subject "none") or several (Subject "several") of the oneOf/anyOf shapes
)

// ValidationError is a single violation.
type ValidationError struct {
	Path    string
	Message string
	// Kind classifies the violation (one of the Kind constants, or empty).
	Kind string
	// Subject is what the violation is about when Kind says so: the status code, the Content-Type
	// that was sent, or the header or parameter name.
	Subject string
	// In is where a request parameter lives: "path", "query", "header", "cookie" or "body".
	In string
	// Expected, Got and Variable describe a request body field whose value is not valid JSON:
	// the type the spec declares, the value that was inserted and the {{variable}} it came from.
	Expected string
	Got      string
	Variable string
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
	return safeViolations(validateResponse(s, operationID, resp), resp.Body, resp.Headers)
}

func validateResponse(s *Spec, operationID string, resp *domain.Response) []ValidationError {
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
	if mediaType != nil && !strings.Contains(strings.ToLower(ct), "json") {
		return nil // declared media type this check cannot read (XML, CSV, ...): not checked
	}
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
	return safeViolations(validateResponseByPath(s, method, path, statusCode, resp), resp.Body, resp.Headers)
}

func validateResponseByPath(s *Spec, method, path string, statusCode int, resp *domain.Response) []ValidationError {
	router, err := s.pathRouter()
	if err != nil {
		return []ValidationError{{Message: "building router: " + err.Error()}}
	}

	httpReq, err := http.NewRequest(method, s.relativePath(path), nil)
	if err != nil {
		return []ValidationError{{Message: "building request: " + err.Error()}}
	}

	route, _, err := router.FindRoute(httpReq)
	if err != nil {
		return nil // path not in spec — skip
	}

	body := []byte(resp.BodyString())
	ct := contentType(resp)

	if resp.Truncated && !isOpaqueMediaType(ct) {
		announced := ""
		if resp.ContentLength > 0 {
			announced = fmt.Sprintf(" (the server announced %d MB)", resp.ContentLength>>20)
		}
		return []ValidationError{{
			Kind: KindBodyTooLarge,
			Message: fmt.Sprintf("%s %s %d: the response body is larger than the %d MB Clear Signal reads%s, so it was cut off and not compared with the spec",
				route.Method, route.Path, statusCode, len(resp.Body)>>20, announced),
		}}
	}

	httpResp := &http.Response{
		StatusCode: resp.StatusCode,
		Header:     make(http.Header),
		Body:       nopCloser(body),
	}
	// Every header the server really sent goes into the check, so headers the spec declares are
	// compared with what arrived. Content-Type falls back to JSON when the server sent none.
	for _, h := range resp.Headers {
		httpResp.Header.Add(h.Name, h.Value)
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

	var violations []ValidationError
	if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
		if me, ok := err.(*openapi3filter.ResponseError); ok {
			violations = responseViolations(route, statusCode, ct, me)
		} else {
			violations = []ValidationError{{Message: err.Error()}}
		}
	}
	// The library's JSON decoder is lenient (it repairs bytes that are not UTF-8 and stops after
	// the first value), so a body that is not strictly JSON is checked here. A problem with the
	// status or a header comes first, as before.
	if len(violations) == 0 || violations[0].Kind == KindBody {
		if why := strictJSONProblem(body, ct, route, statusCode); why != "" {
			return []ValidationError{{Kind: KindBodyNotJSON,
				Message: fmt.Sprintf("%s %s %d: the response body is not valid JSON (%s)", route.Method, route.Path, statusCode, why)}}
		}
	}
	return violations
}

// isJSONDocumentType is true for a media type whose body is one JSON document: application/json
// and any +json type (application/problem+json). A type that only contains "json" is not one:
// application/x-ndjson is one JSON value per line, application/json-seq is a record sequence.
func isJSONDocumentType(ct string) bool {
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		mt = strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
	}
	return mt == "application/json" || strings.HasSuffix(mt, "+json")
}

// strictJSONProblem says why a body the spec declares as JSON is not valid JSON text, or returns
// "". Only what the library would let through is looked for: bytes that are not UTF-8 (RFC 8259
// section 8.1) and text after the value.
func strictJSONProblem(body []byte, sentType string, route *routers.Route, status int) string {
	if len(body) == 0 || !isJSONDocumentType(sentType) {
		return ""
	}
	declaresJSON := false
	for _, ct := range declaredContentTypes(route, status) {
		if isJSONDocumentType(ct) {
			declaresJSON = true
		}
	}
	if !declaresJSON {
		return ""
	}
	if !utf8.Valid(body) {
		i := 0
		for i < len(body) {
			r, size := utf8.DecodeRune(body[i:])
			if r == utf8.RuneError && size <= 1 {
				break
			}
			i += size
		}
		return fmt.Sprintf("byte 0x%02x at offset %d is not UTF-8", body[i], i)
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	var v json.RawMessage
	if err := dec.Decode(&v); err != nil {
		return "" // the library reports a syntax error in its own words
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return "text after the JSON value"
	}
	return ""
}

var quotedName = regexp.MustCompile(`"([^"]+)"`)

// responseViolations turns the library's ResponseError into violations. The library reports an
// undeclared status, a wrong Content-Type and a missing header only in Reason (Err is nil); those
// used to be dropped, which made such responses look conforming and, when a header was declared,
// stopped the body from ever being checked.
func responseViolations(route *routers.Route, status int, sentType string, me *openapi3filter.ResponseError) []ValidationError {
	where := fmt.Sprintf("%s %s", route.Method, route.Path)
	switch {
	case me.Reason == "status is not supported":
		declared := declaredStatuses(route)
		return []ValidationError{{
			Kind: KindUndeclaredStatus, Subject: strconv.Itoa(status),
			Message: fmt.Sprintf("%s answered %d, but the spec declares only %s", where, status, listOrNone(declared)),
		}}
	case strings.Contains(me.Reason, "Content-Type has unexpected value"):
		return []ValidationError{{
			Kind: KindContentType, Subject: sentType,
			Message: fmt.Sprintf("%s %d answered with Content-Type %q, but the spec declares %s for this response; the body was not checked",
				where, status, sentType, listOrNone(declaredContentTypes(route, status))),
		}}
	case strings.HasPrefix(me.Reason, "response header ") && strings.HasSuffix(me.Reason, " missing"):
		name := firstQuoted(me.Reason)
		return []ValidationError{{
			Kind: KindMissingHeader, Subject: name,
			Message: fmt.Sprintf("%s %d: the response header %q is required by the spec but was not sent", where, status, name),
		}}
	case strings.HasPrefix(me.Reason, "unable to decode header"):
		name := firstQuoted(me.Reason)
		detail := ""
		if me.Err != nil {
			detail = ": " + strings.SplitN(me.Err.Error(), "\n", 2)[0]
		}
		return []ValidationError{{
			Kind: KindBadHeader, Subject: name,
			Message: fmt.Sprintf("%s %d: the response header %q has a value that does not fit the spec%s", where, status, name, detail),
		}}
	case me.Err != nil && strings.HasPrefix(me.Err.Error(), "unsupported content type"):
		return nil // no decoder for the declared media type: the body is not checked, not a violation
	case me.Err != nil:
		if v, ok := bodyShapeViolation(route, status, where, me.Err); ok {
			return []ValidationError{v}
		}
		errs := parseSchemaErrors(me.Err)
		for i := range errs {
			errs[i].Kind = KindBody
		}
		return errs
	}
	return []ValidationError{{Message: me.Error()}}
}

func firstQuoted(s string) string {
	if m := quotedName.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

func listOrNone(items []string) string {
	if len(items) == 0 {
		return "no responses"
	}
	return strings.Join(items, ", ")
}

func declaredStatuses(route *routers.Route) []string {
	if route.Operation == nil || route.Operation.Responses == nil {
		return nil
	}
	var codes []string
	for code := range route.Operation.Responses.Map() {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

func declaredContentTypes(route *routers.Route, status int) []string {
	if route.Operation == nil || route.Operation.Responses == nil {
		return nil
	}
	ref := route.Operation.Responses.Status(status)
	if ref == nil {
		ref = route.Operation.Responses.Default()
	}
	if ref == nil || ref.Value == nil {
		return nil
	}
	var types []string
	for t := range ref.Value.Content {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// HasOperation reports whether the spec declares an operation for method and path, so a caller
// can tell "matches the spec" apart from "nothing to check" (ValidateResponseByPath returns no
// violations in both cases).
func (s *Spec) HasOperation(method, path string) bool {
	router, err := s.pathRouter()
	if err != nil {
		return false
	}
	httpReq, err := http.NewRequest(method, s.relativePath(path), nil)
	if err != nil {
		return false
	}
	_, _, err = router.FindRoute(httpReq)
	return err == nil
}

// ValidateRequest checks that req conforms to the spec's declared request schema
// for the matching operation. Returns nil if valid or if the path is not in the spec.
func ValidateRequest(s *Spec, req *domain.Request) []ValidationError {
	var body []byte
	if req.Body != nil {
		body = []byte(req.Body.Content)
	}
	return safeViolations(validateRequest(s, req), body, req.Headers)
}

func validateRequest(s *Spec, req *domain.Request) []ValidationError {
	router, err := s.pathRouter()
	if err != nil {
		return []ValidationError{{Message: "building router: " + err.Error()}}
	}

	rawURL := req.URL
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if u, perr := url.Parse(rawURL); perr == nil {
		u.Path = s.relativePath(u.Path)
		u.RawPath = ""
		rawURL = u.String()
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
		b := []byte(req.Body.Content)
		if req.Body.Type == domain.BodyMultipart {
			b, ct = multipartForValidation(req.Body.Content)
		}
		httpReq.Header.Set("Content-Type", ct)
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
			return requestViolations(route, re)
		}
		return []ValidationError{{Message: err.Error()}}
	}
	return nil
}

// requestViolations turns the library's RequestError into violations that say which parameter or
// part of the body is wrong. The parameter, the "request body" context and the Reason used to be
// dropped, and a RequestError without an inner error produced no violation at all.
func requestViolations(route *routers.Route, re *openapi3filter.RequestError) []ValidationError {
	where := fmt.Sprintf("%s %s", route.Method, route.Path)
	reason := re.Reason
	if re.Err != nil {
		if reason == "" || reason == re.Err.Error() {
			reason = re.Err.Error()
		} else {
			reason += ": " + re.Err.Error()
		}
	}
	switch {
	case re.Parameter != nil:
		return []ValidationError{{
			Kind: KindRequestParameter, Subject: re.Parameter.Name, In: re.Parameter.In,
			Message: fmt.Sprintf("%s: the request %s parameter %q does not fit the spec: %s", where, re.Parameter.In, re.Parameter.Name, strings.SplitN(reason, "\n", 2)[0]),
		}}
	case re.RequestBody != nil:
		if re.Err != nil {
			errs := parseSchemaErrors(re.Err)
			for i := range errs {
				errs[i].Kind = KindRequestBody
				errs[i].Message = where + ": the request body does not fit the spec: " + errs[i].Message
			}
			return errs
		}
		return []ValidationError{{Kind: KindRequestBody, Message: where + ": the request body does not fit the spec: " + reason}}
	}
	return []ValidationError{{Message: where + ": " + reason}}
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
	Method     string
	URL        string
	StatusCode int   // HTTP status code returned by the server
	HadAuth    bool  // whether the request carried any auth credentials
	Passed     bool  // whether this step's assertions all passed
	DurationMs int64 // response time of the request, in milliseconds (counted only when a response arrived)
}

// ExercisedResult pairs a matched spec operation with the last-run pass/fail outcome.
type ExercisedResult struct {
	CoveredOperation
	Passed      bool
	StatusCodes []int // all unique status codes observed across matching executions
	HadAuth     bool  // whether at least one execution carried auth credentials
	// LastDurationMs is the response time of the last execution that got a response, SlowestDurationMs
	// the slowest in this run (both in milliseconds; 0 when none got a response).
	LastDurationMs    int64
	SlowestDurationMs int64
}

// CoveredOperation identifies a spec operation (method + path pattern) exercised or missed.
type CoveredOperation struct {
	Method string
	Path   string
}

func (o CoveredOperation) String() string { return o.Method + " " + o.Path }

// AllOperations returns every operation declared in the spec as (METHOD, path) pairs,
// sorted by path then method.
func (s *Spec) AllOperations() []CoveredOperation {
	var ops []CoveredOperation
	for path, item := range s.doc.Paths.Map() {
		for method := range item.Operations() {
			ops = append(ops, CoveredOperation{Method: strings.ToUpper(method), Path: path})
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})
	return ops
}

// ScenarioCoverageResult is the coverage computed from matching executed requests
// against an OpenAPI spec's declared operations.
type ScenarioCoverageResult struct {
	Covered          int
	Total            int
	Percent          float64
	Exercised        []CoveredOperation
	Missed           []CoveredOperation
	ExercisedResults []ExercisedResult // per-operation last pass/fail outcome
}

// ScenarioCoverage matches each executed request against the spec's declared operations
// and returns a coverage summary.
func (s *Spec) ScenarioCoverage(executed []ExecutedRequest) *ScenarioCoverageResult {
	router, err := s.pathRouter()
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

	exercised := make(map[opKey]struct{})
	exercisedPassed := make(map[opKey]bool)
	exercisedCodes := make(map[opKey]map[int]struct{})
	exercisedHadAuth := make(map[opKey]bool)
	exercisedLastMs := make(map[opKey]int64)
	exercisedSlowestMs := make(map[opKey]int64)
	for _, ex := range executed {
		matchPath := ex.URL
		if u, err := url.Parse(ex.URL); err == nil && u.Path != "" {
			matchPath = u.Path
		}
		req, err := http.NewRequest(strings.ToUpper(ex.Method), s.relativePath(matchPath), nil)
		if err != nil {
			continue
		}
		route, _, err := router.FindRoute(req)
		if err != nil {
			continue
		}
		k := opKey{strings.ToUpper(ex.Method), route.Path}
		exercised[k] = struct{}{}
		exercisedPassed[k] = ex.Passed // last execution wins
		if ex.StatusCode > 0 {
			if exercisedCodes[k] == nil {
				exercisedCodes[k] = make(map[int]struct{})
			}
			exercisedCodes[k][ex.StatusCode] = struct{}{}
			exercisedLastMs[k] = ex.DurationMs
			if ex.DurationMs > exercisedSlowestMs[k] {
				exercisedSlowestMs[k] = ex.DurationMs
			}
		}
		if ex.HadAuth {
			exercisedHadAuth[k] = true
		}
	}

	cr := &ScenarioCoverageResult{Total: len(all)}
	for k := range exercised {
		if _, ok := all[k]; ok {
			cr.Covered++
			op := CoveredOperation{Method: k.method, Path: k.path}
			cr.Exercised = append(cr.Exercised, op)
			var codes []int
			for code := range exercisedCodes[k] {
				codes = append(codes, code)
			}
			sort.Ints(codes)
			cr.ExercisedResults = append(cr.ExercisedResults, ExercisedResult{
				CoveredOperation:  op,
				Passed:            exercisedPassed[k],
				StatusCodes:       codes,
				HadAuth:           exercisedHadAuth[k],
				LastDurationMs:    exercisedLastMs[k],
				SlowestDurationMs: exercisedSlowestMs[k],
			})
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

// pathRouter matches paths the way the spec declares them: relative to a server's base path and
// whatever the host. Building the router with the servers in place would only match requests to
// the declared host and base path, so a bare path (what a report or a step carries) matched
// nothing and was silently skipped.
func (s *Spec) pathRouter() (routers.Router, error) {
	d := *s.doc
	d.Servers = nil
	return gorillamux.NewRouter(&d)
}

// serverBasePaths lists the base paths of the spec's servers (for example "/v1" for
// https://api.example.com/v1). Servers with variables in the path are left out.
func (s *Spec) serverBasePaths() []string {
	var out []string
	for _, sv := range s.doc.Servers {
		if sv == nil {
			continue
		}
		raw := sv.URL
		if i := strings.Index(raw, "://"); i >= 0 {
			raw = raw[i+3:]
			if j := strings.Index(raw, "/"); j >= 0 {
				raw = raw[j:]
			} else {
				raw = ""
			}
		}
		raw = strings.TrimRight(raw, "/")
		if raw == "" || !strings.HasPrefix(raw, "/") || strings.ContainsAny(raw, "{}") {
			continue
		}
		out = append(out, raw)
	}
	return out
}

// relativePath removes a server's base path from the front of path, so a path taken from a real
// URL (/v1/users) compares with the paths the spec declares (/users). A path already relative to
// the spec is returned as it is.
func (s *Spec) relativePath(path string) string {
	for _, base := range s.serverBasePaths() {
		if path == base {
			return "/"
		}
		if strings.HasPrefix(path, base+"/") {
			return path[len(base):]
		}
	}
	return path
}

// multipartForValidation turns the name=value pairs of a multipart body into a real multipart
// document, so the validator can read the fields (a bare "a=1&b=2" has no boundary and cannot be
// decoded). Fields given in the desktop's JSON form are written as text fields too.
func multipartForValidation(content string) ([]byte, string) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if strings.HasPrefix(content, "[") {
		var fields []struct{ N, V string }
		if json.Unmarshal([]byte(content), &fields) == nil {
			for _, f := range fields {
				_ = mw.WriteField(f.N, f.V)
			}
		}
	} else if values, err := url.ParseQuery(content); err == nil {
		names := make([]string, 0, len(values))
		for n := range values {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			for _, v := range values[n] {
				_ = mw.WriteField(n, v)
			}
		}
	}
	_ = mw.Close()
	return buf.Bytes(), mw.FormDataContentType()
}

// isOpaqueMediaType is true for media types whose body is never compared with a schema (binary
// downloads), where a body cut at the read limit changes nothing about the check.
func isOpaqueMediaType(ct string) bool {
	ct = strings.ToLower(ct)
	for _, p := range []string{"application/octet-stream", "image/", "video/", "audio/"} {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

// bodyShapeViolation classifies a body problem that is about the body as a whole, not a field: no
// body at all, a body that is not JSON, or a body that fits none (or several) of the oneOf / anyOf
// shapes. The library reports these in its own words ("EOF", "invalid character ...", the first
// failing branch); the messages here say what is wrong.
func bodyShapeViolation(route *routers.Route, status int, where string, err error) (ValidationError, bool) {
	text := err.Error()
	switch {
	case text == "EOF":
		return ValidationError{Kind: KindBodyEmpty, Message: fmt.Sprintf("%s %d: the response body is empty, but the spec declares a schema for it", where, status)}, true
	case strings.HasPrefix(text, "invalid character") || text == "unexpected EOF" || strings.HasPrefix(text, "unexpected end of JSON"):
		return ValidationError{Kind: KindBodyNotJSON, Message: fmt.Sprintf("%s %d: the response body is not valid JSON (%s)", where, status, firstLineOf(text))}, true
	}
	keyword, n := bodyComposition(route, status)
	if n == 0 {
		return ValidationError{}, false
	}
	switch {
	case strings.HasPrefix(text, "doesn't match schema due to"), strings.HasPrefix(text, "doesn't match any schema from"):
		closest := ""
		if rest := strings.TrimSpace(strings.TrimPrefix(firstLineOf(text), "doesn't match schema due to:")); rest != "" && !strings.HasPrefix(text, "doesn't match any schema from") {
			closest = "; the first shape fails because: " + rest
		}
		return ValidationError{Kind: KindBodyBranch, Subject: "none",
			Message: fmt.Sprintf("%s %d: the response body matches none of the %d shapes the spec allows (%s)%s", where, status, n, keyword, closest)}, true
	case strings.Contains(text, "matches more than one"):
		return ValidationError{Kind: KindBodyBranch, Subject: "several",
			Message: fmt.Sprintf("%s %d: the response body fits more than one of the %d shapes the spec allows (%s), so it is ambiguous", where, status, n, keyword)}, true
	}
	return ValidationError{}, false
}

// bodyComposition says whether the response's JSON schema is a oneOf or anyOf, and how many shapes
// it lists.
func bodyComposition(route *routers.Route, status int) (string, int) {
	if route.Operation == nil || route.Operation.Responses == nil {
		return "", 0
	}
	r := route.Operation.Responses.Value(strconv.Itoa(status))
	if r == nil {
		r = route.Operation.Responses.Value("default")
	}
	if r == nil || r.Value == nil {
		return "", 0
	}
	for ct, mt := range r.Value.Content {
		if !strings.Contains(ct, "json") || mt.Schema == nil || mt.Schema.Value == nil {
			continue
		}
		if n := len(mt.Schema.Value.OneOf); n > 0 {
			return "oneOf", n
		}
		if n := len(mt.Schema.Value.AnyOf); n > 0 {
			return "anyOf", n
		}
	}
	return "", 0
}

func firstLineOf(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
