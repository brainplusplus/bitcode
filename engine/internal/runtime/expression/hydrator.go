package expression

import (
	"context"
	"fmt"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"gorm.io/gorm"
)

type Hydrator struct {
	db            *gorm.DB
	modelRegistry ModelLookup
}

type ModelLookup interface {
	Get(name string) (*parser.ModelDefinition, error)
}

func NewHydrator(db *gorm.DB, registry ModelLookup) *Hydrator {
	return &Hydrator{db: db, modelRegistry: registry}
}

func (h *Hydrator) HydrateRecord(ctx context.Context, modelDef *parser.ModelDefinition, record map[string]any) error {
	if modelDef == nil || record == nil {
		return nil
	}

	childCollections := make(map[string][]map[string]any)

	for fieldName, field := range modelDef.Fields {
		if field.Type == parser.FieldOne2Many && field.Model != "" && field.Inverse != "" {
			children, err := h.loadChildren(ctx, field.Model, field.Inverse, record, modelDef)
			if err != nil {
				continue
			}
			childCollections[fieldName] = children
		}
	}

	for fieldName, field := range modelDef.Fields {
		expr := field.Computed
		if expr == "" {
			expr = field.Formula
		}
		if expr == "" {
			continue
		}

		evalCtx := &EvalContext{
			Record:           record,
			ChildCollections: childCollections,
		}

		val, err := Evaluate(expr, evalCtx)
		if err != nil {
			continue
		}

		record[fieldName] = val
	}

	return nil
}

func (h *Hydrator) HydrateRecords(ctx context.Context, modelDef *parser.ModelDefinition, records []map[string]any) error {
	if modelDef == nil || len(records) == 0 {
		return nil
	}

	hasComputed := false
	for _, field := range modelDef.Fields {
		if field.Computed != "" || field.Formula != "" {
			hasComputed = true
			break
		}
	}
	if !hasComputed {
		return nil
	}

	for i := range records {
		if err := h.HydrateRecord(ctx, modelDef, records[i]); err != nil {
			continue
		}
	}

	return nil
}

func (h *Hydrator) loadChildren(ctx context.Context, childModel string, inverseField string, parentRecord map[string]any, parentModelDef *parser.ModelDefinition) ([]map[string]any, error) {
	parentID := resolveParentID(parentRecord, parentModelDef)
	if parentID == "" {
		return nil, nil
	}

	tableName := childModel + "s"
	var results []map[string]any
	query := h.db.WithContext(ctx).Table(tableName).Where("active = ?", true)
	query = query.Where(fmt.Sprintf("%s = ?", inverseField), parentID)

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	if results == nil {
		results = []map[string]any{}
	}

	if h.modelRegistry != nil {
		childDef, err := h.modelRegistry.Get(childModel)
		if err == nil && childDef != nil {
			hasChildComputed := false
			for _, f := range childDef.Fields {
				if f.Computed != "" || f.Formula != "" {
					hasChildComputed = true
					break
				}
			}
			if hasChildComputed {
				for i := range results {
					evalCtx := &EvalContext{
						Record:           results[i],
						ChildCollections: make(map[string][]map[string]any),
					}
					for fn, fd := range childDef.Fields {
						expr := fd.Computed
						if expr == "" {
							expr = fd.Formula
						}
						if expr == "" {
							continue
						}
						val, err := Evaluate(expr, evalCtx)
						if err != nil {
							continue
						}
						results[i][fn] = val
					}
				}
			}
		}
	}

	return results, nil
}

func resolveParentID(record map[string]any, modelDef *parser.ModelDefinition) string {
	if modelDef != nil && modelDef.PrimaryKey != nil {
		if modelDef.PrimaryKey.Field != "" {
			if v, ok := record[modelDef.PrimaryKey.Field]; ok {
				return fmt.Sprintf("%v", v)
			}
		}
	}
	if v, ok := record["id"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
