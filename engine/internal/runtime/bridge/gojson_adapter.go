package bridge

import (
	"encoding/json"
	"fmt"
	"strings"

	gojsonrt "github.com/bitcode-framework/go-json/runtime"
)

// BuildGoJSONExtension creates a go-json Extension from a bridge.Context.
// Maps all 20 bridge namespaces (50+ functions) to go-json callable functions.
func BuildGoJSONExtension(bc *Context) gojsonrt.Extension {
	return gojsonrt.Extension{
		Name: "bitcode",
		Functions: map[string]any{
			"model":      func(name string) any { return buildModelProxy(bc, name, false) },
			"db.query":   buildDBQuery(bc),
			"db.execute": buildDBExecute(bc),

			"http.get":    buildHTTPFunc(bc, "GET"),
			"http.post":   buildHTTPFunc(bc, "POST"),
			"http.put":    buildHTTPFunc(bc, "PUT"),
			"http.patch":  buildHTTPFunc(bc, "PATCH"),
			"http.delete": buildHTTPFunc(bc, "DELETE"),

			"cache.get": func(key string) (any, error) { return bc.Cache().Get(key) },
			"cache.set": buildCacheSet(bc),
			"cache.del": func(key string) (any, error) { return nil, bc.Cache().Del(key) },

			"fs.read":   func(path string) (any, error) { return bc.FS().Read(path) },
			"fs.write":  buildFSWrite(bc),
			"fs.exists": func(path string) (any, error) { return bc.FS().Exists(path) },
			"fs.list":   buildFSList(bc),
			"fs.mkdir":  func(path string) (any, error) { return nil, bc.FS().Mkdir(path) },
			"fs.remove": func(path string) (any, error) { return nil, bc.FS().Remove(path) },

			"env":     func(key string) (any, error) { return bc.Env(key) },
			"config":  func(key string) any { return bc.Config(key) },
			"session": buildSessionFunc(bc),
			"log":     buildLogFunc(bc),
			"emit":    buildEmitFunc(bc),
			"call":    buildCallFunc(bc),
			"exec":    buildExecFunc(bc),

			"email.send":      func(opts map[string]any) (any, error) { return nil, bc.Email().Send(mapToEmailOptions(opts)) },
			"notify.send":     func(opts map[string]any) (any, error) { return nil, bc.Notify().Send(mapToNotifyOptions(opts)) },
			"notify.broadcast": buildNotifyBroadcast(bc),

			"storage.upload":   func(opts map[string]any) (any, error) { r, e := bc.Storage().Upload(mapToUploadOptions(opts)); return convertToAny(r), e },
			"storage.url":      func(id string) (any, error) { return bc.Storage().URL(id) },
			"storage.download": func(id string) (any, error) { return bc.Storage().Download(id) },
			"storage.delete":   func(id string) (any, error) { return nil, bc.Storage().Delete(id) },

			"t": func(key string) any { return bc.T(key) },

			"security.permissions": func(model string) (any, error) { r, e := bc.Security().Permissions(model); return convertToAny(r), e },
			"security.hasGroup":    func(group string) (any, error) { return bc.Security().HasGroup(group) },
			"security.groups":      buildSecurityGroups(bc),

			"audit.log": func(opts map[string]any) (any, error) { return nil, bc.Audit().Log(mapToAuditOptions(opts)) },

			"crypto.encrypt": func(plaintext string) (any, error) { return bc.Crypto().Encrypt(plaintext) },
			"crypto.decrypt": func(ciphertext string) (any, error) { return bc.Crypto().Decrypt(ciphertext) },
			"crypto.hash":    func(value string) (any, error) { return bc.Crypto().Hash(value) },
			"crypto.verify":  func(value, hash string) (any, error) { return bc.Crypto().Verify(value, hash) },

			"execution.current": func() any { return convertToAny(bc.Execution().Current()) },
			"execution.search":  func(opts map[string]any) (any, error) { r, e := bc.Execution().Search(mapToExecutionSearchOptions(opts)); return convertToAny(r), e },
			"execution.get":     func(id string) (any, error) { r, e := bc.Execution().Get(id); return convertToAny(r), e },
			"execution.retry":   func(id string) (any, error) { r, e := bc.Execution().Retry(id); return convertToAny(r), e },
			"execution.cancel":  func(id string) (any, error) { return nil, bc.Execution().Cancel(id) },

			"tx": func(fn func() error) (any, error) { return nil, bc.Tx(func(tx *Context) error { return fn() }) },
		},
	}
}

func buildDBQuery(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("db.query: sql is required")
		}
		sqlStr, _ := params[0].(string)
		var args []any
		if len(params) > 1 {
			if a, ok := params[1].([]any); ok {
				args = a
			}
		}
		rows, err := bc.DB().Query(sqlStr, args...)
		if err != nil {
			return nil, err
		}
		return convertToAny(rows), nil
	}
}

func buildDBExecute(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("db.execute: sql is required")
		}
		sqlStr, _ := params[0].(string)
		var args []any
		if len(params) > 1 {
			if a, ok := params[1].([]any); ok {
				args = a
			}
		}
		result, err := bc.DB().Execute(sqlStr, args...)
		if err != nil {
			return nil, err
		}
		return map[string]any{"rows_affected": result.RowsAffected}, nil
	}
}

func buildHTTPFunc(bc *Context, method string) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("http.%s: url required", strings.ToLower(method))
		}
		url, _ := params[0].(string)
		var opts *HTTPOptions
		if len(params) > 1 {
			if o, ok := params[1].(map[string]any); ok {
				opts = mapToHTTPOptions(o)
			}
		}
		var resp *HTTPResponse
		var err error
		switch method {
		case "GET":
			resp, err = bc.HTTP().Get(url, opts)
		case "POST":
			resp, err = bc.HTTP().Post(url, opts)
		case "PUT":
			resp, err = bc.HTTP().Put(url, opts)
		case "PATCH":
			resp, err = bc.HTTP().Patch(url, opts)
		case "DELETE":
			resp, err = bc.HTTP().Delete(url, opts)
		}
		if err != nil {
			return nil, err
		}
		return convertToAny(resp), nil
	}
}

func buildCacheSet(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("cache.set: key and value required")
		}
		key, _ := params[0].(string)
		val := params[1]
		var opts *CacheSetOptions
		if len(params) > 2 {
			if o, ok := params[2].(map[string]any); ok {
				opts = &CacheSetOptions{}
				if ttl, ok := o["ttl"].(float64); ok {
					opts.TTL = int(ttl)
				}
			}
		}
		return nil, bc.Cache().Set(key, val, opts)
	}
}

func buildFSWrite(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("fs.write: path and content required")
		}
		path, _ := params[0].(string)
		content, _ := params[1].(string)
		return nil, bc.FS().Write(path, content)
	}
}

func buildFSList(bc *Context) func(path string) (any, error) {
	return func(path string) (any, error) {
		items, err := bc.FS().List(path)
		if err != nil {
			return nil, err
		}
		result := make([]any, len(items))
		for i, item := range items {
			result[i] = item
		}
		return result, nil
	}
}

func buildSessionFunc(bc *Context) func() any {
	return func() any {
		s := bc.Session()
		return map[string]any{
			"userId": s.UserID, "username": s.Username, "email": s.Email,
			"tenantId": s.TenantID, "groups": s.Groups, "locale": s.Locale, "context": s.Context,
		}
	}
}

func buildLogFunc(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("log: level and msg required")
		}
		level, _ := params[0].(string)
		msg, _ := params[1].(string)
		var data []map[string]any
		for i := 2; i < len(params); i++ {
			if d, ok := params[i].(map[string]any); ok {
				data = append(data, d)
			}
		}
		bc.Log(level, msg, data...)
		return nil, nil
	}
}

func buildEmitFunc(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("emit: event name required")
		}
		event, _ := params[0].(string)
		var data map[string]any
		if len(params) > 1 {
			data, _ = params[1].(map[string]any)
		}
		return nil, bc.Emit(event, data)
	}
}

func buildCallFunc(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("call: process name required")
		}
		process, _ := params[0].(string)
		var input map[string]any
		if len(params) > 1 {
			input, _ = params[1].(map[string]any)
		}
		return bc.Call(process, input)
	}
}

func buildExecFunc(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("exec: cmd required")
		}
		cmd, _ := params[0].(string)
		var args []string
		if len(params) > 1 {
			if a, ok := params[1].([]any); ok {
				for _, v := range a {
					args = append(args, fmt.Sprintf("%v", v))
				}
			}
		}
		var opts *ExecOptions
		if len(params) > 2 {
			if o, ok := params[2].(map[string]any); ok {
				opts = &ExecOptions{}
				if cwd, ok := o["cwd"].(string); ok {
					opts.Cwd = cwd
				}
				if timeout, ok := o["timeout"].(float64); ok {
					opts.Timeout = int(timeout)
				}
			}
		}
		result, err := bc.Exec(cmd, args, opts)
		if err != nil {
			return nil, err
		}
		return map[string]any{"exit_code": result.ExitCode, "stdout": result.Stdout, "stderr": result.Stderr}, nil
	}
}

func buildNotifyBroadcast(bc *Context) func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("notify.broadcast: channel and data required")
		}
		channel, _ := params[0].(string)
		data, _ := params[1].(map[string]any)
		return nil, bc.Notify().Broadcast(channel, data)
	}
}

func buildSecurityGroups(bc *Context) func() (any, error) {
	return func() (any, error) {
		groups, err := bc.Security().Groups()
		if err != nil {
			return nil, err
		}
		result := make([]any, len(groups))
		for i, g := range groups {
			result[i] = g
		}
		return result, nil
	}
}

func buildModelProxy(bc *Context, name string, sudo bool) map[string]any {
	handle := bc.model.Model(name, bc.session, sudo)
	return map[string]any{
		"search": func(opts ...any) (any, error) {
			so := SearchOptions{}
			if len(opts) > 0 {
				if m, ok := opts[0].(map[string]any); ok {
					so = mapToSearchOptions(m)
				}
			}
			r, e := handle.Search(so)
			return convertToAny(r), e
		},
		"get": func(id any) (any, error) {
			r, e := handle.Get(fmt.Sprintf("%v", id))
			return convertToAny(r), e
		},
		"create": func(data map[string]any) (any, error) {
			r, e := handle.Create(data)
			return convertToAny(r), e
		},
		"write": func(params ...any) (any, error) {
			if len(params) < 2 {
				return nil, fmt.Errorf("model.write: id and data required")
			}
			data, _ := params[1].(map[string]any)
			return nil, handle.Write(fmt.Sprintf("%v", params[0]), data)
		},
		"delete":    func(id any) (any, error) { return nil, handle.Delete(fmt.Sprintf("%v", id)) },
		"count":     func(opts ...any) (any, error) { so := SearchOptions{}; if len(opts) > 0 { if m, ok := opts[0].(map[string]any); ok { so = mapToSearchOptions(m) } }; return handle.Count(so) },
		"sum":       func(params ...any) (any, error) { if len(params) < 1 { return nil, fmt.Errorf("model.sum: field required") }; field, _ := params[0].(string); so := SearchOptions{}; if len(params) > 1 { if m, ok := params[1].(map[string]any); ok { so = mapToSearchOptions(m) } }; return handle.Sum(field, so) },
		"upsert":    func(params ...any) (any, error) { if len(params) < 2 { return nil, fmt.Errorf("model.upsert: data and unique required") }; data, _ := params[0].(map[string]any); r, e := handle.Upsert(data, toStringSlice(params[1])); return convertToAny(r), e },
		"createMany": func(records []any) (any, error) { r, e := handle.CreateMany(toMapSlice(records)); return convertToAny(r), e },
		"writeMany": func(params ...any) (any, error) { if len(params) < 2 { return nil, fmt.Errorf("model.writeMany: ids and data required") }; data, _ := params[1].(map[string]any); r, e := handle.WriteMany(toStringSlice(params[0]), data); return convertToAny(r), e },
		"deleteMany": func(ids any) (any, error) { r, e := handle.DeleteMany(toStringSlice(ids)); return convertToAny(r), e },
		"upsertMany": func(params ...any) (any, error) { if len(params) < 2 { return nil, fmt.Errorf("model.upsertMany: records and unique required") }; records, _ := params[0].([]any); r, e := handle.UpsertMany(toMapSlice(records), toStringSlice(params[1])); return convertToAny(r), e },
		"addRelation":    func(params ...any) (any, error) { if len(params) < 3 { return nil, fmt.Errorf("model.addRelation: id, field, ids required") }; return nil, handle.AddRelation(fmt.Sprintf("%v", params[0]), fmt.Sprintf("%v", params[1]), toStringSlice(params[2])) },
		"removeRelation": func(params ...any) (any, error) { if len(params) < 3 { return nil, fmt.Errorf("model.removeRelation: id, field, ids required") }; return nil, handle.RemoveRelation(fmt.Sprintf("%v", params[0]), fmt.Sprintf("%v", params[1]), toStringSlice(params[2])) },
		"setRelation":    func(params ...any) (any, error) { if len(params) < 3 { return nil, fmt.Errorf("model.setRelation: id, field, ids required") }; return nil, handle.SetRelation(fmt.Sprintf("%v", params[0]), fmt.Sprintf("%v", params[1]), toStringSlice(params[2])) },
		"loadRelation":   func(params ...any) (any, error) { if len(params) < 2 { return nil, fmt.Errorf("model.loadRelation: id and field required") }; r, e := handle.LoadRelation(fmt.Sprintf("%v", params[0]), fmt.Sprintf("%v", params[1])); return convertToAny(r), e },
		"sudo":           func() any { return buildModelProxy(bc, name, true) },
	}
}

func convertToAny(v any) any {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return v
	}
	return result
}

func mapToSearchOptions(m map[string]any) SearchOptions {
	so := SearchOptions{}
	if domain, ok := m["domain"].([]any); ok {
		for _, d := range domain {
			if arr, ok := d.([]any); ok {
				so.Domain = append(so.Domain, arr)
			}
		}
	}
	if fields, ok := m["fields"].([]any); ok {
		for _, f := range fields {
			so.Fields = append(so.Fields, fmt.Sprintf("%v", f))
		}
	}
	if order, ok := m["order"].(string); ok { so.Order = order }
	if limit, ok := m["limit"].(float64); ok { so.Limit = int(limit) }
	if offset, ok := m["offset"].(float64); ok { so.Offset = int(offset) }
	if include, ok := m["include"].([]any); ok {
		for _, inc := range include {
			so.Include = append(so.Include, fmt.Sprintf("%v", inc))
		}
	}
	return so
}

func mapToHTTPOptions(m map[string]any) *HTTPOptions {
	opts := &HTTPOptions{}
	if headers, ok := m["headers"].(map[string]any); ok {
		opts.Headers = make(map[string]string)
		for k, v := range headers { opts.Headers[k] = fmt.Sprintf("%v", v) }
	}
	if body, ok := m["body"]; ok { opts.Body = body }
	if timeout, ok := m["timeout"].(float64); ok { opts.Timeout = int(timeout) }
	return opts
}

func mapToEmailOptions(m map[string]any) EmailOptions {
	eo := EmailOptions{}
	if to, ok := m["to"].(string); ok { eo.To = to }
	if subject, ok := m["subject"].(string); ok { eo.Subject = subject }
	if body, ok := m["body"].(string); ok { eo.Body = body }
	if template, ok := m["template"].(string); ok { eo.Template = template }
	if data, ok := m["data"].(map[string]any); ok { eo.Data = data }
	return eo
}

func mapToNotifyOptions(m map[string]any) NotifyOptions {
	no := NotifyOptions{}
	if to, ok := m["to"].(string); ok { no.To = to }
	if title, ok := m["title"].(string); ok { no.Title = title }
	if msg, ok := m["message"].(string); ok { no.Message = msg }
	if t, ok := m["type"].(string); ok { no.Type = t }
	return no
}

func mapToUploadOptions(m map[string]any) UploadOptions {
	uo := UploadOptions{}
	if filename, ok := m["filename"].(string); ok { uo.Filename = filename }
	if content, ok := m["content"].(string); ok { uo.Content = []byte(content) }
	if model, ok := m["model"].(string); ok { uo.Model = model }
	if recordID, ok := m["recordId"].(string); ok { uo.RecordID = recordID }
	return uo
}

func mapToAuditOptions(m map[string]any) AuditOptions {
	ao := AuditOptions{}
	if action, ok := m["action"].(string); ok { ao.Action = action }
	if model, ok := m["model"].(string); ok { ao.Model = model }
	if recordID, ok := m["recordId"].(string); ok { ao.RecordID = recordID }
	if detail, ok := m["detail"].(string); ok { ao.Detail = detail }
	return ao
}

func mapToExecutionSearchOptions(m map[string]any) ExecutionSearchOptions {
	eso := ExecutionSearchOptions{}
	if process, ok := m["process"].(string); ok { eso.Process = process }
	if status, ok := m["status"].(string); ok { eso.Status = status }
	if userID, ok := m["userId"].(string); ok { eso.UserID = userID }
	if limit, ok := m["limit"].(float64); ok { eso.Limit = int(limit) }
	if offset, ok := m["offset"].(float64); ok { eso.Offset = int(offset) }
	if order, ok := m["order"].(string); ok { eso.Order = order }
	return eso
}

func toStringSlice(v any) []string {
	if arr, ok := v.([]any); ok {
		result := make([]string, len(arr))
		for i, item := range arr { result[i] = fmt.Sprintf("%v", item) }
		return result
	}
	if arr, ok := v.([]string); ok { return arr }
	return nil
}

func toMapSlice(records []any) []map[string]any {
	var maps []map[string]any
	for _, r := range records {
		if m, ok := r.(map[string]any); ok { maps = append(maps, m) }
	}
	return maps
}
