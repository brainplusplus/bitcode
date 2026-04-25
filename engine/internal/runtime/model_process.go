package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

type ModelProcessRegistry struct {
	repos map[string]persistence.Repository
	mu    sync.RWMutex
}

func NewModelProcessRegistry() *ModelProcessRegistry {
	return &ModelProcessRegistry{
		repos: make(map[string]persistence.Repository),
	}
}

func (r *ModelProcessRegistry) Register(modelName string, repo persistence.Repository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.repos[modelName] = repo
}

func (r *ModelProcessRegistry) Execute(ctx context.Context, processName string, args map[string]any) (any, error) {
	modelName, operation, err := parseModelProcess(processName)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	repo, ok := r.repos[modelName]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("model %q not registered", modelName)
	}

	switch operation {
	case "Get", "Find":
		id, _ := args["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("id required for %s", operation)
		}
		return repo.FindByID(ctx, id)

	case "GetAll", "FindAll":
		query := persistence.ParseQueryFromMap(args)
		page := intFromMap(args, "page", 1)
		pageSize := intFromMap(args, "page_size", 20)
		records, total, err := repo.FindAll(ctx, query, page, pageSize)
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": records, "total": total, "page": page, "page_size": pageSize}, nil

	case "Paginate":
		query := persistence.ParseQueryFromMap(args)
		page := intFromMap(args, "page", 1)
		pageSize := intFromMap(args, "page_size", 20)
		records, total, err := repo.FindAll(ctx, query, page, pageSize)
		if err != nil {
			return nil, err
		}
		totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
		return map[string]any{
			"data":        records,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		}, nil

	case "Create":
		data, _ := args["data"].(map[string]any)
		if data == nil {
			data = args
			delete(data, "model")
		}
		return repo.Create(ctx, data)

	case "Update":
		id, _ := args["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("id required for Update")
		}
		data, _ := args["data"].(map[string]any)
		if data == nil {
			data = make(map[string]any)
			for k, v := range args {
				if k != "id" && k != "model" {
					data[k] = v
				}
			}
		}
		return nil, repo.Update(ctx, id, data)

	case "Delete":
		id, _ := args["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("id required for Delete")
		}
		return nil, repo.Delete(ctx, id)

	case "Upsert":
		data, _ := args["data"].(map[string]any)
		if data == nil {
			data = args
		}
		var uniqueFields []string
		if uf, ok := args["unique"].([]any); ok {
			for _, f := range uf {
				if s, ok := f.(string); ok {
					uniqueFields = append(uniqueFields, s)
				}
			}
		}
		if ufs, ok := args["unique"].([]string); ok {
			uniqueFields = ufs
		}
		return repo.Upsert(ctx, data, uniqueFields)

	case "Count":
		query := persistence.ParseQueryFromMap(args)
		count, err := repo.Count(ctx, query)
		if err != nil {
			return nil, err
		}
		return count, nil

	case "Sum":
		field, _ := args["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("field required for Sum")
		}
		query := persistence.ParseQueryFromMap(args)
		sum, err := repo.Sum(ctx, field, query)
		if err != nil {
			return nil, err
		}
		return sum, nil

	default:
		return nil, fmt.Errorf("unknown model operation %q", operation)
	}
}

func parseModelProcess(name string) (string, string, error) {
	// format: models.{model_name}.{operation}
	if len(name) < 9 || name[:7] != "models." {
		return "", "", fmt.Errorf("invalid model process name %q", name)
	}
	rest := name[7:]
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == '.' {
			return rest[:i], rest[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid model process name %q: missing operation", name)
}

func intFromMap(m map[string]any, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int64:
			return int(val)
		}
	}
	return defaultVal
}
