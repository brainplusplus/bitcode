package bridge

import "gorm.io/gorm"

// Context is the single struct passed to all runtimes (Node.js, Python, goja, yaegi).
// It holds all 20 bridge namespaces and provides the unified bitcode.* API.
type Context struct {
	txManager TxManager
	model     ModelFactory
	db        DB
	http      HTTPClient
	cache     Cache
	fs        FS
	session   Session
	config    ConfigReader
	env       EnvReader
	emitter   EventEmitter
	caller    ProcessCaller
	execer    CommandExecutor
	logger    Logger
	email     EmailSender
	notify    Notifier
	storage   Storage
	i18n      I18N
	security  SecurityChecker
	audit     AuditLogger
	crypto    Crypto
	execution ExecutionLog
}

func (c *Context) Tx(fn func(tx *Context) error) error {
	return c.txManager.RunTx(c, fn)
}

func (c *Context) Model(name string) ModelHandle {
	return c.model.Model(name, c.session, false)
}

func (c *Context) DB() DB             { return c.db }
func (c *Context) HTTP() HTTPClient   { return c.http }
func (c *Context) Cache() Cache       { return c.cache }
func (c *Context) FS() FS             { return c.fs }
func (c *Context) Session() Session   { return c.session }

func (c *Context) Config(key string) any       { return c.config.Get(key) }
func (c *Context) Env(key string) (string, error) { return c.env.Get(key) }
func (c *Context) Emit(event string, data map[string]any) error {
	return c.emitter.Emit(event, data)
}
func (c *Context) Call(process string, input map[string]any) (any, error) {
	return c.caller.Call(process, input)
}
func (c *Context) Exec(cmd string, args []string, opts *ExecOptions) (*ExecResult, error) {
	return c.execer.Exec(cmd, args, opts)
}
func (c *Context) Log(level, msg string, data ...map[string]any) {
	c.logger.Log(level, msg, data...)
}

func (c *Context) Email() EmailSender         { return c.email }
func (c *Context) Notify() Notifier           { return c.notify }
func (c *Context) Storage() Storage           { return c.storage }
func (c *Context) T(key string) string        { return c.i18n.Translate(c.session.Locale, key) }
func (c *Context) Security() SecurityChecker  { return c.security }
func (c *Context) Audit() AuditLogger         { return c.audit }
func (c *Context) Crypto() Crypto             { return c.crypto }
func (c *Context) Execution() ExecutionLog    { return c.execution }

func (c *Context) cloneWithTx(gormTx *gorm.DB) *Context {
	clone := *c
	if dbImpl, ok := c.db.(*dbBridge); ok {
		clone.db = dbImpl.withTx(gormTx)
	}
	return &clone
}
