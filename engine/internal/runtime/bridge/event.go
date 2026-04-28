package bridge

import (
	"context"

	"github.com/bitcode-framework/bitcode/internal/domain/event"
)

type eventBridge struct {
	bus *event.Bus
}

func newEventBridge(bus *event.Bus) *eventBridge {
	return &eventBridge{bus: bus}
}

func (e *eventBridge) Emit(eventName string, data map[string]any) error {
	return e.bus.Publish(context.Background(), eventName, data)
}
