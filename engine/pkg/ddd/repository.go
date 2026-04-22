package ddd

import "context"

// Repository is the generic interface for data access.
type Repository[T any] interface {
	Save(ctx context.Context, entity *T) error
	FindByID(ctx context.Context, id string) (*T, error)
	FindAll(ctx context.Context, filters map[string]interface{}, page int, pageSize int) ([]T, int64, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) error
}
