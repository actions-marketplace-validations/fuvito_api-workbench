package variables

import "time"

var checkClock = func() time.Time { return time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC) }

// Check reports every {{$function}} call in text that could never work: an unknown function, the
// wrong number of arguments, a bad offset, format or time zone. It does not need any run data, so
// it can be used when a file is validated, before anything is sent.
func Check(text string) []error {
	var errs []error
	eng := NewEngine(1).WithClock(checkClock)
	for _, m := range varPattern.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if len(name) == 0 || name[0] != '$' {
			continue
		}
		if _, err := eng.Call(name); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// References lists the plain {{name}} variable references in text (function calls are not
// included), in order of appearance.
func References(text string) []string {
	var names []string
	for _, m := range varPattern.FindAllStringSubmatch(text, -1) {
		if name := m[1]; name != "" && name[0] != '$' {
			names = append(names, name)
		}
	}
	return names
}
