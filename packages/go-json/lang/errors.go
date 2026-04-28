package lang

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ErrorCategory string

const (
	CategoryCompile ErrorCategory = "compile"
	CategoryRuntime ErrorCategory = "runtime"
	CategoryLimit   ErrorCategory = "limit"
	CategoryIO      ErrorCategory = "io"
)

type StackFrame struct {
	Function string `json:"function"`
	Step     int    `json:"step"`
	Program  string `json:"program,omitempty"`
}

// GoJSONError is the unified error type for all go-json errors.
// It carries rich context for debugging, visual editor display, and programmatic handling.
type GoJSONError struct {
	Code       string            `json:"code"`
	Title      string            `json:"title"`
	Category   ErrorCategory     `json:"category"`
	Message    string            `json:"message"`
	Fix        string            `json:"fix,omitempty"`
	Suggestion []string          `json:"suggestion,omitempty"`
	Step       int               `json:"step"`
	Function   string            `json:"function,omitempty"`
	Program    string            `json:"program,omitempty"`
	Context    map[string]any    `json:"context,omitempty"`
	Stack      []StackFrame      `json:"stack,omitempty"`
}

func (e *GoJSONError) Error() string {
	var b strings.Builder

	if e.Program != "" {
		fmt.Fprintf(&b, "[%s] ", e.Program)
	}
	if e.Function != "" {
		fmt.Fprintf(&b, "in %s() ", e.Function)
	}

	fmt.Fprintf(&b, "%s error at step %d: %s", e.Category, e.Step, e.Message)

	if e.Fix != "" {
		fmt.Fprintf(&b, "\n  fix: %s", e.Fix)
	}
	if len(e.Suggestion) > 0 {
		fmt.Fprintf(&b, "\n  did you mean: %s?", strings.Join(e.Suggestion, ", "))
	}
	if len(e.Stack) > 0 {
		b.WriteString("\n  stack:")
		for _, frame := range e.Stack {
			if frame.Function != "" {
				fmt.Fprintf(&b, "\n    → %s() step %d", frame.Function, frame.Step)
			} else {
				fmt.Fprintf(&b, "\n    → <main> step %d", frame.Step)
			}
		}
	}

	return b.String()
}

// JSON returns a structured map suitable for visual editor consumption.
func (e *GoJSONError) JSON() map[string]any {
	result := map[string]any{
		"code":     e.Code,
		"title":    e.Title,
		"category": string(e.Category),
		"message":  e.Message,
		"step":     e.Step,
	}
	if e.Fix != "" {
		result["fix"] = e.Fix
	}
	if len(e.Suggestion) > 0 {
		result["suggestion"] = e.Suggestion
	}
	if e.Function != "" {
		result["function"] = e.Function
	}
	if e.Program != "" {
		result["program"] = e.Program
	}
	if e.Context != nil {
		result["context"] = e.Context
	}
	if len(e.Stack) > 0 {
		frames := make([]map[string]any, len(e.Stack))
		for i, f := range e.Stack {
			frames[i] = map[string]any{"function": f.Function, "step": f.Step}
			if f.Program != "" {
				frames[i]["program"] = f.Program
			}
		}
		result["stack"] = frames
	}
	return result
}

// JSONString returns the JSON representation as a string.
func (e *GoJSONError) JSONString() string {
	data, _ := json.Marshal(e.JSON())
	return string(data)
}

// Short returns a one-line summary.
func (e *GoJSONError) Short() string {
	return fmt.Sprintf("%s: %s (step %d)", e.Code, e.Message, e.Step)
}

// --- Fluent builder methods ---

func (e *GoJSONError) WithFix(fix string) *GoJSONError {
	e.Fix = fix
	return e
}

func (e *GoJSONError) WithSuggestions(suggestions ...string) *GoJSONError {
	e.Suggestion = suggestions
	return e
}

func (e *GoJSONError) WithContext(ctx map[string]any) *GoJSONError {
	e.Context = ctx
	return e
}

func (e *GoJSONError) InFunction(name string) *GoJSONError {
	e.Function = name
	return e
}

func (e *GoJSONError) InProgram(name string) *GoJSONError {
	e.Program = name
	return e
}

func (e *GoJSONError) WithStack(stack []StackFrame) *GoJSONError {
	e.Stack = stack
	return e
}

// --- Constructor helpers ---

func CompileError(code, message string, step int) *GoJSONError {
	return &GoJSONError{
		Code:     code,
		Title:    "Compile Error",
		Category: CategoryCompile,
		Message:  message,
		Step:     step,
	}
}

func RuntimeError(code, message string, step int) *GoJSONError {
	return &GoJSONError{
		Code:     code,
		Title:    "Runtime Error",
		Category: CategoryRuntime,
		Message:  message,
		Step:     step,
	}
}

func LimitError(code, message string, step int) *GoJSONError {
	return &GoJSONError{
		Code:     code,
		Title:    "Limit Exceeded",
		Category: CategoryLimit,
		Message:  message,
		Step:     step,
	}
}

// --- Levenshtein distance ---

// levenshtein computes the edit distance between two strings using rune-based
// comparison for correct Unicode handling.
func levenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two rows instead of full matrix for O(min(la,lb)) space.
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost

			curr[j] = del
			if ins < curr[j] {
				curr[j] = ins
			}
			if sub < curr[j] {
				curr[j] = sub
			}
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

type suggestion struct {
	name     string
	distance int
}

// SuggestSimilar finds candidates similar to target within maxDistance,
// returning up to maxResults sorted by edit distance (closest first).
func SuggestSimilar(target string, candidates []string, maxResults, maxDistance int) []string {
	if target == "" || len(candidates) == 0 {
		return nil
	}

	var matches []suggestion
	for _, c := range candidates {
		if c == target {
			continue
		}
		d := levenshtein(target, c)
		if d <= maxDistance {
			matches = append(matches, suggestion{name: c, distance: d})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].distance != matches[j].distance {
			return matches[i].distance < matches[j].distance
		}
		return matches[i].name < matches[j].name
	})

	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	result := make([]string, len(matches))
	for i, m := range matches {
		result[i] = m.name
	}
	return result
}
