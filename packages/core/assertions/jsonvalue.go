package assertions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// maxExactInt is the largest integer a float64 holds exactly (2^53).
const maxExactInt = 1 << 53

// DecodeJSON reads a JSON document for jsonpath assertions and extractions. It rejects text after
// the value, and it keeps an integer that a float64 would round (an id above 2^53, a number
// beyond int64 or float64) as its exact digits instead of a rounded float; every other number is
// a float64, so filters such as ?(@.price > 5) keep working.
func DecodeJSON(body string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid character after top-level value: the body has text after the JSON value")
	}
	return exactNumbers(data), nil
}

func exactNumbers(v any) any {
	switch t := v.(type) {
	case json.Number:
		text := t.String()
		if n, err := strconv.ParseInt(text, 10, 64); err == nil {
			if n > -maxExactInt && n < maxExactInt {
				return float64(n)
			}
			return t
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil && !math.IsInf(f, 0) && !isBigIntegerText(text) {
			return f
		}
		return t
	case map[string]any:
		for k, x := range t {
			t[k] = exactNumbers(x)
		}
	case []any:
		for i, x := range t {
			t[i] = exactNumbers(x)
		}
	}
	return v
}

// isBigIntegerText is true for digits only (an integer beyond int64), which ParseFloat would round.
func isBigIntegerText(s string) bool {
	s = strings.TrimPrefix(s, "-")
	return s != "" && strings.Trim(s, "0123456789") == ""
}

// FormatValue is the text of a value found by jsonpath. A number is written in full (never in
// exponent form: 1234567 is "1234567", not "1.234567e+06").
func FormatValue(v any) string {
	switch t := v.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	}
	return fmt.Sprintf("%v", v)
}
