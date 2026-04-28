package bridge

import (
	"context"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
)

type ProcessRegistry interface {
	LoadProcess(name string) (*parser.ProcessDefinition, error)
}

type processBridge struct {
	executor        *executor.Executor
	processRegistry ProcessRegistry
	session         Session
}

func newProcessBridge(exec *executor.Executor, registry ProcessRegistry, session Session) *processBridge {
	return &processBridge{
		executor:        exec,
		processRegistry: registry,
		session:         session,
	}
}

func (p *processBridge) Call(process string, input map[string]any) (any, error) {
	procDef, err := p.processRegistry.LoadProcess(process)
	if err != nil {
		return nil, NewErrorf(ErrInternalError, "process '%s' not found", process)
	}

	execCtx, err := p.executor.Execute(context.Background(), procDef, input, p.session.UserID)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}

	return execCtx.Result, nil
}
