package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fuvito/api-workbench/packages/core/clispec"
)

// cobraFor returns the real cobra command for a catalogued command name.
func cobraFor(t *testing.T, name string) *cobra.Command {
	t.Helper()
	switch name {
	case "run":
		return runCmd()
	case "conform":
		return conformCmd()
	case "diff":
		return diffCmd()
	}
	t.Fatalf("catalog command %q has no cobra command in cobraFor; add it here", name)
	return nil
}

func wantFlagType(in clispec.Input) string {
	if in.Repeatable {
		return "stringArray"
	}
	switch in.Kind {
	case clispec.KindBool:
		return "bool"
	case clispec.KindInt:
		return "int"
	}
	return "string"
}

// The desktop CLI Command Builder is driven by the clispec catalog. If a flag is added,
// renamed or removed in the CLI without updating the catalog, this test fails.
func TestCatalogMatchesCobra(t *testing.T) {
	for _, spec := range clispec.All() {
		spec := spec
		t.Run(spec.Name, func(t *testing.T) {
			cmd := cobraFor(t, spec.Name)

			if got := strings.Fields(cmd.Use)[0]; got != spec.Name {
				t.Errorf("cobra command is %q, catalog says %q", got, spec.Name)
			}

			wantArgs := strings.Count(cmd.Use, "<")
			if got := len(spec.Positionals()); got != wantArgs {
				t.Errorf("cobra Use %q has %d required argument(s), catalog has %d", cmd.Use, wantArgs, got)
			}

			inCatalog := map[string]bool{}
			for _, in := range spec.Flags() {
				inCatalog[in.Name] = true
				f := cmd.Flags().Lookup(in.Name)
				if f == nil {
					t.Errorf("catalog flag --%s does not exist on `clrsig %s`", in.Name, spec.Name)
					continue
				}
				if f.Shorthand != in.Short {
					t.Errorf("--%s: shorthand is %q in cobra, %q in catalog", in.Name, f.Shorthand, in.Short)
				}
				wantDefault := in.Default
				if in.Repeatable {
					wantDefault = "[]"
				}
				if f.DefValue != wantDefault {
					t.Errorf("--%s: default is %q in cobra, %q in catalog", in.Name, f.DefValue, wantDefault)
				}
				if got := f.Value.Type(); got != wantFlagType(in) {
					t.Errorf("--%s: cobra type is %q, catalog kind %q expects %q", in.Name, got, in.Kind, wantFlagType(in))
				}
				if in.Kind == clispec.KindEnum {
					for _, v := range in.Enum {
						if !strings.Contains(f.Usage, v) {
							t.Errorf("--%s: catalog allows %q but the flag help does not mention it: %q", in.Name, v, f.Usage)
						}
					}
				}
			}

			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				if f.Name == "help" || f.Hidden {
					return
				}
				if !inCatalog[f.Name] {
					t.Errorf("`clrsig %s` has --%s but the catalog does not (add it to packages/core/clispec/catalog.go)", spec.Name, f.Name)
				}
			})
		})
	}
}
