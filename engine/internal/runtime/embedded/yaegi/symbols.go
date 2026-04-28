package yaegi_runtime

import (
	"reflect"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/traefik/yaegi/interp"
)

// BuildBridgeSymbols creates the interp.Exports map that exposes bridge.Context
// as the "bitcode" Go package. Scripts import it via: import "bitcode"
func BuildBridgeSymbols(bc *bridge.Context) interp.Exports {
	return interp.Exports{
		"bitcode/bitcode": buildBitcodePackage(bc),
	}
}

func buildBitcodePackage(bc *bridge.Context) map[string]reflect.Value {
	return map[string]reflect.Value{
		"Model":   reflect.ValueOf(func(name string) *goModelProxy { return newModelProxy(bc, name) }),
		"Session": reflect.ValueOf(bc.Session),
		"DB":      reflect.ValueOf(newDBProxy(bc)),
		"HTTP":    reflect.ValueOf(func() *goHTTPProxy { return newHTTPProxy(bc) }),
		"Cache":   reflect.ValueOf(func() *goCacheProxy { return newCacheProxy(bc) }),
		"FS":      reflect.ValueOf(func() *goFSProxy { return newFSProxy(bc) }),
		"Env":     reflect.ValueOf(func(key string) (string, error) { return bc.Env(key) }),
		"Config":  reflect.ValueOf(func(key string) any { return bc.Config(key) }),
		"Log":     reflect.ValueOf(func(level, msg string, data ...map[string]any) { bc.Log(level, msg, data...) }),
		"Emit":    reflect.ValueOf(func(event string, data map[string]any) error { return bc.Emit(event, data) }),
		"Call":    reflect.ValueOf(func(process string, input map[string]any) (any, error) { return bc.Call(process, input) }),
		"T":       reflect.ValueOf(func(key string) string { return bc.T(key) }),
		"Exec": reflect.ValueOf(func(cmd string, args []string, opts ...map[string]any) (*bridge.ExecResult, error) {
			return bc.Exec(cmd, args, embedded.ParseExecOpts(firstMap(opts)))
		}),
		"Email":     reflect.ValueOf(func() *goEmailProxy { return newEmailProxy(bc) }),
		"Notify":    reflect.ValueOf(func() *goNotifyProxy { return newNotifyProxy(bc) }),
		"Storage":   reflect.ValueOf(func() *goStorageProxy { return newStorageProxy(bc) }),
		"Security":  reflect.ValueOf(func() *goSecurityProxy { return newSecurityProxy(bc) }),
		"Audit":     reflect.ValueOf(func() *goAuditProxy { return newAuditProxy(bc) }),
		"Crypto":    reflect.ValueOf(func() *goCryptoProxy { return newCryptoProxy(bc) }),
		"Execution": reflect.ValueOf(func() *goExecutionProxy { return newExecutionProxy(bc) }),
		"Tx": reflect.ValueOf(func(fn func() error) error {
			return bc.Tx(func(_ *bridge.Context) error {
				return fn()
			})
		}),
	}
}

// --- Model proxy (PascalCase for Go convention) ---

type goModelProxy struct {
	handle bridge.ModelHandle
}

func newModelProxy(bc *bridge.Context, name string) *goModelProxy {
	return &goModelProxy{handle: bc.Model(name)}
}

func (m *goModelProxy) Search(opts ...map[string]any) ([]map[string]any, error) {
	return m.handle.Search(embedded.ParseSearchOpts(firstMap(opts)))
}
func (m *goModelProxy) Get(id string) (map[string]any, error) { return m.handle.Get(id) }
func (m *goModelProxy) Create(data map[string]any) (map[string]any, error) {
	return m.handle.Create(data)
}
func (m *goModelProxy) Write(id string, data map[string]any) error { return m.handle.Write(id, data) }
func (m *goModelProxy) Delete(id string) error                     { return m.handle.Delete(id) }
func (m *goModelProxy) Count(opts ...map[string]any) (int64, error) {
	return m.handle.Count(embedded.ParseSearchOpts(firstMap(opts)))
}
func (m *goModelProxy) Sum(field string, opts ...map[string]any) (float64, error) {
	return m.handle.Sum(field, embedded.ParseSearchOpts(firstMap(opts)))
}
func (m *goModelProxy) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	return m.handle.Upsert(data, uniqueFields)
}
func (m *goModelProxy) CreateMany(records []map[string]any) ([]map[string]any, error) {
	return m.handle.CreateMany(records)
}
func (m *goModelProxy) WriteMany(ids []string, data map[string]any) (*bridge.BulkResult, error) {
	return m.handle.WriteMany(ids, data)
}
func (m *goModelProxy) DeleteMany(ids []string) (*bridge.BulkResult, error) {
	return m.handle.DeleteMany(ids)
}
func (m *goModelProxy) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	return m.handle.UpsertMany(records, uniqueFields)
}
func (m *goModelProxy) AddRelation(id, field string, relatedIDs []string) error {
	return m.handle.AddRelation(id, field, relatedIDs)
}
func (m *goModelProxy) RemoveRelation(id, field string, relatedIDs []string) error {
	return m.handle.RemoveRelation(id, field, relatedIDs)
}
func (m *goModelProxy) SetRelation(id, field string, relatedIDs []string) error {
	return m.handle.SetRelation(id, field, relatedIDs)
}
func (m *goModelProxy) LoadRelation(id, field string) ([]map[string]any, error) {
	return m.handle.LoadRelation(id, field)
}
func (m *goModelProxy) Sudo() *goSudoModelProxy {
	return &goSudoModelProxy{handle: m.handle.Sudo()}
}

// --- Sudo model proxy ---

type goSudoModelProxy struct {
	handle bridge.SudoModelHandle
}

func (s *goSudoModelProxy) Search(opts ...map[string]any) ([]map[string]any, error) {
	return s.handle.Search(embedded.ParseSearchOpts(firstMap(opts)))
}
func (s *goSudoModelProxy) Get(id string) (map[string]any, error) { return s.handle.Get(id) }
func (s *goSudoModelProxy) Create(data map[string]any) (map[string]any, error) {
	return s.handle.Create(data)
}
func (s *goSudoModelProxy) Write(id string, data map[string]any) error {
	return s.handle.Write(id, data)
}
func (s *goSudoModelProxy) Delete(id string) error { return s.handle.Delete(id) }
func (s *goSudoModelProxy) Count(opts ...map[string]any) (int64, error) {
	return s.handle.Count(embedded.ParseSearchOpts(firstMap(opts)))
}
func (s *goSudoModelProxy) Sum(field string, opts ...map[string]any) (float64, error) {
	return s.handle.Sum(field, embedded.ParseSearchOpts(firstMap(opts)))
}
func (s *goSudoModelProxy) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	return s.handle.Upsert(data, uniqueFields)
}
func (s *goSudoModelProxy) CreateMany(records []map[string]any) ([]map[string]any, error) {
	return s.handle.CreateMany(records)
}
func (s *goSudoModelProxy) WriteMany(ids []string, data map[string]any) (*bridge.BulkResult, error) {
	return s.handle.WriteMany(ids, data)
}
func (s *goSudoModelProxy) DeleteMany(ids []string) (*bridge.BulkResult, error) {
	return s.handle.DeleteMany(ids)
}
func (s *goSudoModelProxy) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	return s.handle.UpsertMany(records, uniqueFields)
}
func (s *goSudoModelProxy) AddRelation(id, field string, relatedIDs []string) error {
	return s.handle.AddRelation(id, field, relatedIDs)
}
func (s *goSudoModelProxy) RemoveRelation(id, field string, relatedIDs []string) error {
	return s.handle.RemoveRelation(id, field, relatedIDs)
}
func (s *goSudoModelProxy) SetRelation(id, field string, relatedIDs []string) error {
	return s.handle.SetRelation(id, field, relatedIDs)
}
func (s *goSudoModelProxy) LoadRelation(id, field string) ([]map[string]any, error) {
	return s.handle.LoadRelation(id, field)
}
func (s *goSudoModelProxy) HardDelete(id string) error { return s.handle.HardDelete(id) }
func (s *goSudoModelProxy) HardDeleteMany(ids []string) (*bridge.BulkResult, error) {
	return s.handle.HardDeleteMany(ids)
}
func (s *goSudoModelProxy) WithTenant(tenantID string) *goSudoModelProxy {
	return &goSudoModelProxy{handle: s.handle.WithTenant(tenantID)}
}
func (s *goSudoModelProxy) SkipValidation() *goSudoModelProxy {
	return &goSudoModelProxy{handle: s.handle.SkipValidation()}
}

// --- DB proxy ---

type goDBProxy struct {
	db bridge.DB
}

func newDBProxy(bc *bridge.Context) *goDBProxy { return &goDBProxy{db: bc.DB()} }
func (d *goDBProxy) Query(sql string, args ...any) ([]map[string]any, error) {
	return d.db.Query(sql, args...)
}
func (d *goDBProxy) Execute(sql string, args ...any) (*bridge.ExecDBResult, error) {
	return d.db.Execute(sql, args...)
}

// --- HTTP proxy ---

type goHTTPProxy struct {
	http bridge.HTTPClient
}

func newHTTPProxy(bc *bridge.Context) *goHTTPProxy { return &goHTTPProxy{http: bc.HTTP()} }
func (h *goHTTPProxy) Get(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return h.http.Get(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (h *goHTTPProxy) Post(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return h.http.Post(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (h *goHTTPProxy) Put(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return h.http.Put(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (h *goHTTPProxy) Patch(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return h.http.Patch(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (h *goHTTPProxy) Delete(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return h.http.Delete(url, embedded.ParseHTTPOpts(firstMap(opts)))
}

// --- Cache proxy ---

type goCacheProxy struct {
	cache bridge.Cache
}

func newCacheProxy(bc *bridge.Context) *goCacheProxy { return &goCacheProxy{cache: bc.Cache()} }
func (c *goCacheProxy) Get(key string) (any, error)  { return c.cache.Get(key) }
func (c *goCacheProxy) Set(key string, val any, opts ...map[string]any) error {
	return c.cache.Set(key, val, embedded.ParseCacheOpts(firstMap(opts)))
}
func (c *goCacheProxy) Del(key string) error { return c.cache.Del(key) }

// --- FS proxy ---

type goFSProxy struct {
	fs bridge.FS
}

func newFSProxy(bc *bridge.Context) *goFSProxy       { return &goFSProxy{fs: bc.FS()} }
func (f *goFSProxy) Read(path string) (string, error) { return f.fs.Read(path) }
func (f *goFSProxy) Write(path, content string) error  { return f.fs.Write(path, content) }
func (f *goFSProxy) Exists(path string) (bool, error)  { return f.fs.Exists(path) }
func (f *goFSProxy) List(path string) ([]string, error) { return f.fs.List(path) }
func (f *goFSProxy) Mkdir(path string) error            { return f.fs.Mkdir(path) }
func (f *goFSProxy) Remove(path string) error           { return f.fs.Remove(path) }

// --- Email proxy ---

type goEmailProxy struct {
	email bridge.EmailSender
}

func newEmailProxy(bc *bridge.Context) *goEmailProxy { return &goEmailProxy{email: bc.Email()} }
func (e *goEmailProxy) Send(opts map[string]any) error {
	return e.email.Send(embedded.ParseEmailOpts(opts))
}

// --- Notify proxy ---

type goNotifyProxy struct {
	notify bridge.Notifier
}

func newNotifyProxy(bc *bridge.Context) *goNotifyProxy { return &goNotifyProxy{notify: bc.Notify()} }
func (n *goNotifyProxy) Send(opts map[string]any) error {
	return n.notify.Send(embedded.ParseNotifyOpts(opts))
}
func (n *goNotifyProxy) Broadcast(channel string, data map[string]any) error {
	return n.notify.Broadcast(channel, data)
}

// --- Storage proxy ---

type goStorageProxy struct {
	storage bridge.Storage
}

func newStorageProxy(bc *bridge.Context) *goStorageProxy {
	return &goStorageProxy{storage: bc.Storage()}
}
func (s *goStorageProxy) Upload(opts map[string]any) (*bridge.Attachment, error) {
	return s.storage.Upload(embedded.ParseUploadOpts(opts))
}
func (s *goStorageProxy) URL(id string) (string, error)      { return s.storage.URL(id) }
func (s *goStorageProxy) Download(id string) ([]byte, error) { return s.storage.Download(id) }
func (s *goStorageProxy) Delete(id string) error             { return s.storage.Delete(id) }

// --- Security proxy ---

type goSecurityProxy struct {
	security bridge.SecurityChecker
}

func newSecurityProxy(bc *bridge.Context) *goSecurityProxy {
	return &goSecurityProxy{security: bc.Security()}
}
func (s *goSecurityProxy) Permissions(model string) (*bridge.ModelPermissions, error) {
	return s.security.Permissions(model)
}
func (s *goSecurityProxy) HasGroup(group string) (bool, error) { return s.security.HasGroup(group) }
func (s *goSecurityProxy) Groups() ([]string, error)           { return s.security.Groups() }

// --- Audit proxy ---

type goAuditProxy struct {
	audit bridge.AuditLogger
}

func newAuditProxy(bc *bridge.Context) *goAuditProxy { return &goAuditProxy{audit: bc.Audit()} }
func (a *goAuditProxy) Log(opts map[string]any) error {
	return a.audit.Log(embedded.ParseAuditOpts(opts))
}

// --- Crypto proxy ---

type goCryptoProxy struct {
	crypto bridge.Crypto
}

func newCryptoProxy(bc *bridge.Context) *goCryptoProxy { return &goCryptoProxy{crypto: bc.Crypto()} }
func (c *goCryptoProxy) Encrypt(plaintext string) (string, error) {
	return c.crypto.Encrypt(plaintext)
}
func (c *goCryptoProxy) Decrypt(ciphertext string) (string, error) {
	return c.crypto.Decrypt(ciphertext)
}
func (c *goCryptoProxy) Hash(value string) (string, error) { return c.crypto.Hash(value) }
func (c *goCryptoProxy) Verify(value, hash string) (bool, error) {
	return c.crypto.Verify(value, hash)
}

// --- Execution proxy ---

type goExecutionProxy struct {
	exec bridge.ExecutionLog
}

func newExecutionProxy(bc *bridge.Context) *goExecutionProxy {
	return &goExecutionProxy{exec: bc.Execution()}
}
func (e *goExecutionProxy) Search(opts map[string]any) ([]map[string]any, error) {
	return e.exec.Search(parseExecSearchOpts(opts))
}
func (e *goExecutionProxy) Get(id string) (map[string]any, error) { return e.exec.Get(id) }
func (e *goExecutionProxy) Current() *bridge.ExecutionInfo         { return e.exec.Current() }
func (e *goExecutionProxy) Cancel(id string) error                 { return e.exec.Cancel(id) }

// --- helpers ---

func parseExecSearchOpts(raw map[string]any) bridge.ExecutionSearchOptions {
	opts := bridge.ExecutionSearchOptions{}
	if raw == nil {
		return opts
	}
	if process, ok := raw["process"].(string); ok {
		opts.Process = process
	}
	if status, ok := raw["status"].(string); ok {
		opts.Status = status
	}
	if userID, ok := raw["userId"].(string); ok {
		opts.UserID = userID
	}
	if limit, ok := raw["limit"]; ok {
		opts.Limit = embedded.ToInt(limit)
	}
	if offset, ok := raw["offset"]; ok {
		opts.Offset = embedded.ToInt(offset)
	}
	if order, ok := raw["order"].(string); ok {
		opts.Order = order
	}
	return opts
}

func firstMap(opts []map[string]any) map[string]any {
	if len(opts) > 0 {
		return opts[0]
	}
	return nil
}
