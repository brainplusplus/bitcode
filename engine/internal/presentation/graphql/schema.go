package graphql

import (
	"github.com/graphql-go/graphql"
	"github.com/jinzhu/inflection"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type SchemaBuilder struct {
	models    []*parser.ModelDefinition
	types     map[string]*graphql.Object
	resolver  *Resolver
}

func NewSchemaBuilder(resolver *Resolver) *SchemaBuilder {
	return &SchemaBuilder{
		types:    make(map[string]*graphql.Object),
		resolver: resolver,
	}
}

func (b *SchemaBuilder) AddModel(model *parser.ModelDefinition) {
	if model.API == nil || !model.API.Protocols.GraphQL {
		return
	}
	b.models = append(b.models, model)
}

func (b *SchemaBuilder) Build() (*graphql.Schema, error) {
	for _, model := range b.models {
		b.types[model.Name] = b.buildObjectType(model)
	}

	queryFields := graphql.Fields{}
	mutationFields := graphql.Fields{}

	for _, model := range b.models {
		b.addQueryFields(model, queryFields)
		b.addMutationFields(model, mutationFields)
	}

	if len(queryFields) == 0 {
		queryFields["_empty"] = &graphql.Field{
			Type:    graphql.String,
			Resolve: func(p graphql.ResolveParams) (any, error) { return "no models with graphql enabled", nil },
		}
	}

	schemaConfig := graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "Query",
			Fields: queryFields,
		}),
	}

	if len(mutationFields) > 0 {
		schemaConfig.Mutation = graphql.NewObject(graphql.ObjectConfig{
			Name:   "Mutation",
			Fields: mutationFields,
		})
	}

	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func (b *SchemaBuilder) buildObjectType(model *parser.ModelDefinition) *graphql.Object {
	fields := graphql.Fields{
		"id": &graphql.Field{Type: graphql.String},
		"created_at": &graphql.Field{Type: graphql.String},
		"updated_at": &graphql.Field{Type: graphql.String},
	}

	for name, field := range model.Fields {
		gqlType := fieldTypeToGraphQL(field.Type)
		if gqlType == nil {
			continue
		}
		f := &graphql.Field{Type: gqlType}
		if field.Required {
			f.Type = graphql.NewNonNull(gqlType)
		}
		fields[name] = f
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   model.Name,
		Fields: fields,
	})
}

func (b *SchemaBuilder) addQueryFields(model *parser.ModelDefinition, fields graphql.Fields) {
	objType := b.types[model.Name]

	listType := graphql.NewObject(graphql.ObjectConfig{
		Name: model.Name + "_list_response",
		Fields: graphql.Fields{
			"data":        &graphql.Field{Type: graphql.NewList(objType)},
			"total":       &graphql.Field{Type: graphql.Int},
			"page":        &graphql.Field{Type: graphql.Int},
			"page_size":   &graphql.Field{Type: graphql.Int},
			"total_pages": &graphql.Field{Type: graphql.Int},
		},
	})

	fields[model.Name+"_list"] = &graphql.Field{
		Type: listType,
		Args: graphql.FieldConfigArgument{
			"page":      &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 1},
			"page_size": &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 20},
			"q":         &graphql.ArgumentConfig{Type: graphql.String},
		},
		Resolve: b.resolver.List(model.Name),
	}

	fields[model.Name] = &graphql.Field{
		Type: objType,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: b.resolver.Read(model.Name),
	}
}

func (b *SchemaBuilder) addMutationFields(model *parser.ModelDefinition, fields graphql.Fields) {
	objType := b.types[model.Name]

	inputFields := graphql.InputObjectConfigFieldMap{}
	for name, field := range model.Fields {
		gqlType := fieldTypeToGraphQL(field.Type)
		if gqlType == nil {
			continue
		}
		inputFields[name] = &graphql.InputObjectFieldConfig{Type: gqlType}
	}

	inputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   model.Name + "_input",
		Fields: inputFields,
	})

	fields["create_"+model.Name] = &graphql.Field{
		Type: objType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(inputType)},
		},
		Resolve: b.resolver.Create(model.Name),
	}

	fields["update_"+model.Name] = &graphql.Field{
		Type: objType,
		Args: graphql.FieldConfigArgument{
			"id":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(inputType)},
		},
		Resolve: b.resolver.Update(model.Name),
	}

	fields["delete_"+model.Name] = &graphql.Field{
		Type: graphql.NewObject(graphql.ObjectConfig{
			Name: "delete_" + model.Name + "_response",
			Fields: graphql.Fields{
				"message": &graphql.Field{Type: graphql.String},
			},
		}),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
		},
		Resolve: b.resolver.Delete(model.Name),
	}
}

func fieldTypeToGraphQL(ft parser.FieldType) graphql.Output {
	switch ft {
	case parser.FieldString, parser.FieldEmail, parser.FieldSmallText, parser.FieldText,
		parser.FieldRichText, parser.FieldMarkdown, parser.FieldHTML, parser.FieldCode,
		parser.FieldPassword, parser.FieldBarcode, parser.FieldColor, parser.FieldSignature:
		return graphql.String
	case parser.FieldInteger, parser.FieldRating:
		return graphql.Int
	case parser.FieldFloat, parser.FieldDecimal, parser.FieldCurrency, parser.FieldPercent:
		return graphql.Float
	case parser.FieldBoolean, parser.FieldToggle:
		return graphql.Boolean
	case parser.FieldDate, parser.FieldDatetime, parser.FieldTime, parser.FieldDuration:
		return graphql.String
	case parser.FieldSelection, parser.FieldRadio:
		return graphql.String
	case parser.FieldMany2One, parser.FieldDynamicLink:
		return graphql.String
	case parser.FieldFile, parser.FieldImage:
		return graphql.String
	case parser.FieldJSON, parser.FieldGeolocation:
		return graphql.String
	case parser.FieldOne2Many, parser.FieldMany2Many:
		return nil
	default:
		return graphql.String
	}
}

func pluralizeModel(name string) string {
	return inflection.Plural(name)
}


