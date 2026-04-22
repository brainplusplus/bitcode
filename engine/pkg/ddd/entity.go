package ddd

import "time"

// Entity is the base interface for all domain entities.
type Entity interface {
	GetID() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// BaseEntity provides a default implementation of Entity.
type BaseEntity struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (e BaseEntity) GetID() string           { return e.ID }
func (e BaseEntity) GetCreatedAt() time.Time { return e.CreatedAt }
func (e BaseEntity) GetUpdatedAt() time.Time { return e.UpdatedAt }
