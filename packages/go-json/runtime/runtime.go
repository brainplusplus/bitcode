package runtime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/expr-lang/expr"
)

type Option func(*Runtime)

func WithStdlib(funcs []expr.Option) Option {
	return func(r *Runtime) { r.stdlibOpts = append(r.stdlibOpts, funcs...) }
}

func WithLimits(limits Limits) Option {
	return func(r *Runtime) { r.limits = limits }
}

func WithRuntimeLogger(l lang.Logger) Option {
	return func(r *Runtime) { r.logger = l }
}

func WithRuntimeDebugger(d lang.Debugger) Option {
	return func(r *Runtime) { r.debugger = d }
}

func WithRuntimeTrace(enabled bool) Option {
	return func(r *Runtime) { r.traceEnabled = enabled }
}

func WithSession(s *Session) Option {
	return func(r *Runtime) { r.session = s }
}

func WithRuntimeContext(ctx context.Context) Option {
	return func(r *Runtime) { r.ctx = ctx }
}

type Runtime struct {
	engine       *lang.ExprLangEngine
	limits       Limits
	logger       lang.Logger
	debugger     lang.Debugger
	traceEnabled bool
	session      *Session
	ctx          context.Context
	stdlibOpts   []expr.Option

	cache   map[string]*lang.CompiledProgram
	cacheMu sync.RWMutex
}

func NewRuntime(opts ...Option) *Runtime {
	r := &Runtime{
		engine: lang.NewExprLangEngine(),
		limits: DefaultLimits(),
		cache:  make(map[string]*lang.CompiledProgram),
		ctx:    context.Background(),
	}

	for _, opt := range opts {
		opt(r)
	}

	// Register stdlib options with the expression engine.
	if len(r.stdlibOpts) > 0 {
		r.engine.AddOptions(r.stdlibOpts...)
	}

	return r
}

// Compile parses and compiles a go-json program.
func (r *Runtime) Compile(input []byte) (*lang.CompiledProgram, error) {
	key := contentHash(input)

	r.cacheMu.RLock()
	if cached, ok := r.cache[key]; ok {
		r.cacheMu.RUnlock()
		return cached, nil
	}
	r.cacheMu.RUnlock()

	program, err := lang.Parse(input)
	if err != nil {
		return nil, err
	}

	compiled, err := lang.Compile(program, r.engine, r.limits.ToResolved())
	if err != nil {
		return nil, err
	}

	r.cacheMu.Lock()
	r.cache[key] = compiled
	r.cacheMu.Unlock()

	return compiled, nil
}

// Execute runs a compiled program with the given input.
func (r *Runtime) Execute(program *lang.CompiledProgram, input map[string]any) (*lang.ExecutionResult, error) {
	var vmOpts []lang.VMOption

	ctx := r.ctx
	vmOpts = append(vmOpts, lang.WithContext(ctx))

	if r.logger != nil {
		vmOpts = append(vmOpts, lang.WithLogger(r.logger))
	}
	if r.debugger != nil {
		vmOpts = append(vmOpts, lang.WithDebugger(r.debugger))
	}
	if r.traceEnabled {
		vmOpts = append(vmOpts, lang.WithTrace(true))
	}

	// Inject session into input.
	if r.session != nil && input != nil {
		input["session"] = r.session.ToMap()
	}

	vm := lang.NewVM(program, r.engine, vmOpts...)
	return vm.Execute(input)
}

// ExecuteJSON compiles and executes a program in one call (with caching).
func (r *Runtime) ExecuteJSON(programJSON []byte, input map[string]any) (*lang.ExecutionResult, error) {
	compiled, err := r.Compile(programJSON)
	if err != nil {
		return nil, err
	}
	return r.Execute(compiled, input)
}

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}
