package io

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ExecModule provides command execution functions for go-json programs.
type ExecModule struct {
	security *SecurityConfig
	config   map[string]any
}

// NewExecModule creates a new command execution I/O module.
func NewExecModule(security *SecurityConfig) *ExecModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &ExecModule{security: security}
}

func (m *ExecModule) Name() string { return "exec" }

func (m *ExecModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *ExecModule) Functions() map[string]any {
	return map[string]any{
		"run": m.execRun,
	}
}

func (m *ExecModule) execRun(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("exec.run: cmd is required")
	}

	cmdName, ok := params[0].(string)
	if !ok {
		return nil, fmt.Errorf("exec.run: cmd must be a string")
	}

	if err := m.security.ValidateCommand(cmdName); err != nil {
		return nil, err
	}

	opts := extractExecOpts(params)

	var args []string
	if rawArgs, ok := opts["args"].([]any); ok {
		for _, a := range rawArgs {
			args = append(args, fmt.Sprintf("%v", a))
		}
	}

	timeoutMs := m.security.Exec.MaxTimeout
	if timeoutMs <= 0 {
		timeoutMs = 60
	}
	if t, ok := opts["timeout"]; ok {
		if tf, ok := toFloat64Val(t); ok {
			tSec := int(tf / 1000)
			if tSec > 0 && tSec < timeoutMs {
				timeoutMs = tSec
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, args...)

	if cwd, ok := opts["cwd"].(string); ok && cwd != "" {
		cmd.Dir = cwd
	}

	m.setupEnv(cmd, opts)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	maxOutput := m.security.Exec.MaxOutputSize
	if maxOutput <= 0 {
		maxOutput = 1024 * 1024
	}

	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	truncated := false

	if int64(len(stdoutStr)) > maxOutput {
		stdoutStr = stdoutStr[:maxOutput]
		truncated = true
	}
	if int64(len(stderrStr)) > maxOutput {
		stderrStr = stderrStr[:maxOutput]
		truncated = true
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("exec.run: command timed out after %d seconds", timeoutMs)
		} else {
			return nil, fmt.Errorf("exec.run: %s", err.Error())
		}
	}

	result := map[string]any{
		"exit_code": exitCode,
		"stdout":    stdoutStr,
		"stderr":    stderrStr,
	}
	if truncated {
		result["truncated"] = true
	}

	return result, nil
}

func (m *ExecModule) setupEnv(cmd *exec.Cmd, opts map[string]any) {
	if envMap, ok := opts["env"].(map[string]any); ok {
		env := make([]string, 0, len(envMap))
		for k, v := range envMap {
			if !isEngineSecret(k) {
				env = append(env, fmt.Sprintf("%s=%v", k, v))
			}
		}
		cmd.Env = env
		return
	}

	// Inherit host environment minus EngineSecrets.
	filtered := make([]string, 0)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && !isEngineSecret(parts[0]) {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = filtered
}

func extractExecOpts(params []any) map[string]any {
	opts := make(map[string]any)
	for i := 1; i < len(params); i++ {
		if m, ok := params[i].(map[string]any); ok {
			for k, v := range m {
				opts[k] = v
			}
		}
	}
	return opts
}
