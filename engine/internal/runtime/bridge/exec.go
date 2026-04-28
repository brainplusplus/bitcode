package bridge

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type execBridge struct {
	rules SecurityRules
}

func newExecBridge(rules SecurityRules) *execBridge {
	return &execBridge{rules: rules}
}

func (e *execBridge) Exec(cmd string, args []string, opts *ExecOptions) (*ExecResult, error) {
	basename := filepath.Base(cmd)
	lowerBase := strings.ToLower(basename)

	for _, denied := range DeniedCommands {
		if lowerBase == strings.ToLower(denied) {
			return nil, NewErrorf(ErrExecDenied, "command '%s' is denied", cmd)
		}
	}

	if matchesAny(lowerBase, toLower(e.rules.ExecDeny)) {
		return nil, NewErrorf(ErrExecDenied, "command '%s' is denied by module rules", cmd)
	}

	if len(e.rules.ExecAllow) == 0 {
		return nil, NewError(ErrExecDenied, "exec not enabled for this module (no exec_allow configured)")
	}

	if !matchesAny(lowerBase, toLower(e.rules.ExecAllow)) {
		return nil, NewErrorf(ErrExecDenied, "command '%s' not in exec_allow list", cmd)
	}

	timeout := 30 * time.Second
	if opts != nil && opts.Timeout > 0 {
		timeout = time.Duration(opts.Timeout) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	command := exec.CommandContext(ctx, cmd, args...)
	if opts != nil && opts.Cwd != "" {
		command.Dir = opts.Cwd
	}

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()

	result := &ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result, NewErrorf(ErrExecTimeout, "command '%s' timed out after %v", cmd, timeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		if execErr, ok := err.(*exec.Error); ok {
			return nil, NewErrorf(ErrExecNotFound, "command not found: '%s' (%s)", cmd, execErr.Err)
		}
		return result, NewError(ErrInternalError, err.Error())
	}

	return result, nil
}

func toLower(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToLower(s)
	}
	return out
}
