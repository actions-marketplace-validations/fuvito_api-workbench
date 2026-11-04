package domain

import (
	"errors"
	"time"
)

type HTTPMethod string

const (
	MethodGET     HTTPMethod = "GET"
	MethodPOST    HTTPMethod = "POST"
	MethodPUT     HTTPMethod = "PUT"
	MethodPATCH   HTTPMethod = "PATCH"
	MethodDELETE  HTTPMethod = "DELETE"
	MethodHEAD    HTTPMethod = "HEAD"
	MethodOPTIONS HTTPMethod = "OPTIONS"
)

func (m HTTPMethod) IsValid() bool {
	switch m {
	case MethodGET, MethodPOST, MethodPUT, MethodPATCH, MethodDELETE, MethodHEAD, MethodOPTIONS:
		return true
	}
	return false
}

type BodyType string

const (
	BodyNone           BodyType = "none"
	BodyJSON           BodyType = "json"
	BodyRaw            BodyType = "raw"
	BodyFormURLEncoded BodyType = "form-urlencoded"
	BodyMultipart      BodyType = "multipart"
)

type Header struct {
	Name  string
	Value string
}

type QueryParam struct {
	Name  string
	Value string
}

type RequestBody struct {
	Type    BodyType
	Content string
}

// Request is the canonical definition of an HTTP request.
// URL and header values may contain {{variable}} references resolved before execution.
type Request struct {
	SchemaVersion string
	Name          string
	Description   string
	Method        HTTPMethod
	URL           string
	Headers       []Header
	QueryParams   []QueryParam
	Body          *RequestBody
	Auth          *Auth
	Timeout       time.Duration // zero means use executor default
	Assertions    []Assertion
}

func (r *Request) Validate() error {
	if !r.Method.IsValid() {
		return errors.New("invalid or missing HTTP method")
	}
	if r.URL == "" {
		return errors.New("URL is required")
	}
	return nil
}
