package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type EndpointDefinition struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Action      string   `json:"action,omitempty"`
	Handler     string   `json:"handler,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type WorkflowActionDefinition struct {
	Transition string `json:"transition"`
	Permission string `json:"permission"`
}

type PaginationConfig struct {
	PageSize int `json:"page_size,omitempty"`
	Max      int `json:"max,omitempty"`
}

type APIDefinition struct {
	Name       string                              `json:"name"`
	Model      string                              `json:"model,omitempty"`
	BasePath   string                              `json:"base_path,omitempty"`
	Auth       bool                                `json:"auth,omitempty"`
	AutoCRUD   bool                                `json:"auto_crud,omitempty"`
	SoftDelete *bool                               `json:"soft_delete,omitempty"`
	Workflow   string                              `json:"workflow,omitempty"`
	Actions    map[string]WorkflowActionDefinition `json:"actions,omitempty"`
	Endpoints  []EndpointDefinition                `json:"endpoints,omitempty"`
	Pagination PaginationConfig                    `json:"pagination,omitempty"`
	Search     []string                            `json:"search,omitempty"`
}

func (a *APIDefinition) GetBasePath() string {
	if a.BasePath != "" {
		return a.BasePath
	}
	if a.Model != "" {
		return "/api/" + a.Model + "s"
	}
	return "/api/" + a.Name
}

func (a *APIDefinition) IsSoftDelete() bool {
	if a.SoftDelete != nil {
		return *a.SoftDelete
	}
	return true
}

func (a *APIDefinition) GetPageSize() int {
	if a.Pagination.PageSize > 0 {
		return a.Pagination.PageSize
	}
	return 20
}

func (a *APIDefinition) ExpandAutoCRUD() []EndpointDefinition {
	if !a.AutoCRUD || a.Model == "" {
		return a.Endpoints
	}

	perm := func(action string) []string {
		return []string{a.Model + "." + action}
	}

	endpoints := []EndpointDefinition{
		{Method: "GET", Path: "/", Action: "list", Permissions: perm("read")},
		{Method: "GET", Path: "/:id", Action: "read", Permissions: perm("read")},
		{Method: "POST", Path: "/", Action: "create", Permissions: perm("create")},
		{Method: "PUT", Path: "/:id", Action: "update", Permissions: perm("write")},
		{Method: "DELETE", Path: "/:id", Action: "delete", Permissions: perm("delete")},
	}

	for actionName, actionDef := range a.Actions {
		endpoints = append(endpoints, EndpointDefinition{
			Method:      "POST",
			Path:        "/:id/" + actionName,
			Action:      actionName,
			Permissions: []string{actionDef.Permission},
		})
	}

	endpoints = append(endpoints, a.Endpoints...)
	return endpoints
}

func ParseAPI(data []byte) (*APIDefinition, error) {
	var api APIDefinition
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("invalid API JSON: %w", err)
	}
	if api.Name == "" {
		return nil, fmt.Errorf("API name is required")
	}
	if api.AutoCRUD && api.Model == "" {
		return nil, fmt.Errorf("auto_crud requires a model")
	}
	for i, ep := range api.Endpoints {
		if ep.Method == "" {
			return nil, fmt.Errorf("endpoint %d must have a method", i)
		}
		if ep.Path == "" {
			return nil, fmt.Errorf("endpoint %d must have a path", i)
		}
	}
	return &api, nil
}

func ParseAPIFile(path string) (*APIDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read API file %s: %w", path, err)
	}
	return ParseAPI(data)
}
