package stdlib

import (
	"fmt"
	"regexp"
	"sync"
)

const (
	maxPatternLength = 1000
	maxInputSize     = 1024 * 1024 // 1MB
)

var (
	regexCache   = make(map[string]*regexp.Regexp)
	regexCacheMu sync.RWMutex
)

func getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if len(pattern) > maxPatternLength {
		return nil, fmt.Errorf("regex: pattern too long (%d chars, max %d)", len(pattern), maxPatternLength)
	}

	regexCacheMu.RLock()
	if re, ok := regexCache[pattern]; ok {
		regexCacheMu.RUnlock()
		return re, nil
	}
	regexCacheMu.RUnlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex: invalid pattern '%s': %s", pattern, err.Error())
	}

	regexCacheMu.Lock()
	regexCache[pattern] = re
	regexCacheMu.Unlock()

	return re, nil
}

func validateInputSize(input string) error {
	if len(input) > maxInputSize {
		return fmt.Errorf("regex: input too large (%d bytes, max %d)", len(input), maxInputSize)
	}
	return nil
}

// RegisterRegex registers regex functions in the stdlib registry.
// regex.match, regex.findAll, regex.replace are available without import (stdlib).
// matches() is kept as backward-compatible alias for regex.match.
func RegisterRegex(r *Registry) {
	regexMatch := func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("regex.match: requires (str, pattern)")
		}
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("regex.match: first argument must be a string")
		}
		pattern, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("regex.match: second argument must be a string")
		}

		if err := validateInputSize(s); err != nil {
			return nil, err
		}

		re, err := getCompiledRegex(pattern)
		if err != nil {
			return nil, err
		}

		return re.MatchString(s), nil
	}

	regexFindAll := func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("regex.findAll: requires (str, pattern)")
		}
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("regex.findAll: first argument must be a string")
		}
		pattern, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("regex.findAll: second argument must be a string")
		}

		if err := validateInputSize(s); err != nil {
			return nil, err
		}

		re, err := getCompiledRegex(pattern)
		if err != nil {
			return nil, err
		}

		matches := re.FindAllString(s, -1)
		result := make([]any, len(matches))
		for i, m := range matches {
			result[i] = m
		}
		return result, nil
	}

	regexReplace := func(params ...any) (any, error) {
		if len(params) < 3 {
			return nil, fmt.Errorf("regex.replace: requires (str, pattern, replacement)")
		}
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("regex.replace: first argument must be a string")
		}
		pattern, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("regex.replace: second argument must be a string")
		}
		replacement, ok := params[2].(string)
		if !ok {
			return nil, fmt.Errorf("regex.replace: third argument must be a string")
		}

		if err := validateInputSize(s); err != nil {
			return nil, err
		}

		re, err := getCompiledRegex(pattern)
		if err != nil {
			return nil, err
		}

		return re.ReplaceAllString(s, replacement), nil
	}

	r.RegisterEnv("regex", map[string]any{
		"match":   regexMatch,
		"findAll": regexFindAll,
		"replace": regexReplace,
	})
}
