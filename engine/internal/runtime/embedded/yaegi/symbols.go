package yaegi_runtime

import (
	"reflect"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/traefik/yaegi/interp"
)

// bridgeHolder holds a swappable pointer to bridge.Context.
// All proxy structs reference this holder so that Tx can swap the context
// and all subsequent bridge calls within the transaction use the tx connection.
type bridgeHolder struct {
	ctx *bridge.Context
}

func (h *bridgeHolder) get() *bridge.Context { return h.ctx }

// BuildBridgeSymbols creates the interp.Exports map that exposes bridge.Context
// as the "bitcode" Go package. Scripts import it via: import "bitcode"
// Returns both the exports and the holder (stored on VM for Tx swapping).
func BuildBridgeSymbols(bc *bridge.Context) (interp.Exports, *bridgeHolder) {
	h := &bridgeHolder{ctx: bc}
	return interp.Exports{
		"bitcode/bitcode": buildBitcodePackage(h),
	}, h
}

func buildBitcodePackage(h *bridgeHolder) map[string]reflect.Value {
	return map[string]reflect.Value{
		"Model":   reflect.ValueOf(func(name string) *goModelProxy { return newModelProxy(h, name) }),
		"Session": reflect.ValueOf(func() bridge.Session { return h.get().Session() }),
		"DB":      reflect.ValueOf(func() *goDBProxy { return newDBProxy(h) }),
		"HTTP":    reflect.ValueOf(func() *goHTTPProxy { return newHTTPProxy(h) }),
		"Cache":   reflect.ValueOf(func() *goCacheProxy { return newCacheProxy(h) }),
		"FS":      reflect.ValueOf(func() *goFSProxy { return newFSProxy(h) }),
		"Env":     reflect.ValueOf(func(key string) (string, error) { return h.get().Env(key) }),
		"Config":  reflect.ValueOf(func(key string) any { return h.get().Config(key) }),
		"Log":     reflect.ValueOf(func(level, msg string, data ...map[string]any) { h.get().Log(level, msg, data...) }),
		"Emit":    reflect.ValueOf(func(event string, data map[string]any) error { return h.get().Emit(event, data) }),
		"Call":    reflect.ValueOf(func(process string, input map[string]any) (any, error) { return h.get().Call(process, input) }),
		"T":       reflect.ValueOf(func(key string) string { return h.get().T(key) }),
		"Exec": reflect.ValueOf(func(cmd string, args []string, opts ...map[string]any) (*bridge.ExecResult, error) {
			return h.get().Exec(cmd, args, embedded.ParseExecOpts(firstMap(opts)))
		}),
		"Email":     reflect.ValueOf(func() *goEmailProxy { return newEmailProxy(h) }),
		"Notify":    reflect.ValueOf(func() *goNotifyProxy { return newNotifyProxy(h) }),
		"Storage":   reflect.ValueOf(func() *goStorageProxy { return newStorageProxy(h) }),
		"Security":  reflect.ValueOf(func() *goSecurityProxy { return newSecurityProxy(h) }),
		"Audit":     reflect.ValueOf(func() *goAuditProxy { return newAuditProxy(h) }),
		"Crypto":    reflect.ValueOf(func() *goCryptoProxy { return newCryptoProxy(h) }),
		"Execution": reflect.ValueOf(func() *goExecutionProxy { return newExecutionProxy(h) }),
		"Tx": reflect.ValueOf(func(fn func() error) error {
			return h.get().Tx(func(txCtx *bridge.Context) error {
				original := h.ctx
				h.ctx = txCtx
				defer func() { h.ctx = original }()
				return fn()
			})
		}),
	}
}

// --- Model proxy (PascalCase for Go convention) ---

type goModelProxy struct {
	h    *bridgeHolder
	name string
}

func newModelProxy(h *bridgeHolder, name string) *goModelProxy {
	return &goModelProxy{h: h, name: name}
}

func (m *goModelProxy) handle() bridge.ModelHandle { return m.h.get().Model(m.name) }

func (m *goModelProxy) Search(opts ...map[string]any) ([]map[string]any, error) {
	return m.handle().Search(embedded.ParseSearchOpts(firstMap(opts)))
}
func (m *goModelProxy) Get(id string) (map[string]any, error) { return m.handle().Get(id) }
func (m *goModelProxy) Create(data map[string]any) (map[string]any, error) {
	return m.handle().Create(data)
}
func (m *goModelProxy) Write(id string, data map[string]any) error {
	return m.handle().Write(id, data)
}
func (m *goModelProxy) Delete(id string) error { return m.handle().Delete(id) }
func (m *goModelProxy) Count(opts ...map[string]any) (int64, error) {
	return m.handle().Count(embedded.ParseSearchOpts(firstMap(opts)))
}
func (m *goModelProxy) Sum(field string, opts ...map[string]any) (float64, error) {
	return m.handle().Sum(field, embedded.ParseSearchOpts(firstMap(opts)))
}
func (m *goModelProxy) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	return m.handle().Upsert(data, uniqueFields)
}
func (m *goModelProxy) CreateMany(records []map[string]any) ([]map[string]any, error) {
	return m.handle().CreateMany(records)
}
func (m *goModelProxy) WriteMany(ids []string, data map[string]any) (*bridge.BulkResult, error) {
	return m.handle().WriteMany(ids, data)
}
func (m *goModelProxy) DeleteMany(ids []string) (*bridge.BulkResult, error) {
	return m.handle().DeleteMany(ids)
}
func (m *goModelProxy) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	return m.handle().UpsertMany(records, uniqueFields)
}
func (m *goModelProxy) AddRelation(id, field string, relatedIDs []string) error {
	return m.handle().AddRelation(id, field, relatedIDs)
}
func (m *goModelProxy) RemoveRelation(id, field string, relatedIDs []string) error {
	return m.handle().RemoveRelation(id, field, relatedIDs)
}
func (m *goModelProxy) SetRelation(id, field string, relatedIDs []string) error {
	return m.handle().SetRelation(id, field, relatedIDs)
}
func (m *goModelProxy) LoadRelation(id, field string) ([]map[string]any, error) {
	return m.handle().LoadRelation(id, field)
}
func (m *goModelProxy) Sudo() *goSudoModelProxy {
	return &goSudoModelProxy{h: m.h, name: m.name}
}

// --- Sudo model proxy ---

type goSudoModelProxy struct {
	h    *bridgeHolder
	name string
}

func (s *goSudoModelProxy) sudoHandle() bridge.SudoModelHandle {
	return s.h.get().Model(s.name).Sudo()
}

func (s *goSudoModelProxy) Search(opts ...map[string]any) ([]map[string]any, error) {
	return s.sudoHandle().Search(embedded.ParseSearchOpts(firstMap(opts)))
}
func (s *goSudoModelProxy) Get(id string) (map[string]any, error) { return s.sudoHandle().Get(id) }
func (s *goSudoModelProxy) Create(data map[string]any) (map[string]any, error) {
	return s.sudoHandle().Create(data)
}
func (s *goSudoModelProxy) Write(id string, data map[string]any) error {
	return s.sudoHandle().Write(id, data)
}
func (s *goSudoModelProxy) Delete(id string) error { return s.sudoHandle().Delete(id) }
func (s *goSudoModelProxy) Count(opts ...map[string]any) (int64, error) {
	return s.sudoHandle().Count(embedded.ParseSearchOpts(firstMap(opts)))
}
func (s *goSudoModelProxy) Sum(field string, opts ...map[string]any) (float64, error) {
	return s.sudoHandle().Sum(field, embedded.ParseSearchOpts(firstMap(opts)))
}
func (s *goSudoModelProxy) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	return s.sudoHandle().Upsert(data, uniqueFields)
}
func (s *goSudoModelProxy) CreateMany(records []map[string]any) ([]map[string]any, error) {
	return s.sudoHandle().CreateMany(records)
}
func (s *goSudoModelProxy) WriteMany(ids []string, data map[string]any) (*bridge.BulkResult, error) {
	return s.sudoHandle().WriteMany(ids, data)
}
func (s *goSudoModelProxy) DeleteMany(ids []string) (*bridge.BulkResult, error) {
	return s.sudoHandle().DeleteMany(ids)
}
func (s *goSudoModelProxy) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	return s.sudoHandle().UpsertMany(records, uniqueFields)
}
func (s *goSudoModelProxy) AddRelation(id, field string, relatedIDs []string) error {
	return s.sudoHandle().AddRelation(id, field, relatedIDs)
}
func (s *goSudoModelProxy) RemoveRelation(id, field string, relatedIDs []string) error {
	return s.sudoHandle().RemoveRelation(id, field, relatedIDs)
}
func (s *goSudoModelProxy) SetRelation(id, field string, relatedIDs []string) error {
	return s.sudoHandle().SetRelation(id, field, relatedIDs)
}
func (s *goSudoModelProxy) LoadRelation(id, field string) ([]map[string]any, error) {
	return s.sudoHandle().LoadRelation(id, field)
}
func (s *goSudoModelProxy) HardDelete(id string) error { return s.sudoHandle().HardDelete(id) }
func (s *goSudoModelProxy) HardDeleteMany(ids []string) (*bridge.BulkResult, error) {
	return s.sudoHandle().HardDeleteMany(ids)
}
func (s *goSudoModelProxy) WithTenant(tenantID string) *goSudoModelProxy {
	return s
}
func (s *goSudoModelProxy) SkipValidation() *goSudoModelProxy {
	return s
}

// --- DB proxy ---

type goDBProxy struct {
	h *bridgeHolder
}

func newDBProxy(h *bridgeHolder) *goDBProxy { return &goDBProxy{h: h} }
func (d *goDBProxy) Query(sql string, args ...any) ([]map[string]any, error) {
	return d.h.get().DB().Query(sql, args...)
}
func (d *goDBProxy) Execute(sql string, args ...any) (*bridge.ExecDBResult, error) {
	return d.h.get().DB().Execute(sql, args...)
}

// --- HTTP proxy ---

type goHTTPProxy struct {
	h *bridgeHolder
}

func newHTTPProxy(h *bridgeHolder) *goHTTPProxy { return &goHTTPProxy{h: h} }
func (p *goHTTPProxy) Get(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return p.h.get().HTTP().Get(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (p *goHTTPProxy) Post(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return p.h.get().HTTP().Post(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (p *goHTTPProxy) Put(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return p.h.get().HTTP().Put(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (p *goHTTPProxy) Patch(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return p.h.get().HTTP().Patch(url, embedded.ParseHTTPOpts(firstMap(opts)))
}
func (p *goHTTPProxy) Delete(url string, opts ...map[string]any) (*bridge.HTTPResponse, error) {
	return p.h.get().HTTP().Delete(url, embedded.ParseHTTPOpts(firstMap(opts)))
}

// --- Cache proxy ---

type goCacheProxy struct {
	h *bridgeHolder
}

func newCacheProxy(h *bridgeHolder) *goCacheProxy { return &goCacheProxy{h: h} }
func (c *goCacheProxy) Get(key string) (any, error) { return c.h.get().Cache().Get(key) }
func (c *goCacheProxy) Set(key string, val any, opts ...map[string]any) error {
	return c.h.get().Cache().Set(key, val, embedded.ParseCacheOpts(firstMap(opts)))
}
func (c *goCacheProxy) Del(key string) error { return c.h.get().Cache().Del(key) }

// --- FS proxy ---

type goFSProxy struct {
	h *bridgeHolder
}

func newFSProxy(h *bridgeHolder) *goFSProxy          { return &goFSProxy{h: h} }
func (f *goFSProxy) Read(path string) (string, error)  { return f.h.get().FS().Read(path) }
func (f *goFSProxy) Write(path, content string) error   { return f.h.get().FS().Write(path, content) }
func (f *goFSProxy) Exists(path string) (bool, error)   { return f.h.get().FS().Exists(path) }
func (f *goFSProxy) List(path string) ([]string, error) { return f.h.get().FS().List(path) }
func (f *goFSProxy) Mkdir(path string) error             { return f.h.get().FS().Mkdir(path) }
func (f *goFSProxy) Remove(path string) error            { return f.h.get().FS().Remove(path) }

// --- Email proxy ---

type goEmailProxy struct {
	h *bridgeHolder
}

func newEmailProxy(h *bridgeHolder) *goEmailProxy { return &goEmailProxy{h: h} }
func (e *goEmailProxy) Send(opts map[string]any) error {
	return e.h.get().Email().Send(embedded.ParseEmailOpts(opts))
}

// --- Notify proxy ---

type goNotifyProxy struct {
	h *bridgeHolder
}

func newNotifyProxy(h *bridgeHolder) *goNotifyProxy { return &goNotifyProxy{h: h} }
func (n *goNotifyProxy) Send(opts map[string]any) error {
	return n.h.get().Notify().Send(embedded.ParseNotifyOpts(opts))
}
func (n *goNotifyProxy) Broadcast(channel string, data map[string]any) error {
	return n.h.get().Notify().Broadcast(channel, data)
}

// --- Storage proxy ---

type goStorageProxy struct {
	h *bridgeHolder
}

func newStorageProxy(h *bridgeHolder) *goStorageProxy { return &goStorageProxy{h: h} }
func (s *goStorageProxy) Upload(opts map[string]any) (*bridge.Attachment, error) {
	return s.h.get().Storage().Upload(embedded.ParseUploadOpts(opts))
}
func (s *goStorageProxy) URL(id string) (string, error)      { return s.h.get().Storage().URL(id) }
func (s *goStorageProxy) Download(id string) ([]byte, error) { return s.h.get().Storage().Download(id) }
func (s *goStorageProxy) Delete(id string) error             { return s.h.get().Storage().Delete(id) }

// --- Security proxy ---

type goSecurityProxy struct {
	h *bridgeHolder
}

func newSecurityProxy(h *bridgeHolder) *goSecurityProxy { return &goSecurityProxy{h: h} }
func (s *goSecurityProxy) Permissions(model string) (*bridge.ModelPermissions, error) {
	return s.h.get().Security().Permissions(model)
}
func (s *goSecurityProxy) HasGroup(group string) (bool, error) {
	return s.h.get().Security().HasGroup(group)
}
func (s *goSecurityProxy) Groups() ([]string, error) { return s.h.get().Security().Groups() }

// --- Audit proxy ---

type goAuditProxy struct {
	h *bridgeHolder
}

func newAuditProxy(h *bridgeHolder) *goAuditProxy { return &goAuditProxy{h: h} }
func (a *goAuditProxy) Log(opts map[string]any) error {
	return a.h.get().Audit().Log(embedded.ParseAuditOpts(opts))
}

// --- Crypto proxy ---

type goCryptoProxy struct {
	h *bridgeHolder
}

func newCryptoProxy(h *bridgeHolder) *goCryptoProxy { return &goCryptoProxy{h: h} }
func (c *goCryptoProxy) Encrypt(plaintext string) (string, error) {
	return c.h.get().Crypto().Encrypt(plaintext)
}
func (c *goCryptoProxy) Decrypt(ciphertext string) (string, error) {
	return c.h.get().Crypto().Decrypt(ciphertext)
}
func (c *goCryptoProxy) Hash(value string) (string, error) { return c.h.get().Crypto().Hash(value) }
func (c *goCryptoProxy) Verify(value, hash string) (bool, error) {
	return c.h.get().Crypto().Verify(value, hash)
}

// --- Execution proxy ---

type goExecutionProxy struct {
	h *bridgeHolder
}

func newExecutionProxy(h *bridgeHolder) *goExecutionProxy { return &goExecutionProxy{h: h} }
func (e *goExecutionProxy) Search(opts map[string]any) ([]map[string]any, error) {
	return e.h.get().Execution().Search(parseExecSearchOpts(opts))
}
func (e *goExecutionProxy) Get(id string) (map[string]any, error) {
	return e.h.get().Execution().Get(id)
}
func (e *goExecutionProxy) Current() *bridge.ExecutionInfo { return e.h.get().Execution().Current() }
func (e *goExecutionProxy) Cancel(id string) error         { return e.h.get().Execution().Cancel(id) }

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
