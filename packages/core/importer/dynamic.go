package importer

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

var dollarRe = regexp.MustCompile(`\{\{\s*(\$[A-Za-z0-9_]+)\s*\}\}`)

// postmanDynamic maps Postman's dynamic variables to the equivalent clrsig functions. Names not
// listed are left as written and reported, so nothing is silently dropped.
var postmanDynamic = map[string]string{
	"$guid":               "$guid",
	"$randomUUID":         "$guid",
	"$timestamp":          "$timestamp",
	"$isoTimestamp":       "$isoTimestamp",
	"$randomInt":          "$randomInt",
	"$randomBoolean":      "$randomBool",
	"$randomAlphaNumeric": "$randomString 1",
	"$randomEmail":        "$randomEmail",
	"$randomFirstName":    "$randomFirstName",
	"$randomLastName":     "$randomLastName",
	"$randomFullName":     "$randomFullName",
	"$randomPhoneNumber":  "$randomPhone",
	"$randomWord":         "$randomWord",
}

// mapDynamic rewrites Postman dynamic variables inside req and returns the names it could not map.
func mapDynamic(req *domain.Request) []string {
	unsupported := map[string]bool{}
	fix := func(s string) string {
		return dollarRe.ReplaceAllStringFunc(s, func(m string) string {
			name := dollarRe.FindStringSubmatch(m)[1]
			if to, ok := postmanDynamic[name]; ok {
				return "{{" + to + "}}"
			}
			unsupported[name] = true
			return m
		})
	}
	req.URL = fix(req.URL)
	for i := range req.Headers {
		req.Headers[i].Value = fix(req.Headers[i].Value)
	}
	for i := range req.QueryParams {
		req.QueryParams[i].Value = fix(req.QueryParams[i].Value)
	}
	if req.Body != nil {
		req.Body.Content = fix(req.Body.Content)
	}
	if a := req.Auth; a != nil {
		if a.Bearer != nil {
			a.Bearer.Token = fix(a.Bearer.Token)
		}
		if a.Basic != nil {
			a.Basic.Username, a.Basic.Password = fix(a.Basic.Username), fix(a.Basic.Password)
		}
		if a.APIKey != nil {
			a.APIKey.Value = fix(a.APIKey.Value)
		}
	}
	names := make([]string, 0, len(unsupported))
	for n := range unsupported {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = fmt.Sprintf("%s is not supported by clrsig and was left as written", n)
	}
	return out
}
