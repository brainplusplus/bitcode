package persistence

import (
	"context"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type Repository interface {
	FindByID(ctx context.Context, id string) (map[string]any, error)
	FindAll(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error)
	Create(ctx context.Context, data map[string]any) (map[string]any, error)
	Update(ctx context.Context, id string, data map[string]any) error
	Delete(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error

	Upsert(ctx context.Context, data map[string]any, uniqueFields []string) (map[string]any, error)
	Count(ctx context.Context, query *Query) (int64, error)
	Sum(ctx context.Context, field string, query *Query) (float64, error)
	BulkCreate(ctx context.Context, records []map[string]any) ([]map[string]any, error)

	FindActive(ctx context.Context, id string) (map[string]any, error)
	FindAllActive(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error)
	CountActive(ctx context.Context, query *Query) (int64, error)
	SumActive(ctx context.Context, field string, query *Query) (float64, error)

	AddMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error
	RemoveMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error
	LoadMany2Many(ctx context.Context, id string, field string) ([]map[string]any, error)

	SetModelDef(def *parser.ModelDefinition)
	SetModelName(name string)
	SetCurrentUser(userID string)
	GetTableName() string
}

type SystemRepository interface {
	Insert(ctx context.Context, collection string, data map[string]any) error
	Find(ctx context.Context, collection string, query *Query, limit int) ([]map[string]any, error)
	FindOne(ctx context.Context, collection string, query *Query) (map[string]any, error)
	Count(ctx context.Context, collection string, query *Query) (int64, error)
	Migrate(collection string, columns []SystemColumn) error
}

type SystemColumn struct {
	Name     string
	Type     string
	Nullable bool
}

type SequenceEngine interface {
	MigrateSequenceTable() error
	NextValue(modelName, fieldName, sequenceKey string, step int) (int64, error)
}

type MigrationEngine interface {
	MigrateModel(model *parser.ModelDefinition, resolver TableNameResolver) error
}
