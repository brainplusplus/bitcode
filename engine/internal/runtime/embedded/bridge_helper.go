package embedded

import (
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

func ParseSearchOpts(raw map[string]any) bridge.SearchOptions {
	opts := bridge.SearchOptions{}
	if raw == nil {
		return opts
	}
	if domain, ok := raw["domain"]; ok {
		opts.Domain = toAnySliceSlice(domain)
	}
	if fields, ok := raw["fields"]; ok {
		opts.Fields = ToStringSlice(fields)
	}
	if order, ok := raw["order"]; ok {
		opts.Order = fmt.Sprintf("%v", order)
	}
	if limit, ok := raw["limit"]; ok {
		opts.Limit = ToInt(limit)
	}
	if offset, ok := raw["offset"]; ok {
		opts.Offset = ToInt(offset)
	}
	if include, ok := raw["include"]; ok {
		opts.Include = ToStringSlice(include)
	}
	return opts
}

func ParseHTTPOpts(raw map[string]any) *bridge.HTTPOptions {
	if raw == nil {
		return nil
	}
	opts := &bridge.HTTPOptions{}
	if headers, ok := raw["headers"].(map[string]any); ok {
		opts.Headers = make(map[string]string)
		for k, v := range headers {
			opts.Headers[k] = fmt.Sprintf("%v", v)
		}
	}
	if body, ok := raw["body"]; ok {
		opts.Body = body
	}
	if timeout, ok := raw["timeout"]; ok {
		opts.Timeout = ToInt(timeout)
	}
	if profile, ok := raw["profile"].(string); ok {
		opts.Profile = profile
	}
	if proxy, ok := raw["proxy"].(string); ok {
		opts.Proxy = proxy
	}
	if cookieJar, ok := raw["cookieJar"].(string); ok {
		opts.CookieJar = cookieJar
	}
	if followRedirects, ok := raw["followRedirects"].(bool); ok {
		opts.FollowRedirects = &followRedirects
	}
	if insecure, ok := raw["insecureSkipVerify"].(bool); ok {
		opts.InsecureSkipVerify = insecure
	}
	if headerOrder, ok := raw["headerOrder"]; ok {
		opts.HeaderOrder = ToStringSlice(headerOrder)
	}
	return opts
}

func ParseCacheOpts(raw map[string]any) *bridge.CacheSetOptions {
	if raw == nil {
		return nil
	}
	opts := &bridge.CacheSetOptions{}
	if ttl, ok := raw["ttl"]; ok {
		opts.TTL = ToInt(ttl)
	}
	return opts
}

func ParseExecOpts(raw map[string]any) *bridge.ExecOptions {
	if raw == nil {
		return nil
	}
	opts := &bridge.ExecOptions{}
	if cwd, ok := raw["cwd"].(string); ok {
		opts.Cwd = cwd
	}
	if timeout, ok := raw["timeout"]; ok {
		opts.Timeout = ToInt(timeout)
	}
	return opts
}

func ParseEmailOpts(raw map[string]any) bridge.EmailOptions {
	opts := bridge.EmailOptions{}
	if to, ok := raw["to"].(string); ok {
		opts.To = to
	}
	if subject, ok := raw["subject"].(string); ok {
		opts.Subject = subject
	}
	if body, ok := raw["body"].(string); ok {
		opts.Body = body
	}
	if tmpl, ok := raw["template"].(string); ok {
		opts.Template = tmpl
	}
	if data, ok := raw["data"].(map[string]any); ok {
		opts.Data = data
	}
	return opts
}

func ParseNotifyOpts(raw map[string]any) bridge.NotifyOptions {
	opts := bridge.NotifyOptions{}
	if to, ok := raw["to"].(string); ok {
		opts.To = to
	}
	if title, ok := raw["title"].(string); ok {
		opts.Title = title
	}
	if message, ok := raw["message"].(string); ok {
		opts.Message = message
	}
	if typ, ok := raw["type"].(string); ok {
		opts.Type = typ
	}
	return opts
}

func ParseUploadOpts(raw map[string]any) bridge.UploadOptions {
	opts := bridge.UploadOptions{}
	if filename, ok := raw["filename"].(string); ok {
		opts.Filename = filename
	}
	if content, ok := raw["content"].([]byte); ok {
		opts.Content = content
	}
	if model, ok := raw["model"].(string); ok {
		opts.Model = model
	}
	if recordID, ok := raw["recordId"].(string); ok {
		opts.RecordID = recordID
	}
	return opts
}

func ParseAuditOpts(raw map[string]any) bridge.AuditOptions {
	opts := bridge.AuditOptions{}
	if action, ok := raw["action"].(string); ok {
		opts.Action = action
	}
	if model, ok := raw["model"].(string); ok {
		opts.Model = model
	}
	if recordID, ok := raw["recordId"].(string); ok {
		opts.RecordID = recordID
	}
	if detail, ok := raw["detail"].(string); ok {
		opts.Detail = detail
	}
	return opts
}

func ToInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	default:
		return 0
	}
}

func ToStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return nil
	}
}

func toAnySliceSlice(v any) [][]any {
	switch val := v.(type) {
	case [][]any:
		return val
	case []any:
		result := make([][]any, len(val))
		for i, item := range val {
			if inner, ok := item.([]any); ok {
				result[i] = inner
			}
		}
		return result
	default:
		return nil
	}
}
