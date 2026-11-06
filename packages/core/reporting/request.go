package reporting

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

// TextRequest formats a single HTTP request execution for terminal output.
// Plain ASCII, CI-safe (no ANSI codes).
func TextRequest(method, url string, resp *domain.Response, execErr error) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s %s\n", method, url)
	fmt.Fprintln(&b)

	if execErr != nil {
		fmt.Fprintf(&b, "ERROR: %s\n", execErr.Error())
		return b.String()
	}

	fmt.Fprintf(&b, "%s  %dms  %dB\n", resp.Status, resp.Duration.Milliseconds(), resp.Size)

	if len(resp.Headers) > 0 {
		fmt.Fprintln(&b)
		for _, h := range resp.Headers {
			fmt.Fprintf(&b, "%s: %s\n", h.Name, h.Value)
		}
	}

	body := resp.BodyString()
	if body != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, prettyBody(resp, body))
	}

	return b.String()
}

// JSONRequestReport is the machine-readable output for a single request execution.
type JSONRequestReport struct {
	Method     string          `json:"method"`
	URL        string          `json:"url"`
	StatusCode int             `json:"statusCode"`
	Status     string          `json:"status"`
	DurationMs int64           `json:"durationMs"`
	SizeBytes  int64           `json:"sizeBytes"`
	Headers    []domain.Header `json:"headers"`
	Body       string          `json:"body"`
	Error      string          `json:"error"`
}

// JSONRequest formats a single HTTP request execution as a JSON report.
func JSONRequest(method, url string, resp *domain.Response, execErr error) ([]byte, error) {
	report := JSONRequestReport{
		Method:  method,
		URL:     url,
		Headers: []domain.Header{},
	}
	if execErr != nil {
		report.Error = execErr.Error()
	} else {
		report.StatusCode = resp.StatusCode
		report.Status = resp.Status
		report.DurationMs = resp.Duration.Milliseconds()
		report.SizeBytes = resp.Size
		if resp.Headers != nil {
			report.Headers = resp.Headers
		}
		report.Body = resp.BodyString()
	}
	return json.MarshalIndent(report, "", "  ")
}

// prettyBody returns the body pretty-printed if it looks like JSON, otherwise as-is.
func prettyBody(resp *domain.Response, body string) string {
	for _, h := range resp.Headers {
		if strings.EqualFold(h.Name, "content-type") && strings.Contains(h.Value, "json") {
			var v interface{}
			if err := json.Unmarshal(resp.Body, &v); err == nil {
				if pretty, err := json.MarshalIndent(v, "", "  "); err == nil {
					return string(pretty)
				}
			}
			break
		}
	}
	return body
}
