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

func resolveQueryFromArgs(args map[string]any) *persistence.Query {
	if oql, ok := args["oql"].(string); ok && oql != "" {
		q, _, err := persistence.ParseOQL(oql)
		if err == nil && q != nil {
			return q
		}
	}
	return resolveQueryFromArgs(args)
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
		query := resolveQueryFromArgs(args)
		page := intFromMap(args, "page", 1)
		pageSize := intFromMap(args, "page_size", 20)
		records, total, err := repo.FindAll(ctx, query, page, pageSize)
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": records, "total": total, "page": page, "page_size": pageSize}, nil

	case "Paginate":
		query := resolveQueryFromArgs(args)
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

	case "FindActive":
		id, _ := args["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("id required for %s", operation)
		}
		return repo.FindActive(ctx, id)

	case "FindAllActive":
		query := resolveQueryFromArgs(args)
		page := intFromMap(args, "page", 1)
		pageSize := intFromMap(args, "page_size", 20)
		records, total, err := repo.FindAllActive(ctx, query, page, pageSize)
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": records, "total": total, "page": page, "page_size": pageSize}, nil

	case "PaginateActive":
		query := resolveQueryFromArgs(args)
		page := intFromMap(args, "page", 1)
		pageSize := intFromMap(args, "page_size", 20)
		records, total, err := repo.FindAllActive(ctx, query, page, pageSize)
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
		query := resolveQueryFromArgs(args)
		count, err := repo.Count(ctx, query)
		if err != nil {
			return nil, err
		}
		return count, nil

	case "CountActive":
		query := resolveQueryFromArgs(args)
		count, err := repo.CountActive(ctx, query)
		if err != nil {
			return nil, err
		}
		return count, nil

	case "Sum":
		field, _ := args["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("field required for Sum")
		}
		query := resolveQueryFromArgs(args)
		sum, err := repo.Sum(ctx, field, query)
		if err != nil {
			return nil, err
		}
		return sum, nil

	case "SumActive":
		field, _ := args["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("field required for SumActive")
		}
		query := resolveQueryFromArgs(args)
		sum, err := repo.SumActive(ctx, field, query)
		if err != nil {
			return nil, err
		}
		return sum, nil

	case "Avg":
		field, _ := args["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("field required for Avg")
		}
		query := resolveQueryFromArgs(args)
		avg, err := repo.Avg(ctx, field, query)
		if err != nil {
			return nil, err
		}
		return avg, nil

	case "Min":
		field, _ := args["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("field required for Min")
		}
		query := resolveQueryFromArgs(args)
		min, err := repo.Min(ctx, field, query)
		if err != nil {
			return nil, err
		}
		return min, nil

	case "Max":
		field, _ := args["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("field required for Max")
		}
		query := resolveQueryFromArgs(args)
		max, err := repo.Max(ctx, field, query)
		if err != nil {
			return nil, err
		}
		return max, nil

	case "Pluck":
		field, _ := args["field"].(string)
		if field == "" {
			return nil, fmt.Errorf("field required for Pluck")
		}
		query := resolveQueryFromArgs(args)
		values, err := repo.Pluck(ctx, field, query)
		if err != nil {
			return nil, err
		}
		return values, nil

	case "Exists":
		query := resolveQueryFromArgs(args)
		exists, err := repo.Exists(ctx, query)
		if err != nil {
			return nil, err
		}
		return exists, nil

	case "Aggregate":
		query := resolveQueryFromArgs(args)
		results, err := repo.Aggregate(ctx, query)
		if err != nil {
			return nil, err
		}
		return results, nil

	case "WithTrashed":
		query := resolveQueryFromArgs(args)
		page := intFromMap(args, "page", 1)
		pageSize := intFromMap(args, "page_size", 20)
		records, total, err := repo.FindAllWithTrashed(ctx, query, page, pageSize)
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": records, "total": total, "page": page, "page_size": pageSize}, nil

	case "OnlyTrashed":
		query := resolveQueryFromArgs(args)
		page := intFromMap(args, "page", 1)
		pageSize := intFromMap(args, "page_size", 20)
		records, total, err := repo.FindAllOnlyTrashed(ctx, query, page, pageSize)
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": records, "total": total, "page": page, "page_size": pageSize}, nil

	case "Increment":
		id, _ := args["id"].(string)
		field, _ := args["field"].(string)
		value := intFromMap(args, "value", 1)
		if id == "" || field == "" {
			return nil, fmt.Errorf("id and field required for Increment")
		}
		return nil, repo.Increment(ctx, id, field, value)

	case "Decrement":
		id, _ := args["id"].(string)
		field, _ := args["field"].(string)
		value := intFromMap(args, "value", 1)
		if id == "" || field == "" {
			return nil, fmt.Errorf("id and field required for Decrement")
		}
		return nil, repo.Decrement(ctx, id, field, value)

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
