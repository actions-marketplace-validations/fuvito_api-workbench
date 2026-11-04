package domain

import "time"

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
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// BodyString returns the response body as a UTF-8 string.
func (r *Response) BodyString() string {
	return string(r.Body)
}
