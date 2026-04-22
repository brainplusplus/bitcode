package ddd

import (
	"testing"
	"time"
)

func TestBaseEntity(t *testing.T) {
	now := time.Now()
	e := BaseEntity{ID: "abc-123", CreatedAt: now, UpdatedAt: now}

	if e.GetID() != "abc-123" {
		t.Errorf("expected ID abc-123, got %s", e.GetID())
	}
	if !e.GetCreatedAt().Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, e.GetCreatedAt())
	}
	if !e.GetUpdatedAt().Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, e.GetUpdatedAt())
	}
}

func TestBaseAggregate_RaiseAndClearEvents(t *testing.T) {
	a := &BaseAggregate{}

	if len(a.GetDomainEvents()) != 0 {
		t.Errorf("expected 0 events, got %d", len(a.GetDomainEvents()))
	}

	evt1 := NewDomainEvent("order.created", "order-1")
	evt2 := NewDomainEvent("order.confirmed", "order-1")
	a.RaiseEvent(evt1)
	a.RaiseEvent(evt2)

	if len(a.GetDomainEvents()) != 2 {
		t.Errorf("expected 2 events, got %d", len(a.GetDomainEvents()))
	}
	if a.GetDomainEvents()[0].EventName() != "order.created" {
		t.Errorf("expected order.created, got %s", a.GetDomainEvents()[0].EventName())
	}

	a.ClearDomainEvents()
	if len(a.GetDomainEvents()) != 0 {
		t.Errorf("expected 0 events after clear, got %d", len(a.GetDomainEvents()))
	}
}

func TestBaseDomainEvent(t *testing.T) {
	before := time.Now()
	evt := NewDomainEvent("user.created", "user-42")
	after := time.Now()

	if evt.EventName() != "user.created" {
		t.Errorf("expected user.created, got %s", evt.EventName())
	}
	if evt.AggregateID() != "user-42" {
		t.Errorf("expected user-42, got %s", evt.AggregateID())
	}
	if evt.OccurredAt().Before(before) || evt.OccurredAt().After(after) {
		t.Errorf("OccurredAt %v not between %v and %v", evt.OccurredAt(), before, after)
	}
}
