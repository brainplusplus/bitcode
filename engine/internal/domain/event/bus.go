package event

import (
	"context"
	"sync"
)

type Handler func(ctx context.Context, eventName string, data map[string]any) error

type Bus struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

func (b *Bus) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

func (b *Bus) SubscribeAll(handler Handler) {
	b.Subscribe("*", handler)
}

func (b *Bus) Publish(ctx context.Context, eventName string, data map[string]any) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, h := range b.handlers[eventName] {
		if err := h(ctx, eventName, data); err != nil {
			return err
		}
	}

	for _, h := range b.handlers["*"] {
		if err := h(ctx, eventName, data); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bus) HasSubscribers(eventName string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[eventName]) > 0 || len(b.handlers["*"]) > 0
}
