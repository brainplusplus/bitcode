package steps

import (
	"context"
	"fmt"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"github.com/bitcode-engine/engine/internal/runtime/executor"
	"gorm.io/gorm"
)

type DataHandler struct {
	DB *gorm.DB
}

func (h *DataHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	if step.Model == "" {
		return fmt.Errorf("data step requires a model")
	}

	repo := persistence.NewGenericRepository(h.DB, step.Model+"s")
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
	default:
		return fmt.Errorf("unknown data step type: %s", step.Type)
	}
}

func (h *DataHandler) executeQuery(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition, repo *persistence.GenericRepository) error {
	results, _, err := repo.FindAll(ctx, step.Domain, 1, 100)
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
