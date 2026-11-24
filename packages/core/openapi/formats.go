package openapi

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// SupportedFormats lists the string formats whose values are checked. Other formats are accepted
// without a check, as the OpenAPI specification allows.
var SupportedFormats = []string{"date", "date-time", "email", "uuid", "ipv4", "ipv6", "byte"}

// The validation library keeps its format validators in package-level maps, so they apply to every
// check in the process: responses, generated requests and the schema assertion.
func init() {
	openapi3.DefineStringFormatValidator("email", openapi3.NewCallbackValidator(validateEmail))
	openapi3.DefineStringFormatValidator("uuid", openapi3.NewRegexpFormatValidator(openapi3.FormatOfStringForUUIDOfRFC9562))
	openapi3.DefineIPv4Format()
	openapi3.DefineIPv6Format()
}

// validateEmail accepts a single address, with no display name, comments or spaces. It is looser
// than the full grammar on purpose (a domain without a dot, such as user@localhost, is allowed) so
// a correct server is never reported for a rare but valid address.
func validateEmail(v string) error {
	if v == "" || strings.ContainsAny(v, " \t\r\n<>,;") {
		return fmt.Errorf("not an email address")
	}
	addr, err := mail.ParseAddress(v)
	if err != nil || addr.Address != v || addr.Name != "" {
		return fmt.Errorf("not an email address")
	}
	return nil
}

var reNewFormatExample = regexp.MustCompile(`doesn't match the format "(email|uuid|ipv4|ipv6)"`)

// onlyAFormatExample reports whether a spec validation error is an example that breaks one of the
// formats this package registers.
func onlyAFormatExample(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "invalid example") && reNewFormatExample.MatchString(msg)
}
