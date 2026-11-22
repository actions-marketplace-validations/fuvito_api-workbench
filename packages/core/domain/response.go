package domain

import (
	"strings"
	"time"
)

// Timing holds per-phase latency for a single HTTP request, measured via httptrace.
// Zero values mean the phase did not occur (e.g. TLSMs is 0 for plain HTTP).
type Timing struct {
	DNSMs      int64 // time to resolve the hostname
	ConnectMs  int64 // time to establish the TCP connection
	TLSMs      int64 // time to complete the TLS handshake (0 for HTTP)
	TTFBMs     int64 // time from request sent to first byte of response body
	TransferMs int64 // time to read the full response body after first byte
}

// TotalMs returns the sum of all timing phases.
func (t Timing) TotalMs() int64 {
	return t.DNSMs + t.ConnectMs + t.TLSMs + t.TTFBMs + t.TransferMs
}

// Response is the result of executing a Request.
// Error handling is done by the executor returning (nil, error) on network failure;
// Response is only populated when the server returned an HTTP response.
type Response struct {
	StatusCode int
	Status     string // e.g. "200 OK"
	Headers    []Header
	Body       []byte
	Duration   time.Duration
	Size       int64
	Timing     Timing
	// Truncated is true when the body was longer than the executor reads, so Body holds only its
	// first part; ContentLength is what the server announced (-1 when it did not say).
	Truncated     bool
	ContentLength int64
}

// HeaderValue returns the value of a response header (case-insensitive). A header the server sent
// more than once is one list: its values are joined with ", " in the order they arrived (RFC 9110
// section 5.3), so an assertion or an extraction can see every one of them.
func (r *Response) HeaderValue(name string) (string, bool) {
	var vals []string
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			vals = append(vals, h.Value)
		}
	}
	if len(vals) == 0 {
		return "", false
	}
	return strings.Join(vals, ", "), true
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// BodyString returns the response body as a UTF-8 string.
func (r *Response) BodyString() string {
	return string(r.Body)
}
