package bridge

import (
	"context"
	"fmt"
	"time"

	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"gorm.io/gorm"
)

type executionBridge struct {
	db      *gorm.DB
	session Session
	current *ExecutionInfo
}

func newExecutionBridge(db *gorm.DB, session Session) *executionBridge {
	return &executionBridge{db: db, session: session}
}

func (e *executionBridge) Search(opts ExecutionSearchOptions) ([]map[string]any, error) {
	repo := persistence.NewGenericRepository(e.db, "process_executions")

	q := persistence.NewQuery()
	if opts.Process != "" {
		q.Where("process_name", "=", opts.Process)
	}
	if opts.Status != "" {
		q.Where("status", "=", opts.Status)
	}
	if opts.UserID != "" {
		q.Where("user_id", "=", opts.UserID)
	}
	if opts.Order != "" {
		q.Order(opts.Order, "desc")
	} else {
		q.Order("started_at", "desc")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	results, _, err := repo.FindAll(context.Background(), q, 1, limit)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func (e *executionBridge) Get(id string, opts ...GetOptions) (map[string]any, error) {
	repo := persistence.NewGenericRepository(e.db, "process_executions")

	record, err := repo.FindByID(context.Background(), id)
	if err != nil {
		return nil, nil
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			for _, inc := range opt.Include {
				if inc == "steps" {
					stepsRepo := persistence.NewGenericRepository(e.db, "process_execution_steps")
					q := persistence.NewQuery()
					q.Where("execution_id", "=", id)
					q.Order("step_index", "asc")
					steps, _, _ := stepsRepo.FindAll(context.Background(), q, 1, 1000)
					record["steps"] = steps
				}
			}
		}
	}

	return record, nil
}

func (e *executionBridge) Current() *ExecutionInfo {
	return e.current
}

func (e *executionBridge) Retry(id string) (map[string]any, error) {
	return nil, NewError(ErrInternalError, "retry not yet implemented")
}

func (e *executionBridge) Cancel(id string) error {
	repo := persistence.NewGenericRepository(e.db, "process_executions")
	return repo.Update(context.Background(), id, map[string]any{
		"status":      "cancelled",
		"finished_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// RecordExecution creates a process_execution record and returns its ID.
func RecordExecution(db *gorm.DB, processName, module, trigger, userID, mode string, input map[string]any) (string, error) {
	repo := persistence.NewGenericRepository(db, "process_executions")
	record, err := repo.Create(context.Background(), map[string]any{
		"process_name": processName,
		"module":       module,
		"trigger":      trigger,
		"status":       "running",
		"started_at":   time.Now().UTC().Format(time.RFC3339),
		"user_id":      userID,
		"input":        truncateJSON(input, 10240),
		"mode":         mode,
		"step_count":   0,
	})
	if err != nil {
		return "", err
	}
	id, _ := record["id"].(string)
	return id, nil
}

// RecordExecutionStep creates a process_execution_step record.
func RecordExecutionStep(db *gorm.DB, executionID string, stepIndex int, stepName, stepType, status string, durationMs int, input, output, stepError, meta map[string]any) error {
	repo := persistence.NewGenericRepository(db, "process_execution_steps")
	_, err := repo.Create(context.Background(), map[string]any{
		"execution_id": executionID,
		"step_index":   stepIndex,
		"step_name":    stepName,
		"step_type":    stepType,
		"status":       status,
		"started_at":   time.Now().UTC().Add(-time.Duration(durationMs) * time.Millisecond).Format(time.RFC3339),
		"duration_ms":  durationMs,
		"input":        truncateJSON(input, 10240),
		"output":       truncateJSON(output, 10240),
		"error":        truncateJSON(stepError, 10240),
		"meta":         meta,
	})
	return err
}

// FinishExecution updates a process_execution record with final status.
func FinishExecution(db *gorm.DB, executionID, status string, durationMs int, output, execError map[string]any, stepCount int) error {
	repo := persistence.NewGenericRepository(db, "process_executions")
	return repo.Update(context.Background(), executionID, map[string]any{
		"status":      status,
		"finished_at": time.Now().UTC().Format(time.RFC3339),
		"duration_ms": durationMs,
		"output":      truncateJSON(output, 10240),
		"error":       truncateJSON(execError, 10240),
		"step_count":  stepCount,
	})
}

func truncateJSON(data map[string]any, maxSize int) any {
	if data == nil {
		return nil
	}
	serialized := fmt.Sprintf("%v", data)
	if len(serialized) <= maxSize {
		return data
	}
	return map[string]any{"_truncated": true, "_size": len(serialized)}
}
