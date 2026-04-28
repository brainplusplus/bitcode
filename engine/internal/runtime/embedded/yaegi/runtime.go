package yaegi_runtime

import (
	"fmt"
	"reflect"

	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/traefik/yaegi/interp"
)

type YaegiRuntime struct {
	stdlibSymbols map[string]map[string]reflect.Value
	bridgeSources []BridgeSource
}

func New(bridgeSources []BridgeSource) *YaegiRuntime {
	return &YaegiRuntime{
		stdlibSymbols: FilteredStdlib(),
		bridgeSources: bridgeSources,
	}
}

func (r *YaegiRuntime) Name() string { return "yaegi" }

func (r *YaegiRuntime) NewVM(opts embedded.VMOptions) (embedded.VM, error) {
	i := interp.New(interp.Options{})

	if err := i.Use(r.stdlibSymbols); err != nil {
		return nil, err
	}

	for _, src := range r.bridgeSources {
		if _, err := i.Eval(src.Code); err != nil {
			return nil, fmt.Errorf("bridge '%s' eval error: %w", src.Filename, err)
		}
	}

	return &YaegiVM{interp: i, opts: opts}, nil
}
