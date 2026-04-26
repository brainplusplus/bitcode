package validation

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

func isEmpty(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case bool:
		return false
	case float64:
		return false
	case int:
		return false
	case json.Number:
		return false
	}
	return false
}

func toString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == math.Trunc(v) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", val)
}

func toFloat(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}

func validateEmail(val string) bool {
	if len(val) > 254 {
		return false
	}
	at := strings.LastIndex(val, "@")
	if at < 1 || at >= len(val)-1 {
		return false
	}
	local := val[:at]
	domain := val[at+1:]
	if len(local) > 64 || len(domain) < 1 {
		return false
	}
	if !strings.Contains(domain, ".") {
		return false
	}
	parts := strings.Split(domain, ".")
	for _, p := range parts {
		if len(p) == 0 || len(p) > 63 {
			return false
		}
	}
	return true
}

func validateURL(val string) bool {
	u, err := url.ParseRequestURI(val)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

var phoneRegex = regexp.MustCompile(`^\+?[0-9\s\-\(\)\.]{7,20}$`)

func validatePhone(val string) bool {
	return phoneRegex.MatchString(val)
}

func validateIP(val string) bool {
	return net.ParseIP(val) != nil
}

func validateIPv4(val string) bool {
	ip := net.ParseIP(val)
	return ip != nil && ip.To4() != nil
}

func validateIPv6(val string) bool {
	ip := net.ParseIP(val)
	return ip != nil && ip.To4() == nil
}

func validateUUID(val string) bool {
	_, err := uuid.Parse(val)
	return err == nil
}

func validateJSON(val string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(val), &js) == nil
}

func validateAlpha(val string) bool {
	for _, r := range val {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return len(val) > 0
}

func validateAlphaNum(val string) bool {
	for _, r := range val {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return len(val) > 0
}

func validateAlphaDash(val string) bool {
	for _, r := range val {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
	}
	return len(val) > 0
}

func validateNumeric(val string) bool {
	_, err := strconv.ParseFloat(val, 64)
	return err == nil
}

func validateRegex(val, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(val)
}

func validateStartsWith(val string, prefixes any) bool {
	switch p := prefixes.(type) {
	case string:
		return strings.HasPrefix(val, p)
	case []any:
		for _, prefix := range p {
			if s, ok := prefix.(string); ok && strings.HasPrefix(val, s) {
				return true
			}
		}
	case []string:
		for _, prefix := range p {
			if strings.HasPrefix(val, prefix) {
				return true
			}
		}
	}
	return false
}

func validateEndsWith(val string, suffixes any) bool {
	switch p := suffixes.(type) {
	case string:
		return strings.HasSuffix(val, p)
	case []any:
		for _, suffix := range p {
			if s, ok := suffix.(string); ok && strings.HasSuffix(val, s) {
				return true
			}
		}
	case []string:
		for _, suffix := range p {
			if strings.HasSuffix(val, suffix) {
				return true
			}
		}
	}
	return false
}

func parseDate(val string) (time.Time, bool) {
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, val); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func resolveDateValue(ref string, data map[string]any) (time.Time, bool) {
	switch ref {
	case "today":
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), true
	case "now":
		return time.Now(), true
	}
	if t, ok := parseDate(ref); ok {
		return t, true
	}
	if fieldVal, ok := data[ref]; ok {
		return parseDate(toString(fieldVal))
	}
	return time.Time{}, false
}

func anyEquals(val any, target any) bool {
	return toString(val) == toString(target)
}

func anyInList(val any, list []any) bool {
	s := toString(val)
	for _, item := range list {
		if toString(item) == s {
			return true
		}
	}
	return false
}

func parseFileSize(size string) int64 {
	size = strings.TrimSpace(strings.ToUpper(size))
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}
	for suffix, mult := range multipliers {
		if strings.HasSuffix(size, suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(size, suffix))
			if n, err := strconv.ParseFloat(numStr, 64); err == nil {
				return int64(n * float64(mult))
			}
		}
	}
	if n, err := strconv.ParseInt(size, 10, 64); err == nil {
		return n
	}
	return 0
}
