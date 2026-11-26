package main

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/redact"
)

// secretName matches header and parameter names that carry credentials.
var secretName = regexp.MustCompile(`(?i)authorization|cookie|api[-_]?key|token|secret|passw|session|signature`)

const masked = "***"

// exchangeRequest describes a request as it was sent, for display next to the response: the
// resolved URL with its query parameters, the headers (including the one the auth block adds) and
// the body. Credentials never leave this function: sensitive header and parameter values are
// masked, and so is every secret environment or scenario value wherever it appears.
func exchangeRequest(req *domain.Request, secrets []string) (method, fullURL string, headers []HeaderIO, body string) {
	hide := func(s string) string {
		for _, sec := range secrets {
			if sec != "" {
				s = strings.ReplaceAll(s, sec, masked)
			}
		}
		return redact.Credentials(s)
	}

	var query []string
	addQuery := func(name, value string) {
		if secretName.MatchString(name) {
			query = append(query, url.QueryEscape(name)+"="+masked)
			return
		}
		query = append(query, url.QueryEscape(name)+"="+url.QueryEscape(value))
	}
	for _, q := range req.QueryParams {
		addQuery(q.Name, q.Value)
	}

	for _, h := range req.Headers {
		v := h.Value
		if secretName.MatchString(h.Name) {
			v = masked
		}
		headers = append(headers, HeaderIO{Name: h.Name, Value: hide(v)})
	}
	if a := req.Auth; a != nil {
		switch a.Type {
		case domain.AuthBearer:
			headers = append(headers, HeaderIO{Name: "Authorization", Value: "Bearer " + masked})
		case domain.AuthBasic:
			headers = append(headers, HeaderIO{Name: "Authorization", Value: "Basic " + masked})
		case domain.AuthAPIKey:
			if a.APIKey != nil && a.APIKey.Location == domain.APIKeyInQuery {
				query = append(query, url.QueryEscape(a.APIKey.Key)+"="+masked)
			} else if a.APIKey != nil {
				headers = append(headers, HeaderIO{Name: a.APIKey.Key, Value: masked})
			}
		}
	}

	fullURL = req.URL
	if len(query) > 0 {
		sep := "?"
		if strings.Contains(fullURL, "?") {
			sep = "&"
		}
		fullURL += sep + strings.Join(query, "&")
	}
	if req.Body != nil {
		body = req.Body.Content
	}
	return string(req.Method), hide(fullURL), headers, hide(body)
}
