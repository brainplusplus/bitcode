package embedded

import (
	"context"
	"fmt"
	"time"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

func ExecuteEmbedded(
	ctx context.Context,
	runtime EmbeddedRuntime,
	scriptPath string,
	params map[string]any,
	bridgeCtx *bridge.Context,
	timeout time.Duration,
) (any, error) {
	code, err := LoadScript(scriptPath)
	if err != nil {
		return nil, bridge.NewErrorf(bridge.ErrFSNotFound, "script not found: %s", scriptPath)
	}

	vm, err := runtime.NewVM(VMOptions{Timeout: timeout})
	if err != nil {
		return nil, bridge.NewErrorf(bridge.ErrInternalError, "failed to create VM: %s", err)
	}
	defer vm.Close()

	if err := vm.InjectBridge(bridgeCtx); err != nil {
		return nil, err
	}

	if err := vm.InjectParams(params); err != nil {
		return nil, err
	}

	type result struct {
		value any
		err   error
	}
	done := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{nil, bridge.NewErrorf(bridge.ErrInternalError, "VM panic: %v", r)}
			}
		}()
		val, execErr := vm.Execute(code, scriptPath)
		done <- result{val, execErr}
	}()

	var timer *time.Timer
	if timeout > 0 {
		timer = time.AfterFunc(timeout, func() {
			vm.Interrupt(fmt.Sprintf("execution timeout after %v", timeout))
		})
		defer timer.Stop()
	}

	select {
	case res := <-done:
		return res.value, res.err
	case <-ctx.Done():
		vm.Interrupt("context cancelled")
		return nil, ctx.Err()
	}
}
