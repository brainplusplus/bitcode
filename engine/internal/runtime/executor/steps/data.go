package steps

import (
	"context"
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
	"gorm.io/gorm"
)

type DataHandler struct {
	DB       *gorm.DB
	Resolver interface{ TableName(string) string }
}

func (h *DataHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	if step.Model == "" {
		return fmt.Errorf("data step requires a model")
	}

	tableName := step.Model
	if h.Resolver != nil {
		tableName = h.Resolver.TableName(step.Model)
	}
	repo := persistence.NewGenericRepository(h.DB, tableName)
	repo.SetRevisionRepo(persistence.NewDataRevisionRepository(h.DB))
	repo.SetModelName(step.Model)

	switch step.Type {
	case parser.StepQuery:
		return h.executeQuery(ctx, execCtx, step, repo)
	case parser.StepCreate:
		return h.executeCreate(ctx, execCtx, step, repo)
	case parser.StepUpdate:
		return h.executeUpdate(ctx, execCtx, step, repo)
	case parser.StepDelete:
		return h.executeDelete(ctx, execCtx, step, repo)
	case parser.StepUpsert:
		return h.executeUpsert(ctx, execCtx, step, repo)
	case parser.StepCount:
		return h.executeCount(ctx, execCtx, step, repo)
	case parser.StepSum:
		return h.executeSum(ctx, execCtx, step, repo)
	default:
		return fmt.Errorf("unknown data step type: %s", step.Type)
	}
}

func (h *DataHandler) executeQuery(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	results, _, err := repo.FindAll(ctx, persistence.QueryFromDomain(step.Domain), 1, 100)
	if err != nil {
		return err
	}

	varName := step.Into
	if varName == "" {
		varName = "result"
	}
	execCtx.Variables[varName] = results
	execCtx.Result = results
	return nil
}

func (h *DataHandler) executeCreate(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	data := make(map[string]any)
	for k, v := range step.Set {
		data[k] = v
	}
	result, err := repo.Create(ctx, data)
	if err != nil {
		return err
	}
	execCtx.Result = result
	return nil
}

func (h *DataHandler) executeUpdate(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	id, ok := execCtx.Input["id"].(string)
	if !ok {
		return fmt.Errorf("update requires an id in input")
	}
	return repo.Update(ctx, id, step.Set)
}

func (h *DataHandler) executeDelete(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	id, ok := execCtx.Input["id"].(string)
	if !ok {
		return fmt.Errorf("delete requires an id in input")
	}
	return repo.Delete(ctx, id)
}

func (h *DataHandler) executeUpsert(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	data := make(map[string]any)
	for k, v := range step.Set {
		data[k] = v
	}
	result, err := repo.Upsert(ctx, data, step.Unique)
	if err != nil {
		return err
	}
	execCtx.Result = result
	return nil
}

func (h *DataHandler) executeCount(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	query := persistence.QueryFromDomain(step.Domain)
	count, err := repo.Count(ctx, query)
	if err != nil {
		return err
	}
	varName := step.Into
	if varName == "" {
		varName = "result"
	}
	execCtx.Variables[varName] = count
	execCtx.Result = count
	return nil
}

func (h *DataHandler) executeSum(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	if step.SumField == "" {
		return fmt.Errorf("sum step requires a sum_field")
	}
	query := persistence.QueryFromDomain(step.Domain)
	sum, err := repo.Sum(ctx, step.SumField, query)
	if err != nil {
		return err
	}
	varName := step.Into
	if varName == "" {
		varName = "result"
	}
	execCtx.Variables[varName] = sum
	execCtx.Result = sum
	return nil
}
