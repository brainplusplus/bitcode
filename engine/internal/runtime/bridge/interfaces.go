package bridge

// ModelHandle provides permission-aware CRUD, bulk, and relation operations on a single model.
type ModelHandle interface {
	Search(opts SearchOptions) ([]map[string]any, error)
	Get(id string, opts ...GetOptions) (map[string]any, error)
	Create(data map[string]any) (map[string]any, error)
	Write(id string, data map[string]any) error
	Delete(id string) error
	Count(opts SearchOptions) (int64, error)
	Sum(field string, opts SearchOptions) (float64, error)
	Upsert(data map[string]any, uniqueFields []string) (map[string]any, error)

	CreateMany(records []map[string]any) ([]map[string]any, error)
	WriteMany(ids []string, data map[string]any) (*BulkResult, error)
	DeleteMany(ids []string) (*BulkResult, error)
	UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error)

	AddRelation(id string, field string, relatedIDs []string) error
	RemoveRelation(id string, field string, relatedIDs []string) error
	SetRelation(id string, field string, relatedIDs []string) error
	LoadRelation(id string, field string) ([]map[string]any, error)

	Sudo() SudoModelHandle
}

// SudoModelHandle extends ModelHandle with system-level operations that bypass permissions.
type SudoModelHandle interface {
	ModelHandle

	HardDelete(id string) error
	HardDeleteMany(ids []string) (*BulkResult, error)
	WithTenant(tenantID string) SudoModelHandle
	SkipValidation() SudoModelHandle
}

type ModelFactory interface {
	Model(name string, session Session, sudo bool) ModelHandle
}

type DB interface {
	Query(sql string, args ...any) ([]map[string]any, error)
	Execute(sql string, args ...any) (*ExecDBResult, error)
}

type HTTPClient interface {
	Get(url string, opts *HTTPOptions) (*HTTPResponse, error)
	Post(url string, opts *HTTPOptions) (*HTTPResponse, error)
	Put(url string, opts *HTTPOptions) (*HTTPResponse, error)
	Patch(url string, opts *HTTPOptions) (*HTTPResponse, error)
	Delete(url string, opts *HTTPOptions) (*HTTPResponse, error)
}

type Cache interface {
	Get(key string) (any, error)
	Set(key string, value any, opts *CacheSetOptions) error
	Del(key string) error
}

type FS interface {
	Read(path string) (string, error)
	Write(path string, content string) error
	Exists(path string) (bool, error)
	List(path string) ([]string, error)
	Mkdir(path string) error
	Remove(path string) error
}

type EnvReader interface {
	Get(key string) (string, error)
}

type ConfigReader interface {
	Get(key string) any
}

type EventEmitter interface {
	Emit(event string, data map[string]any) error
}

type ProcessCaller interface {
	Call(process string, input map[string]any) (any, error)
}

type CommandExecutor interface {
	Exec(cmd string, args []string, opts *ExecOptions) (*ExecResult, error)
}

type Logger interface {
	Log(level, msg string, data ...map[string]any)
}

type EmailSender interface {
	Send(opts EmailOptions) error
}

type Notifier interface {
	Send(opts NotifyOptions) error
	Broadcast(channel string, data map[string]any) error
}

type Storage interface {
	Upload(opts UploadOptions) (*Attachment, error)
	URL(id string) (string, error)
	Download(id string) ([]byte, error)
	Delete(id string) error
}

type I18N interface {
	Translate(locale, key string) string
}

type SecurityChecker interface {
	Permissions(modelName string) (*ModelPermissions, error)
	HasGroup(groupName string) (bool, error)
	Groups() ([]string, error)
}

type AuditLogger interface {
	Log(opts AuditOptions) error
}

type Crypto interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	Hash(value string) (string, error)
	Verify(value, hash string) (bool, error)
}

type ExecutionLog interface {
	Search(opts ExecutionSearchOptions) ([]map[string]any, error)
	Get(id string, opts ...GetOptions) (map[string]any, error)
	Current() *ExecutionInfo
	Retry(id string) (map[string]any, error)
	Cancel(id string) error
}

type TxManager interface {
	RunTx(parent *Context, fn func(tx *Context) error) error
}
