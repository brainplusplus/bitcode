package ddd

import "time"

// DomainEvent represents something that happened in the domain.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
	AggregateID() string
}

// BaseDomainEvent provides a default implementation.
type BaseDomainEvent struct {
	Name        string    `json:"name"`
	Timestamp   time.Time `json:"occurred_at"`
	AggregateId string    `json:"aggregate_id"`
}

func (e BaseDomainEvent) EventName() string     { return e.Name }
func (e BaseDomainEvent) OccurredAt() time.Time { return e.Timestamp }
func (e BaseDomainEvent) AggregateID() string   { return e.AggregateId }

// NewDomainEvent creates a new domain event with the current timestamp.
func NewDomainEvent(name string, aggregateID string) BaseDomainEvent {
	return BaseDomainEvent{
		Name:        name,
		Timestamp:   time.Now(),
		AggregateId: aggregateID,
	}
}
