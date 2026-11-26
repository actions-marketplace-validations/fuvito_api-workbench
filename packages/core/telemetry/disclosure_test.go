package telemetry

import (
	"reflect"
	"strings"
	"testing"
)

// A field added to what is sent must be added to what people are told.
func TestDisclosureNamesEveryFieldSent(t *testing.T) {
	named := map[string]string{
		"installId": "random identifier", "version": "version", "os": "operating system",
		"arch": "processor type", "cmd": "command name", "result": "result",
		"durationMs": "how long it took", "stepCount": "number of steps",
		"assertionCount": "assertions", "ts": "the time",
	}
	text := strings.Join(strings.Fields(Disclosure), " ")
	fields := reflect.TypeOf(payload{})
	for i := 0; i < fields.NumField(); i++ {
		tag, _, _ := strings.Cut(fields.Field(i).Tag.Get("json"), ",")
		words, ok := named[tag]
		if !ok {
			t.Errorf("%q is sent: name it in Disclosure, telemetry.md, the user guide and here", tag)
		} else if !strings.Contains(text, words) {
			t.Errorf("Disclosure does not name %q (%q)", tag, words)
		}
	}
}
