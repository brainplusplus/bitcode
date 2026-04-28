package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/stdlib"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

// GoJSONRunner executes go-json programs as bitcode process steps.
type GoJSONRunner struct {
	BridgeCtx *bridge.Context
	ScriptDir string
}

func (r *GoJSONRunner) CanHandle(rt string) bool {
	return rt == "go-json"
}

func (r *GoJSONRunner) Run(ctx context.Context, script string, params map[string]any) (any, error) {
	scriptPath := script
	if !filepath.IsAbs(scriptPath) && r.ScriptDir != "" {
		scriptPath = filepath.Join(r.ScriptDir, script)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("go-json: cannot read script %s: %w", script, err)
	}

	reg := stdlib.DefaultRegistry()
	opts := []runtime.Option{
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
		runtime.WithRuntimeContext(ctx),
		runtime.WithoutIO(),
	}

	if r.BridgeCtx != nil {
		ext := bridge.BuildGoJSONExtension(r.BridgeCtx)
		opts = append(opts, runtime.WithExtension("bitcode", ext))

		s := r.BridgeCtx.Session()
		opts = append(opts, runtime.WithSession(&runtime.Session{
			UserID:   s.UserID,
			Locale:   s.Locale,
			TenantID: s.TenantID,
			Groups:   s.Groups,
		}))
	}

	rt := runtime.NewRuntime(opts...)

	program, err := lang.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("go-json: compile error in %s: %w", script, err)
	}

	dir := filepath.Dir(scriptPath)
	resolver := lang.NewImportResolver()
	if err := resolver.ResolveImports(program, dir, []string{scriptPath}); err != nil {
		return nil, fmt.Errorf("go-json: import error in %s: %w", script, err)
	}

	compiled, err := rt.Compile(data)
	if err != nil {
		return nil, fmt.Errorf("go-json: compile error in %s: %w", script, err)
	}

	input := make(map[string]any)
	if params != nil {
		for k, v := range params {
			input[k] = v
		}
	}

	result, err := rt.Execute(compiled, input)
	if err != nil {
		return nil, fmt.Errorf("go-json: execution error in %s: %w", script, err)
	}

	return result.Value, nil
}
