package domain

import "time"

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
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// BodyString returns the response body as a UTF-8 string.
func (r *Response) BodyString() string {
	return string(r.Body)
}
