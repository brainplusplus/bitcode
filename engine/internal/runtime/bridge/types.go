package bridge

// SearchOptions configures model search queries.
type SearchOptions struct {
	Domain  [][]any  `json:"domain,omitempty"`
	Fields  []string `json:"fields,omitempty"`
	Order   string   `json:"order,omitempty"`
	Limit   int      `json:"limit,omitempty"`
	Offset  int      `json:"offset,omitempty"`
	Include []string `json:"include,omitempty"`
}

type GetOptions struct {
	Include []string `json:"include,omitempty"`
}

type BulkResult struct {
	Affected int64 `json:"affected"`
}

type HTTPOptions struct {
	Headers            map[string]string `json:"headers,omitempty"`
	HeaderOrder        []string          `json:"headerOrder,omitempty"`
	Body               any               `json:"body,omitempty"`
	Timeout            int               `json:"timeout,omitempty"`
	Profile            string            `json:"profile,omitempty"`
	Proxy              string            `json:"proxy,omitempty"`
	CookieJar          string            `json:"cookieJar,omitempty"`
	FollowRedirects    *bool             `json:"followRedirects,omitempty"`
	InsecureSkipVerify bool              `json:"insecureSkipVerify,omitempty"`
}

type HTTPResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

type CacheSetOptions struct {
	TTL int `json:"ttl,omitempty"` // seconds
}

type ExecOptions struct {
	Cwd     string `json:"cwd,omitempty"`
	Timeout int    `json:"timeout,omitempty"` // ms
}

type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

type ExecDBResult struct {
	RowsAffected int64 `json:"rows_affected"`
}

type EmailOptions struct {
	To       string         `json:"to"`
	Subject  string         `json:"subject"`
	Body     string         `json:"body,omitempty"`
	Template string         `json:"template,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

type NotifyOptions struct {
	To      string `json:"to"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"` // success | warning | error | info
}

type UploadOptions struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
	Model    string `json:"model,omitempty"`
	RecordID string `json:"recordId,omitempty"`
}

type Attachment struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

type AuditOptions struct {
	Action   string `json:"action"`
	Model    string `json:"model,omitempty"`
	RecordID string `json:"recordId,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type ModelPermissions struct {
	CanRead   bool `json:"canRead"`
	CanWrite  bool `json:"canWrite"`
	CanCreate bool `json:"canCreate"`
	CanDelete bool `json:"canDelete"`
	CanPrint  bool `json:"canPrint"`
	CanEmail  bool `json:"canEmail"`
	CanExport bool `json:"canExport"`
	CanImport bool `json:"canImport"`
	CanClone  bool `json:"canClone"`
}

type ExecutionSearchOptions struct {
	Process string `json:"process,omitempty"`
	Status  string `json:"status,omitempty"`
	UserID  string `json:"userId,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	Order   string `json:"order,omitempty"`
}

type ExecutionInfo struct {
	ID          string `json:"id"`
	ProcessName string `json:"processName"`
	StartedAt   string `json:"startedAt"`
	ParentID    string `json:"parentId,omitempty"`
	StepIndex   int    `json:"stepIndex"`
	Trigger     string `json:"trigger"`
	Mode        string `json:"mode"`
}

// SecurityRules controls per-module access to sensitive bridge operations.
type SecurityRules struct {
	EnvAllow  []string `json:"env_allow,omitempty"`
	EnvDeny   []string `json:"env_deny,omitempty"`
	ExecAllow []string `json:"exec_allow,omitempty"`
	ExecDeny  []string `json:"exec_deny,omitempty"`
	FSAllow   []string `json:"fs_allow,omitempty"`
	FSDeny    []string `json:"fs_deny,omitempty"`
	SudoAllow bool     `json:"sudo_allow,omitempty"`
}

// EngineSecrets are always denied from env access regardless of module config.
var EngineSecrets = []string{
	"JWT_SECRET", "DB_PASSWORD", "ENCRYPTION_KEY",
	"SMTP_PASSWORD", "STORAGE_S3_SECRET_KEY", "STORAGE_S3_ACCESS_KEY",
}

// DeniedCommands are always blocked from exec regardless of module config.
var DeniedCommands = []string{
	"rm", "rmdir", "del", "format", "shutdown", "reboot",
	"halt", "poweroff", "dd", "mkfs", "fdisk",
}

// Session holds the current request context passed to all bridge operations.
type Session struct {
	UserID   string         `json:"userId"`
	Username string         `json:"username"`
	Email    string         `json:"email"`
	TenantID string         `json:"tenantId"`
	Groups   []string       `json:"groups"`
	Locale   string         `json:"locale"`
	Context  map[string]any `json:"context"`
}

// ExecutionLogConfig controls execution log retention and behavior.
type ExecutionLogConfig struct {
	Enabled        bool   `json:"enabled"`
	SaveInput      bool   `json:"save_input"`
	SaveOutput     bool   `json:"save_output"`
	SaveSteps      bool   `json:"save_steps"`
	SaveOnSuccess  bool   `json:"save_on_success"`
	MaxAge         string `json:"max_age"`
	MaxRecords     int64  `json:"max_records"`
	CleanupInterval string `json:"cleanup_interval"`
	MaxInputSize   int    `json:"max_input_size"`
	MaxOutputSize  int    `json:"max_output_size"`
}

// DefaultExecutionLogConfig returns sensible defaults.
func DefaultExecutionLogConfig() ExecutionLogConfig {
	return ExecutionLogConfig{
		Enabled:         true,
		SaveInput:       true,
		SaveOutput:      true,
		SaveSteps:       true,
		SaveOnSuccess:   true,
		MaxAge:          "30d",
		MaxRecords:      100000,
		CleanupInterval: "1h",
		MaxInputSize:    10240,
		MaxOutputSize:   10240,
	}
}

// SudoOptions configures sudo mode behavior.
type SudoOptions struct {
	SkipPermission  bool
	SkipValidation  bool
	SkipRecordRules bool
	HardDelete      bool
	TenantID        string
}
