package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	infraModule "github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/spf13/cobra"
)

func publishCrudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish:crud <module> <model> <target>",
		Short: "Generate override files from auto-generated CRUD",
		Long: `Generate API and page override files from auto-generated CRUD definitions.
These files can then be customized.

Targets:
  all    - Generate both API and page files
  api    - Generate API override file only
  pages  - Generate all page files
  pages list  - Generate list page only
  pages form  - Generate form page only

Examples:
  bitcode publish:crud crm contact all
  bitcode publish:crud crm contact api
  bitcode publish:crud crm contact pages
  bitcode publish:crud crm contact pages list`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]
			modelName := args[1]
			target := args[2]
			subTarget := ""
			if len(args) > 3 {
				subTarget = args[3]
			}

			moduleDir := envOrDefault("MODULE_DIR", "modules")
			modPath := filepath.Join(moduleDir, moduleName)

			modelPath := filepath.Join(modPath, "models", modelName+".json")
			data, err := os.ReadFile(modelPath)
			if err != nil {
				return fmt.Errorf("model file not found: %s", modelPath)
			}

			modelDef, err := parser.ParseModel(data)
			if err != nil {
				return fmt.Errorf("failed to parse model: %w", err)
			}

			switch target {
			case "all":
				if err := generateAPIFile(modPath, moduleName, modelDef); err != nil {
					return err
				}
				if err := generatePageFile(modPath, moduleName, modelDef, "list"); err != nil {
					return err
				}
				if err := generatePageFile(modPath, moduleName, modelDef, "form"); err != nil {
					return err
				}
			case "api":
				if err := generateAPIFile(modPath, moduleName, modelDef); err != nil {
					return err
				}
			case "pages":
				if subTarget == "list" {
					return generatePageFile(modPath, moduleName, modelDef, "list")
				} else if subTarget == "form" {
					return generatePageFile(modPath, moduleName, modelDef, "form")
				}
				if err := generatePageFile(modPath, moduleName, modelDef, "list"); err != nil {
					return err
				}
				return generatePageFile(modPath, moduleName, modelDef, "form")
			default:
				return fmt.Errorf("unknown target: %s (use: all, api, pages)", target)
			}
			return nil
		},
	}
	return cmd
}

func generateAPIFile(modPath, moduleName string, modelDef *parser.ModelDefinition) error {
	apiDef := infraModule.GenerateAPIFromModel(modelDef, moduleName)
	if apiDef == nil {
		apiDef = &parser.APIDefinition{
			Name:     modelDef.Name + "_api",
			Model:    modelDef.Name,
			AutoCRUD: true,
			Auth:     true,
			BasePath: fmt.Sprintf("/api/v1/%s/%ss", moduleName, modelDef.Name),
		}
	}

	content := map[string]any{
		"name":      apiDef.Name,
		"model":     apiDef.Model,
		"module":    moduleName,
		"auto_crud": apiDef.AutoCRUD,
		"auth":      apiDef.Auth,
		"base_path": apiDef.BasePath,
		"endpoints": []any{},
	}

	return writeJSONFile(filepath.Join(modPath, "apis", modelDef.Name+"_api.json"), content)
}

func generatePageFile(modPath, moduleName string, modelDef *parser.ModelDefinition, pageType string) error {
	dir := filepath.Join(modPath, "pages")

	var content map[string]any
	switch pageType {
	case "list":
		fields := make([]string, 0)
		for name, field := range modelDef.Fields {
			if field.Type == parser.FieldText || field.Type == parser.FieldRichText ||
				field.Type == parser.FieldOne2Many || field.Type == parser.FieldJSON {
				continue
			}
			fields = append(fields, name)
		}
		content = map[string]any{
			"name":   modelDef.Name + "_list",
			"type":   "list",
			"model":  modelDef.Name,
			"module": moduleName,
			"title":  modelDef.Label,
			"fields": fields,
		}
	case "form":
		var layout []map[string]any
		for name, field := range modelDef.Fields {
			width := 6
			if field.Type == parser.FieldText || field.Type == parser.FieldRichText {
				width = 12
			}
			layout = append(layout, map[string]any{
				"row": []map[string]any{{"field": name, "width": width}},
			})
		}
		content = map[string]any{
			"name":   modelDef.Name + "_form",
			"type":   "form",
			"model":  modelDef.Name,
			"module": moduleName,
			"title":  modelDef.Label,
			"layout": layout,
		}
	default:
		return fmt.Errorf("unknown page type: %s", pageType)
	}

	return writeJSONFile(filepath.Join(dir, modelDef.Name+"_"+pageType+".json"), content)
}

func writeJSONFile(path string, content any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  [SKIP] %s (already exists)\n", path)
		return nil
	}

	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	fmt.Printf("  [CREATED] %s\n", path)
	return nil
}
