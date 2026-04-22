package ddd

// Aggregate is an entity that can raise domain events.
type Aggregate interface {
	Entity
	GetDomainEvents() []DomainEvent
	ClearDomainEvents()
	RaiseEvent(event DomainEvent)
}

// BaseAggregate provides a default implementation of Aggregate.
type BaseAggregate struct {
	BaseEntity
	domainEvents []DomainEvent `gorm:"-" json:"-"`
}

func (a *BaseAggregate) GetDomainEvents() []DomainEvent { return a.domainEvents }
func (a *BaseAggregate) ClearDomainEvents()             { a.domainEvents = nil }
func (a *BaseAggregate) RaiseEvent(event DomainEvent)   { a.domainEvents = append(a.domainEvents, event) }
