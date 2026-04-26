package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/spf13/cobra"
)

func seedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Data migration and seeding",
		Long: `Run data migrations (seeders) defined in module migrations/ directories.

Supports JSON, CSV, XLSX, and XML source files with optional custom processors.
Migrations are tracked in the ir_migration table and only run once.`,
	}

	cmd.AddCommand(seedRunCmd())
	cmd.AddCommand(seedRollbackCmd())
	cmd.AddCommand(seedStatusCmd())
	cmd.AddCommand(seedFreshCmd())
	cmd.AddCommand(seedCreateCmd())

	return cmd
}

func seedRunCmd() *cobra.Command {
	var moduleName string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run pending data migrations",
		Long:  "Execute all pending data migrations. Migrations that have already run are skipped.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}

			if app.MigrationEngine == nil {
				return fmt.Errorf("migration engine not available (requires SQL database)")
			}

			moduleDir := envOrDefault("MODULE_DIR", "modules")
			ctx := context.Background()
			totalRecords := 0

			if moduleName != "" {
				modPath := filepath.Join(moduleDir, moduleName)
				modFile := filepath.Join(modPath, "module.json")
				modDef, err := parser.ParseModuleFile(modFile)
				if err != nil {
					return fmt.Errorf("module %s not found: %w", moduleName, err)
				}

				migrations, err := module.CollectModuleMigrations(modPath, modDef.Migrations)
				if err != nil {
					return fmt.Errorf("failed to discover migrations: %w", err)
				}

				count, err := app.MigrationEngine.RunUp(ctx, modPath, moduleName, migrations)
				if err != nil {
					return err
				}
				totalRecords += count
			} else {
				ordered, err := module.CollectAllModuleMigrationsOrdered(moduleDir, app.ModuleOrder())
				if err != nil {
					return fmt.Errorf("failed to discover migrations: %w", err)
				}

				for _, om := range ordered {
					count, err := app.MigrationEngine.RunUp(ctx, om.Path, om.Module, om.Migrations)
					if err != nil {
						return fmt.Errorf("module %s: %w", om.Module, err)
					}
					totalRecords += count
				}
			}

			if totalRecords > 0 {
				fmt.Printf("Seeded %d records.\n", totalRecords)
			} else {
				fmt.Println("Nothing to seed.")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module", "m", "", "Run migrations for specific module only")
	return cmd
}

func seedRollbackCmd() *cobra.Command {
	var moduleName string
	var steps int

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback the last batch of data migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}

			if app.MigrationEngine == nil {
				return fmt.Errorf("migration engine not available (requires SQL database)")
			}

			moduleDir := envOrDefault("MODULE_DIR", "modules")
			ctx := context.Background()

			for i := 0; i < steps; i++ {
				ordered, err := module.CollectAllModuleMigrationsOrdered(moduleDir, app.ModuleOrder())
				if err != nil {
					return err
				}

				var allFiles []*parser.MigrationFile
				for _, om := range ordered {
					allFiles = append(allFiles, om.Migrations...)
				}

				modPath := moduleDir
				if moduleName != "" {
					modPath = filepath.Join(moduleDir, moduleName)
				}

				if err := app.MigrationEngine.RollbackBatch(ctx, modPath, moduleName, allFiles); err != nil {
					return err
				}
			}

			fmt.Printf("Rolled back %d batch(es).\n", steps)
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module", "m", "", "Rollback migrations for specific module only")
	cmd.Flags().IntVar(&steps, "steps", 1, "Number of batches to rollback")
	return cmd
}

func seedStatusCmd() *cobra.Command {
	var moduleName string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}

			if app.MigrationEngine == nil {
				return fmt.Errorf("migration engine not available (requires SQL database)")
			}

			entries, err := app.MigrationEngine.Tracker().Status()
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("No migrations have been run.")
				return nil
			}

			fmt.Printf("%-5s %-30s %-15s %-15s %-8s %-10s %-10s %s\n",
				"BATCH", "NAME", "MODULE", "MODEL", "RECORDS", "STATUS", "DURATION", "RAN AT")
			fmt.Println(strings.Repeat("-", 120))

			for _, e := range entries {
				if moduleName != "" && e.Module != moduleName {
					continue
				}
				fmt.Printf("%-5d %-30s %-15s %-15s %-8d %-10s %-10s %s\n",
					e.Batch, e.Name, e.Module, e.Model, e.Records, e.Status, e.Duration, e.RanAt)
			}

			moduleDir := envOrDefault("MODULE_DIR", "modules")
			ordered, _ := module.CollectAllModuleMigrationsOrdered(moduleDir, app.ModuleOrder())
			for _, om := range ordered {
				if moduleName != "" && om.Module != moduleName {
					continue
				}
				var names []string
				for _, m := range om.Migrations {
					names = append(names, m.Name)
				}
				pending := app.MigrationEngine.Tracker().GetPending(om.Module, names)
				if len(pending) > 0 {
					fmt.Printf("\nPending (%s): %s\n", om.Module, strings.Join(pending, ", "))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module", "m", "", "Show status for specific module only")
	return cmd
}

func seedFreshCmd() *cobra.Command {
	var moduleName string

	cmd := &cobra.Command{
		Use:   "fresh",
		Short: "Reset and re-run all data migrations",
		Long:  "Drop all migration records and re-run all migrations from scratch.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}

			if app.MigrationEngine == nil {
				return fmt.Errorf("migration engine not available (requires SQL database)")
			}

			if err := app.MigrationEngine.Tracker().Reset(moduleName); err != nil {
				return fmt.Errorf("failed to reset migrations: %w", err)
			}
			fmt.Println("Migration records cleared.")

			moduleDir := envOrDefault("MODULE_DIR", "modules")
			ctx := context.Background()
			totalRecords := 0

			ordered, err := module.CollectAllModuleMigrationsOrdered(moduleDir, app.ModuleOrder())
			if err != nil {
				return err
			}

			for _, om := range ordered {
				if moduleName != "" && om.Module != moduleName {
					continue
				}
				count, err := app.MigrationEngine.RunUp(ctx, om.Path, om.Module, om.Migrations)
				if err != nil {
					return fmt.Errorf("module %s: %w", om.Module, err)
				}
				totalRecords += count
			}

			fmt.Printf("Fresh seed complete: %d records.\n", totalRecords)
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module", "m", "", "Fresh seed for specific module only")
	return cmd
}

func seedCreateCmd() *cobra.Command {
	var moduleName string
	var sourceType string
	var model string

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if moduleName == "" {
				return fmt.Errorf("--module is required")
			}
			if model == "" {
				return fmt.Errorf("--model is required")
			}

			moduleDir := envOrDefault("MODULE_DIR", "modules")
			migDir := filepath.Join(moduleDir, moduleName, "migrations")
			os.MkdirAll(migDir, 0755)

			timestamp := migrationTimestamp()
			filename := fmt.Sprintf("%s_%s.json", timestamp, name)
			filePath := filepath.Join(migDir, filename)

			if sourceType == "" {
				sourceType = "json"
			}

			ext := sourceType
			if ext == "xlsx" {
				ext = "xlsx"
			}

			deps := discoverModelDependencies(filepath.Join(moduleDir, moduleName), model)

			migDef := map[string]any{
				"name":  name,
				"model": model,
				"source": map[string]any{
					"type": sourceType,
					"file": fmt.Sprintf("data/%s.%s", name, ext),
				},
				"options": map[string]any{
					"batch_size":  100,
					"on_conflict": "skip",
				},
			}
			if len(deps) > 0 {
				migDef["depends_on"] = deps
			}

			content, _ := json.MarshalIndent(migDef, "", "  ")

			if err := os.WriteFile(filePath, content, 0644); err != nil {
				return fmt.Errorf("failed to create migration file: %w", err)
			}

			fmt.Printf("Created migration: %s\n", filePath)
			if len(deps) > 0 {
				fmt.Printf("Auto-detected depends_on: %v\n", deps)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module", "m", "", "Module name (required)")
	cmd.Flags().StringVarP(&sourceType, "type", "t", "json", "Source type (json, csv, xlsx, xml)")
	cmd.Flags().StringVar(&model, "model", "", "Target model name (required)")
	return cmd
}

func discoverModelDependencies(modulePath string, modelName string) []string {
	modelsDir := filepath.Join(modulePath, "models")
	modelFile := filepath.Join(modelsDir, modelName+".json")

	data, err := os.ReadFile(modelFile)
	if err != nil {
		return nil
	}

	var modelDef struct {
		Fields map[string]struct {
			Type  string `json:"type"`
			Model string `json:"model"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(data, &modelDef); err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var deps []string
	for _, field := range modelDef.Fields {
		if field.Type == "many2one" && field.Model != "" && field.Model != modelName {
			if !seen[field.Model] {
				seen[field.Model] = true
				deps = append(deps, "seed_"+field.Model)
			}
		}
	}
	return deps
}

func migrationTimestamp() string {
	now := time.Now()
	return fmt.Sprintf("%04d%02d%02d_%02d%02d%02d",
		now.Year(), int(now.Month()), now.Day(),
		now.Hour(), now.Minute(), now.Second())
}
