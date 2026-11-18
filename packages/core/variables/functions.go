package variables

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// call is one parsed {{$name arg arg key=value}}.
type call struct {
	name string
	args []string
	opts map[string]string
}

var optionKeys = map[string]bool{"format": true, "tz": true, "decimals": true}

// tokenize splits s on spaces, keeping "double" and 'single' quoted text together.
func tokenize(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	var quote rune
	started := false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			started = true
		case r == ' ' || r == '\t':
			if started || cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in %q", s)
	}
	if started || cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}

// parseCall parses the text between {{ and }} of a function call, without the leading $.
func parseCall(inner string) (call, error) {
	tokens, err := tokenize(strings.TrimSpace(inner))
	if err != nil {
		return call{}, err
	}
	if len(tokens) == 0 || tokens[0] == "" {
		return call{}, fmt.Errorf("missing function name after $")
	}
	c := call{name: tokens[0], opts: map[string]string{}}
	for _, t := range tokens[1:] {
		if k, v, ok := strings.Cut(t, "="); ok && optionKeys[k] {
			c.opts[k] = v
			continue
		}
		c.args = append(c.args, t)
	}
	return c, nil
}

type function struct {
	min, max int      // argument count limits (max -1: any)
	opts     []string // option keys this function accepts
	usage    string   // shown in errors and in the docs
	run      func(e *Engine, c call) (string, error)
}

const (
	alphaChars    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alnumChars    = alphaChars + "0123456789"
	hexChars      = "0123456789abcdef"
	maxRandomText = 4096
)

var functions map[string]function

func init() {
	functions = map[string]function{
		"guid":         {0, 0, nil, "$guid", fnGUID},
		"uuid":         {0, 0, nil, "$guid", fnGUID},
		"randomHex":    {1, 1, nil, "$randomHex <length>", textFn(hexChars)},
		"randomString": {1, 1, nil, "$randomString <length>", textFn(alnumChars)},
		"randomAlpha":  {1, 1, nil, "$randomAlpha <length>", textFn(alphaChars)},
		"randomChoice": {1, -1, nil, "$randomChoice <a> <b> ...", fnChoice},
		"randomInt":    {0, 2, nil, "$randomInt [<min> <max>]", fnInt},
		"randomFloat":  {0, 3, []string{"decimals"}, "$randomFloat [<min> <max> [<decimals>]]", fnFloat},
		"randomBool":   {0, 0, nil, "$randomBool", fnBool},
		"timestamp":    {0, 0, nil, "$timestamp", func(e *Engine, _ call) (string, error) { return strconv.FormatInt(e.now().Unix(), 10), nil }},
		"timestampMs":  {0, 0, nil, "$timestampMs", func(e *Engine, _ call) (string, error) { return strconv.FormatInt(e.now().UnixMilli(), 10), nil }},
		"isoTimestamp": {0, 0, nil, "$isoTimestamp", func(e *Engine, _ call) (string, error) { return e.now().UTC().Format("2006-01-02T15:04:05.000Z"), nil }},
		"date":         {0, 1, []string{"format", "tz"}, "$date [<offset>] [format=…] [tz=…]", timeFn("date")},
		"datetime":     {0, 1, []string{"format", "tz"}, "$datetime [<offset>] [format=…] [tz=…]", timeFn("iso")},
		"time":         {0, 1, []string{"format", "tz"}, "$time [<offset>] [format=…] [tz=…]", timeFn("time")},
	}
}

// FunctionNames lists the built-in functions, for documentation and autocomplete.
func FunctionNames() []string {
	names := make([]string, 0, len(functions))
	for n := range functions {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Call evaluates one function call: inner is the text between {{ and }}, starting with $.
func (e *Engine) Call(inner string) (string, error) {
	inner = strings.TrimSpace(inner)
	if !strings.HasPrefix(inner, "$") {
		return "", fmt.Errorf("a function call starts with $")
	}
	c, err := parseCall(inner[1:])
	if err != nil {
		return "", fmt.Errorf("{{%s}}: %w", inner, err)
	}
	fn, ok := functions[c.name]
	if !ok {
		return "", fmt.Errorf("{{%s}}: unknown function $%s (known: %s)", inner, c.name, strings.Join(FunctionNames(), ", "))
	}
	if len(c.args) < fn.min || (fn.max >= 0 && len(c.args) > fn.max) {
		return "", fmt.Errorf("{{%s}}: wrong number of arguments; use %s", inner, fn.usage)
	}
	for k := range c.opts {
		allowed := false
		for _, o := range fn.opts {
			if o == k {
				allowed = true
			}
		}
		if !allowed {
			return "", fmt.Errorf("{{%s}}: $%s does not take %s=; use %s", inner, c.name, k, fn.usage)
		}
	}
	out, err := fn.run(e, c)
	if err != nil {
		return "", fmt.Errorf("{{%s}}: %w", inner, err)
	}
	return out, nil
}

func fnGUID(e *Engine, _ call) (string, error) {
	r := e.draw()
	var b [16]byte
	for i := range b {
		b[i] = byte(r.Uint32())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func textFn(charset string) func(e *Engine, c call) (string, error) {
	return func(e *Engine, c call) (string, error) {
		n, err := strconv.Atoi(c.args[0])
		if err != nil || n < 1 || n > maxRandomText {
			return "", fmt.Errorf("length must be a whole number from 1 to %d", maxRandomText)
		}
		r := e.draw()
		out := make([]byte, n)
		for i := range out {
			out[i] = charset[r.IntN(len(charset))]
		}
		return string(out), nil
	}
}

func fnChoice(e *Engine, c call) (string, error) {
	return c.args[e.draw().IntN(len(c.args))], nil
}

func fnInt(e *Engine, c call) (string, error) {
	lo, hi := int64(0), int64(1000)
	switch len(c.args) {
	case 1:
		return "", fmt.Errorf("give both a minimum and a maximum, or neither")
	case 2:
		var err1, err2 error
		lo, err1 = strconv.ParseInt(c.args[0], 10, 64)
		hi, err2 = strconv.ParseInt(c.args[1], 10, 64)
		if err1 != nil || err2 != nil {
			return "", fmt.Errorf("minimum and maximum must be whole numbers")
		}
	}
	if lo > hi {
		return "", fmt.Errorf("the minimum %d is larger than the maximum %d", lo, hi)
	}
	if hi-lo < 0 || hi-lo == math.MaxInt64 {
		return "", fmt.Errorf("the range is too large")
	}
	return strconv.FormatInt(lo+e.draw().Int64N(hi-lo+1), 10), nil
}

func fnFloat(e *Engine, c call) (string, error) {
	lo, hi, decimals := 0.0, 1.0, 2
	args := c.args
	if len(args) == 1 {
		return "", fmt.Errorf("give a minimum and a maximum (and optionally the decimals), or nothing")
	}
	if len(args) >= 2 {
		var err1, err2 error
		lo, err1 = strconv.ParseFloat(args[0], 64)
		hi, err2 = strconv.ParseFloat(args[1], 64)
		if err1 != nil || err2 != nil {
			return "", fmt.Errorf("minimum and maximum must be numbers")
		}
	}
	d := c.opts["decimals"]
	if len(args) == 3 {
		d = args[2]
	}
	if d != "" {
		n, err := strconv.Atoi(d)
		if err != nil || n < 0 || n > 15 {
			return "", fmt.Errorf("decimals must be a whole number from 0 to 15")
		}
		decimals = n
	}
	if lo > hi || math.IsNaN(lo) || math.IsNaN(hi) || math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		return "", fmt.Errorf("the minimum must not be larger than the maximum")
	}
	v := lo + e.draw().Float64()*(hi-lo)
	return strconv.FormatFloat(v, 'f', decimals, 64), nil
}

func fnBool(e *Engine, _ call) (string, error) {
	if e.draw().IntN(2) == 0 {
		return "false", nil
	}
	return "true", nil
}

func timeFn(defaultFormat string) func(e *Engine, c call) (string, error) {
	return func(e *Engine, c call) (string, error) {
		loc, err := location(c.opts["tz"])
		if err != nil {
			return "", err
		}
		t := e.now().In(loc)
		if len(c.args) == 1 {
			if !isOffset(c.args[0]) {
				return "", fmt.Errorf("%q is not an offset like +7d or -1M", c.args[0])
			}
			if t, err = applyOffset(t, c.args[0]); err != nil {
				return "", err
			}
		}
		return formatTime(t, c.opts["format"], defaultFormat)
	}
}
