package event

import (
	"context"
	"testing"
)

func TestBus_PublishSubscribe(t *testing.T) {
	bus := NewBus()
	received := false

	bus.Subscribe("order.created", func(ctx context.Context, name string, data map[string]any) error {
		received = true
		if name != "order.created" {
			t.Errorf("expected order.created, got %s", name)
		}
		if data["id"] != "123" {
			t.Errorf("expected id 123, got %v", data["id"])
		}
		return nil
	})

	err := bus.Publish(context.Background(), "order.created", map[string]any{"id": "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !received {
		t.Error("handler should have been called")
	}
}

func TestBus_SubscribeAll(t *testing.T) {
	bus := NewBus()
	count := 0

	bus.SubscribeAll(func(ctx context.Context, name string, data map[string]any) error {
		count++
		return nil
	})

	bus.Publish(context.Background(), "event.a", nil)
	bus.Publish(context.Background(), "event.b", nil)

	if count != 2 {
		t.Errorf("expected 2 calls, got %d", count)
	}
}

func TestBus_NoSubscribers(t *testing.T) {
	bus := NewBus()
	err := bus.Publish(context.Background(), "nobody.listens", nil)
	if err != nil {
		t.Fatalf("should not error with no subscribers: %v", err)
	}
}

func TestBus_HasSubscribers(t *testing.T) {
	bus := NewBus()
	if bus.HasSubscribers("test") {
		t.Error("should have no subscribers")
	}

	bus.Subscribe("test", func(ctx context.Context, name string, data map[string]any) error { return nil })
	if !bus.HasSubscribers("test") {
		t.Error("should have subscribers")
	}
}
