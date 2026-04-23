package storage

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	domainstorage "github.com/bitcode-engine/engine/internal/domain/storage"
	"github.com/google/uuid"
)

type FormatContext struct {
	TenantID string
	UserID   string
	Model    string
	Original string
	Ext      string
	Input    map[string]string
	Data     map[string]string
}

var templateVarRegex = regexp.MustCompile(`\{([^}]+)\}`)

func FormatPath(template string, ctx FormatContext) string {
	return resolveTemplate(template, ctx)
}

func FormatName(template string, ctx FormatContext) string {
	return resolveTemplate(template, ctx)
}

func resolveTemplate(tmpl string, ctx FormatContext) string {
	now := time.Now()
	uid := uuid.New().String()

	result := templateVarRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		key := match[1 : len(match)-1]

		switch key {
		case "tenant_id":
			return sanitize(ctx.TenantID)
		case "user_id":
			return sanitize(ctx.UserID)
		case "model":
			return sanitize(ctx.Model)
		case "year":
			return now.Format("2006")
		case "month":
			return now.Format("01")
		case "day":
			return now.Format("02")
		case "date":
			return now.Format("2006-01-02")
		case "timestamp":
			return now.Format("20060102_150405")
		case "uuid":
			return uid
		case "original":
			return sanitize(ctx.Original)
		case "ext":
			return ctx.Ext
		}

		if strings.HasPrefix(key, "input.") {
			field := key[6:]
			if val, ok := ctx.Input[field]; ok {
				return sanitize(val)
			}
			return ""
		}

		if strings.HasPrefix(key, "data.") {
			field := key[5:]
			if val, ok := ctx.Data[field]; ok {
				return sanitize(val)
			}
			return ""
		}

		return ""
	})

	return result
}

func sanitize(s string) string {
	if s == "" {
		return ""
	}
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, s)
	s = strings.Trim(s, "_.")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, name)
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}

func OriginalWithoutExt(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx <= 0 {
		return filename
	}
	return filename[:idx]
}

func NewStorageDriver(cfg StorageConfig) (domainstorage.StorageDriver, error) {
	switch cfg.Driver {
	case "s3":
		return NewS3Storage(cfg.S3)
	case "local", "":
		return NewLocalStorage(cfg.Local), nil
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s (use local or s3)", cfg.Driver)
	}
}
