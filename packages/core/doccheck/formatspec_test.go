package doccheck

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/projectconfig"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// Every field a persisted file format can hold is named in apps/desktop/wails/help/format-spec.md,
// which is the human reference and the source of the AI prompt. A field added to a format without a
// line in the spec fails here (CLAUDE.md section 8: the spec changes in the same commit).
func TestFormatSpecNamesEveryField(t *testing.T) {
	path := filepath.Join("..", "..", "..", "apps", "desktop", "wails", "help", "format-spec.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)

	formats := map[string]reflect.Type{
		"request file":     reflect.TypeOf(serialization.RequestFile{}),
		"scenario file":    reflect.TypeOf(serialization.ScenarioFile{}),
		"environment file": reflect.TypeOf(serialization.EnvironmentFile{}),
		"bundle file":      reflect.TypeOf(serialization.BundleFile{}),
		"project file":     reflect.TypeOf(projectconfig.Config{}),
	}
	names := make([]string, 0, len(formats))
	for n := range formats {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, format := range names {
		var fields []string
		collect(formats[format], map[reflect.Type]bool{}, map[string]bool{}, &fields)
		sort.Strings(fields)
		for _, f := range fields {
			if !mentioned(doc, f) {
				t.Errorf("%s: the field %q is not named in format-spec.md (add it where the %s is described)", format, f, format)
			}
		}
		if len(fields) < 3 {
			t.Errorf("%s: only %d fields found; the reflection is not reaching them", format, len(fields))
		}
	}
}

// mentioned reports whether the spec names a field as a YAML key (name:), a code span (`name`) or a
// JSON key ("name").
func mentioned(doc, field string) bool {
	q := regexp.QuoteMeta(field)
	return regexp.MustCompile(`(?m)(^|[\s{,\[` + "`" + `"-])` + q + `(:|` + "`" + `|")`).MatchString(doc)
}

// collect gathers the serialized field names of a type and of the structs it holds.
func collect(t reflect.Type, visited map[reflect.Type]bool, have map[string]bool, fields *[]string) {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || visited[t] {
		return
	}
	visited[t] = true
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := tagName(f.Tag.Get("yaml"))
		if name == "" {
			name = tagName(f.Tag.Get("json"))
		}
		if name != "" && name != "-" && !have[name] {
			have[name] = true
			*fields = append(*fields, name)
		}
		collect(f.Type, visited, have, fields)
	}
}

func tagName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	return name
}

func TestMentionedFindsKeysAndRejectsOthers(t *testing.T) {
	doc := "schemaVersion: \"1\"\n- `requestRef` names a file\nplain words about retry\n  retry: 3\n{\"title\": 1}\n"
	for _, yes := range []string{"schemaVersion", "requestRef", "retry", "title"} {
		if !mentioned(doc, yes) {
			t.Errorf("%s should be found", yes)
		}
	}
	for _, no := range []string{"zzzUnknown", "plain", "schema"} {
		if mentioned(doc, no) {
			t.Errorf("%s must not be found", no)
		}
	}
}
