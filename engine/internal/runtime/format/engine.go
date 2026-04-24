package format

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SequenceProvider interface {
	NextValue(modelName, fieldName, sequenceKey string, step int) (int64, error)
}

type FormatContext struct {
	Data      map[string]any
	Session   map[string]any
	Settings  map[string]string
	ModelName string
	Module    string
	Now       time.Time
}

type Engine struct {
	sequenceProvider SequenceProvider
}

func NewEngine(sp SequenceProvider) *Engine {
	return &Engine{sequenceProvider: sp}
}

var tokenRe = regexp.MustCompile(`\{([^}]+)\}`)
var seqRe = regexp.MustCompile(`^sequence\((\d+)\)$`)
var substringRe = regexp.MustCompile(`^substring\((.+),\s*(\d+),\s*(\d+)\)$`)
var upperRe = regexp.MustCompile(`^upper\((.+)\)$`)
var lowerRe = regexp.MustCompile(`^lower\((.+)\)$`)
var hashRe = regexp.MustCompile(`^hash\((.+)\)$`)
var randomRe = regexp.MustCompile(`^random\((\d+)\)$`)
var randomFixedRe = regexp.MustCompile(`^random_fixed\((\d+),\s*(\d+),\s*(\d+)\)$`)

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func (e *Engine) Resolve(template string, ctx *FormatContext, modelName, fieldName, resetMode string, step int) (string, error) {
	if step <= 0 {
		step = 1
	}

	result := template
	var seqMatches [][]int

	allMatches := tokenRe.FindAllStringSubmatchIndex(result, -1)
	type pendingSeq struct {
		fullMatch string
		digits    int
	}
	var seqs []pendingSeq

	for i := len(allMatches) - 1; i >= 0; i-- {
		m := allMatches[i]
		token := result[m[2]:m[3]]

		if seqRe.MatchString(token) {
			sm := seqRe.FindStringSubmatch(token)
			digits, _ := strconv.Atoi(sm[1])
			seqs = append(seqs, pendingSeq{fullMatch: result[m[0]:m[1]], digits: digits})
			seqMatches = append(seqMatches, m)
			continue
		}

		resolved, err := e.resolveToken(token, ctx)
		if err != nil {
			return "", err
		}
		result = result[:m[0]] + resolved + result[m[1]:]
	}

	if len(seqs) > 0 {
		if e.sequenceProvider == nil {
			return "", fmt.Errorf("sequence provider is nil")
		}

		var seqKey string
		if resetMode == "key" {
			seqKey = tokenRe.ReplaceAllString(result, "")
			seqKey = strings.TrimSpace(seqKey)
			if seqKey == "" {
				seqKey = fmt.Sprintf("%s:%s", modelName, fieldName)
			}
		} else {
			seqKey = buildSequenceKeyFromReset(modelName, fieldName, resetMode, ctx.Now)
		}

		for _, sq := range seqs {
			val, err := e.sequenceProvider.NextValue(modelName, fieldName, seqKey, step)
			if err != nil {
				return "", fmt.Errorf("sequence error: %w", err)
			}
			padded := fmt.Sprintf("%0*d", sq.digits, val)
			result = strings.Replace(result, sq.fullMatch, padded, 1)
		}
	}

	return result, nil
}

func buildSequenceKeyFromReset(modelName, fieldName, reset string, now time.Time) string {
	base := fmt.Sprintf("%s:%s", modelName, fieldName)
	switch reset {
	case "yearly":
		return fmt.Sprintf("%s:%04d", base, now.Year())
	case "monthly":
		return fmt.Sprintf("%s:%04d-%02d", base, now.Year(), now.Month())
	case "daily":
		return fmt.Sprintf("%s:%04d-%02d-%02d", base, now.Year(), now.Month(), now.Day())
	case "hourly":
		return fmt.Sprintf("%s:%04d-%02d-%02dT%02d", base, now.Year(), now.Month(), now.Day(), now.Hour())
	case "minutely":
		return fmt.Sprintf("%s:%04d-%02d-%02dT%02d:%02d", base, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())
	default:
		return base
	}
}

func (e *Engine) resolveToken(token string, ctx *FormatContext) (string, error) {
	if strings.HasPrefix(token, "data.") {
		field := strings.TrimPrefix(token, "data.")
		if ctx.Data == nil {
			return "", fmt.Errorf("data context is nil, cannot resolve %q", token)
		}
		v, ok := ctx.Data[field]
		if !ok {
			return "", fmt.Errorf("data field %q not found", field)
		}
		return fmt.Sprintf("%v", v), nil
	}

	if strings.HasPrefix(token, "session.") {
		key := strings.TrimPrefix(token, "session.")
		if ctx.Session == nil {
			return "", nil
		}
		v, ok := ctx.Session[key]
		if !ok {
			return "", nil
		}
		return fmt.Sprintf("%v", v), nil
	}

	if strings.HasPrefix(token, "setting.") {
		key := strings.TrimPrefix(token, "setting.")
		if ctx.Settings == nil {
			return "", nil
		}
		return ctx.Settings[key], nil
	}

	switch token {
	case "time.now":
		return ctx.Now.Format(time.RFC3339), nil
	case "time.year":
		return fmt.Sprintf("%d", ctx.Now.Year()), nil
	case "time.month":
		return fmt.Sprintf("%02d", ctx.Now.Month()), nil
	case "time.day":
		return fmt.Sprintf("%02d", ctx.Now.Day()), nil
	case "time.hour":
		return fmt.Sprintf("%02d", ctx.Now.Hour()), nil
	case "time.minute":
		return fmt.Sprintf("%02d", ctx.Now.Minute()), nil
	case "time.date":
		return ctx.Now.Format("2006-01-02"), nil
	case "time.unix":
		return fmt.Sprintf("%d", ctx.Now.Unix()), nil
	case "model.name":
		return ctx.ModelName, nil
	case "model.module":
		return ctx.Module, nil
	case "uuid4":
		return uuid.New().String(), nil
	case "uuid7":
		v, err := uuid.NewV7()
		if err != nil {
			return uuid.New().String(), nil
		}
		return v.String(), nil
	}

	if m := substringRe.FindStringSubmatch(token); m != nil {
		src, err := e.resolveSource(m[1], ctx)
		if err != nil {
			return "", err
		}
		start, _ := strconv.Atoi(m[2])
		length, _ := strconv.Atoi(m[3])
		if start >= len(src) {
			return "", nil
		}
		end := start + length
		if end > len(src) {
			end = len(src)
		}
		return src[start:end], nil
	}

	if m := upperRe.FindStringSubmatch(token); m != nil {
		src, err := e.resolveSource(m[1], ctx)
		if err != nil {
			return "", err
		}
		return strings.ToUpper(src), nil
	}

	if m := lowerRe.FindStringSubmatch(token); m != nil {
		src, err := e.resolveSource(m[1], ctx)
		if err != nil {
			return "", err
		}
		return strings.ToLower(src), nil
	}

	if m := hashRe.FindStringSubmatch(token); m != nil {
		src, err := e.resolveSource(m[1], ctx)
		if err != nil {
			return "", err
		}
		h := sha256.Sum256([]byte(src))
		return fmt.Sprintf("%x", h[:4]), nil
	}

	if m := randomRe.FindStringSubmatch(token); m != nil {
		n, _ := strconv.Atoi(m[1])
		b := make([]byte, n)
		for i := range b {
			b[i] = alphanumeric[rand.Intn(len(alphanumeric))]
		}
		return string(b), nil
	}

	if m := randomFixedRe.FindStringSubmatch(token); m != nil {
		length, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		max, _ := strconv.Atoi(m[3])
		if max <= min {
			return "", fmt.Errorf("random_fixed: max must be greater than min")
		}
		val := min + rand.Intn(max-min+1)
		return fmt.Sprintf("%0*d", length, val), nil
	}

	return "", fmt.Errorf("unknown token: %q", token)
}

func (e *Engine) resolveSource(src string, ctx *FormatContext) (string, error) {
	src = strings.TrimSpace(src)
	if strings.HasPrefix(src, "data.") || strings.HasPrefix(src, "session.") || strings.HasPrefix(src, "setting.") {
		return e.resolveToken(src, ctx)
	}
	return src, nil
}
