package module

import (
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func GenerateAPIFromModel(model *parser.ModelDefinition, moduleName string) *parser.APIDefinition {
	if model.API == nil {
		return nil
	}

	apiDef := &parser.APIDefinition{
		Name:     model.Name + "_api",
		Model:    model.Name,
		AutoCRUD: model.API.AutoCRUD,
		Auth:     model.API.Auth,
	}

	if model.API.IsSoftDelete() {
		sd := true
		apiDef.SoftDelete = &sd
	}

	if len(model.API.Search) > 0 {
		apiDef.Search = model.API.Search
	} else if len(model.SearchField) > 0 {
		apiDef.Search = model.SearchField
	}

	if moduleName != "" {
		apiDef.BasePath = "/api/v1/" + moduleName + "/" + pluralize(model.Name)
	}

	return apiDef
}

func MergeAPIs(autoAPIs []*parser.APIDefinition, overrideAPIs []*parser.APIDefinition) []*parser.APIDefinition {
	apiByModel := make(map[string]*parser.APIDefinition)

	for _, a := range autoAPIs {
		if a.Model != "" {
			apiByModel[a.Model] = a
		}
	}

	for _, o := range overrideAPIs {
		if o.Model != "" {
			if base, exists := apiByModel[o.Model]; exists {
				merged := mergeAPIDefinition(base, o)
				apiByModel[o.Model] = merged
			} else {
				apiByModel[o.Model] = o
			}
		} else {
			key := "__custom__" + o.Name
			apiByModel[key] = o
		}
	}

	result := make([]*parser.APIDefinition, 0, len(apiByModel))
	for _, a := range apiByModel {
		result = append(result, a)
	}
	return result
}

func mergeAPIDefinition(base *parser.APIDefinition, override *parser.APIDefinition) *parser.APIDefinition {
	merged := *base

	if override.BasePath != "" {
		merged.BasePath = override.BasePath
	}
	if override.Workflow != "" {
		merged.Workflow = override.Workflow
	}
	if override.Actions != nil {
		if merged.Actions == nil {
			merged.Actions = make(map[string]parser.WorkflowActionDefinition)
		}
		for k, v := range override.Actions {
			merged.Actions[k] = v
		}
	}
	if len(override.Search) > 0 {
		merged.Search = override.Search
	}
	if override.SoftDelete != nil {
		merged.SoftDelete = override.SoftDelete
	}

	if len(override.Endpoints) > 0 {
		endpointMap := make(map[string]parser.EndpointDefinition)
		for _, ep := range merged.Endpoints {
			key := ep.Method + " " + ep.Path
			endpointMap[key] = ep
		}
		for _, ep := range override.Endpoints {
			key := ep.Method + " " + ep.Path
			endpointMap[key] = ep
		}
		merged.Endpoints = make([]parser.EndpointDefinition, 0, len(endpointMap))
		for _, ep := range endpointMap {
			merged.Endpoints = append(merged.Endpoints, ep)
		}
	}

	return &merged
}

func pluralize(name string) string {
	if len(name) == 0 {
		return name
	}
	last := name[len(name)-1]
	switch last {
	case 's', 'x', 'z':
		return name + "es"
	case 'y':
		if len(name) > 1 {
			prev := name[len(name)-2]
			if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
				return name[:len(name)-1] + "ies"
			}
		}
		return name + "s"
	default:
		return name + "s"
	}
}
