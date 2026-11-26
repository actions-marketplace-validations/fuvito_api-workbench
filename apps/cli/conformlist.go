package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

// listExcluded are the conform flags that only mean something when requests are sent.
var listExcluded = []string{
	"negative", "baseline", "write-baseline", "report", "setup", "coverage", "explain", "fail-fast", "fail-on",
	"request-timeout", "max-response-time", "max-response-time-for",
}

// checkListFlags refuses a flag typed next to --list that --list would silently ignore. It must
// run before the project file is applied, which also sets flags: its values are ignored instead.
func checkListFlags(cmd *cobra.Command) error {
	var typed []string
	for _, name := range listExcluded {
		if cmd.Flags().Changed(name) {
			typed = append(typed, "--"+name)
		}
	}
	if len(typed) > 0 {
		return fmt.Errorf("--list sends nothing, so it cannot be used with %s", strings.Join(typed, ", "))
	}
	if f := cmd.Flags().Lookup("format"); f.Changed {
		if v := strings.ToLower(f.Value.String()); v != "text" && v != "json" {
			return fmt.Errorf("--list writes text or json, not %q", f.Value.String())
		}
	}
	return nil
}

// renderNeeds returns the listing in the chosen format.
func renderNeeds(list *conformance.NeedsList, format string) ([]byte, error) {
	if format == "json" {
		b, err := reporting.NeedsJSON(list)
		if err != nil {
			return nil, err
		}
		return append(b, '\n'), nil
	}
	return []byte(reporting.NeedsText(list)), nil
}
