package variables

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var offsetRe = regexp.MustCompile(`^([+-])(\d+)(s|m|h|d|w|M|y)$`)

func isOffset(s string) bool { return offsetRe.MatchString(s) }

// applyOffset moves t by an offset such as +7d, -1M, +2h, +30m or -1y (units: s m h d w M y).
func applyOffset(t time.Time, offset string) (time.Time, error) {
	m := offsetRe.FindStringSubmatch(offset)
	if m == nil {
		return t, fmt.Errorf("bad offset %q (use +7d, -1M, +2h, +30m, -1y; units s m h d w M y)", offset)
	}
	n, err := strconv.Atoi(m[2])
	if err != nil || n > 100000 {
		return t, fmt.Errorf("offset %q is too large", offset)
	}
	if m[1] == "-" {
		n = -n
	}
	switch m[3] {
	case "s":
		return t.Add(time.Duration(n) * time.Second), nil
	case "m":
		return t.Add(time.Duration(n) * time.Minute), nil
	case "h":
		return t.Add(time.Duration(n) * time.Hour), nil
	case "d":
		return t.AddDate(0, 0, n), nil
	case "w":
		return t.AddDate(0, 0, 7*n), nil
	case "M":
		return t.AddDate(0, n, 0), nil
	default: // "y"
		return t.AddDate(n, 0, 0), nil
	}
}

// layoutTokens maps the familiar yyyy-MM-dd HH:mm:ss tokens to Go layout pieces, longest first.
var layoutTokens = []struct{ token, layout string }{
	{"yyyy", "2006"}, {"yy", "06"},
	{"MMMM", "January"}, {"MMM", "Jan"}, {"MM", "01"}, {"M", "1"},
	{"EEEE", "Monday"}, {"EEE", "Mon"},
	{"dd", "02"}, {"d", "2"},
	{"HH", "15"}, {"hh", "03"}, {"h", "3"},
	{"mm", "04"}, {"m", "4"}, {"ss", "05"}, {"s", "5"},
	{"SSS", "000"},
	{"a", "PM"},
	{"XXX", "Z07:00"}, {"ZZZ", "-0700"}, {"zzz", "MST"},
}

// patternToLayout converts a pattern such as "dd/MM/yyyy HH:mm" to a Go layout. Text in single
// quotes is literal ('T'); any other letter that is not a token is an error, so a typo is caught.
func patternToLayout(pattern string) (string, error) {
	var out strings.Builder
	rest := pattern
	for len(rest) > 0 {
		if rest[0] == '\'' {
			end := strings.IndexByte(rest[1:], '\'')
			if end < 0 {
				return "", fmt.Errorf("format %q has an unclosed quote", pattern)
			}
			out.WriteString(rest[1 : 1+end])
			rest = rest[end+2:]
			continue
		}
		matched := false
		for _, t := range layoutTokens {
			if strings.HasPrefix(rest, t.token) {
				out.WriteString(t.layout)
				rest = rest[len(t.token):]
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		c := rest[0]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return "", fmt.Errorf("format %q: %q is not a known token (use yyyy MM dd HH mm ss, or quote text like 'T')", pattern, string(c))
		}
		out.WriteByte(c)
		rest = rest[1:]
	}
	return out.String(), nil
}

// formatTime renders t with a named format or a pattern. Named formats: iso, date, time, rfc1123,
// unix, unixms.
func formatTime(t time.Time, format, defaultNamed string) (string, error) {
	if format == "" {
		format = defaultNamed
	}
	switch format {
	case "iso":
		return t.Format("2006-01-02T15:04:05Z07:00"), nil
	case "date":
		return t.Format("2006-01-02"), nil
	case "time":
		return t.Format("15:04:05"), nil
	case "rfc1123":
		return t.Format(time.RFC1123), nil
	case "unix":
		return strconv.FormatInt(t.Unix(), 10), nil
	case "unixms":
		return strconv.FormatInt(t.UnixMilli(), 10), nil
	}
	layout, err := patternToLayout(format)
	if err != nil {
		return "", err
	}
	return t.Format(layout), nil
}

func location(name string) (*time.Location, error) {
	switch strings.ToLower(name) {
	case "", "utc":
		return time.UTC, nil
	case "local":
		return time.Local, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %q (use a name such as Europe/Istanbul, utc or local)", name)
	}
	return loc, nil
}
