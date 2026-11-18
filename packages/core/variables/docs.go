package variables

// FunctionInfo describes one {{$function}} for documentation and editor autocomplete.
type FunctionInfo struct {
	Name        string `json:"name"`        // without the leading $
	Usage       string `json:"usage"`       // argument shape
	Description string `json:"description"` // one line
	Example     string `json:"example"`     // a complete, valid call without the braces
}

var functionDocs = map[string][2]string{
	"guid":            {"Random UUID (version 4)", "$guid"},
	"uuid":            {"Same as $guid", "$uuid"},
	"randomHex":       {"Random hexadecimal text of the given length", "$randomHex 16"},
	"randomString":    {"Random letters and digits of the given length", "$randomString 10"},
	"randomAlpha":     {"Random letters of the given length", "$randomAlpha 6"},
	"randomChoice":    {"One of the listed values, chosen at random", "$randomChoice red green blue"},
	"randomInt":       {"Random whole number (default 0 to 1000)", "$randomInt 1 100"},
	"randomFloat":     {"Random decimal number (default 0 to 1, 2 decimals)", "$randomFloat 1 10 decimals=2"},
	"randomBool":      {"true or false", "$randomBool"},
	"timestamp":       {"Current Unix time in seconds", "$timestamp"},
	"timestampMs":     {"Current Unix time in milliseconds", "$timestampMs"},
	"isoTimestamp":    {"Current time in ISO 8601 (UTC)", "$isoTimestamp"},
	"date":            {"Current date, with optional offset, format and tz", "$date +7d format=yyyy-MM-dd"},
	"datetime":        {"Current date and time, with optional offset, format and tz", "$datetime -2h"},
	"time":            {"Current time of day, with optional offset, format and tz", "$time +30m format=HH:mm"},
	"calc":            {"Arithmetic on numbers and numeric variables", "$calc price * quantity"},
	"randomEmail":     {"Fake email address at example.com", "$randomEmail"},
	"randomFirstName": {"Fake first name", "$randomFirstName"},
	"randomLastName":  {"Fake last name", "$randomLastName"},
	"randomFullName":  {"Fake first and last name", "$randomFullName"},
	"randomPhone":     {"Fake phone number on a fictional 555-01xx line", "$randomPhone"},
	"randomWord":      {"A random everyday word", "$randomWord"},
}

// Functions lists every built-in function with its documentation, sorted by name.
func Functions() []FunctionInfo {
	out := make([]FunctionInfo, 0, len(functions))
	for _, name := range FunctionNames() {
		d := functionDocs[name]
		out = append(out, FunctionInfo{Name: name, Usage: functions[name].usage, Description: d[0], Example: d[1]})
	}
	return out
}
