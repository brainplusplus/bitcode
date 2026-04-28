package embedded

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

type ScriptRunnerConfig struct {
	Registry      *EngineRegistry
	BridgeFactory BridgeContextFactory
	DefaultEngine string
	DefaultTimeout time.Duration
	ModulePath    string
}

type BridgeContextFactory interface {
	NewContext(moduleName string, session bridge.Session, rules bridge.SecurityRules) *bridge.Context
}

type EmbeddedScriptRunner struct {
	config ScriptRunnerConfig
}

func NewScriptRunner(config ScriptRunnerConfig) *EmbeddedScriptRunner {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	if config.DefaultEngine == "" {
		config.DefaultEngine = "goja"
	}
	return &EmbeddedScriptRunner{config: config}
}

func (r *EmbeddedScriptRunner) CanHandle(runtime string) bool {
	if runtime == "" || runtime == "javascript" {
		return true
	}
	if runtime == "go" || strings.HasPrefix(runtime, "go:") {
		return true
	}
	return strings.HasPrefix(runtime, "javascript:")
}

func (r *EmbeddedScriptRunner) Run(ctx context.Context, script string, params map[string]any) (any, error) {
	runtimeField := ""
	if rt, ok := params["__runtime"].(string); ok {
		runtimeField = rt
		delete(params, "__runtime")
	}

	engine, err := r.config.Registry.Resolve(runtimeField, r.config.DefaultEngine)
	if err != nil {
		return nil, err
	}

	scriptPath := script
	if !filepath.IsAbs(scriptPath) && r.config.ModulePath != "" {
		scriptPath = filepath.Join(r.config.ModulePath, scriptPath)
	}

	var bridgeCtx *bridge.Context
	if r.config.BridgeFactory != nil {
		session := bridge.Session{}
		if userID, ok := params["user_id"].(string); ok {
			session.UserID = userID
		}
		bridgeCtx = r.config.BridgeFactory.NewContext("", session, bridge.SecurityRules{})
	}

	return ExecuteEmbedded(ctx, engine, scriptPath, params, bridgeCtx, r.config.DefaultTimeout)
}
