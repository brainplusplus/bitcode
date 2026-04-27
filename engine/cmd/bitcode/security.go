package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitcode-framework/bitcode/internal"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	infraModule "github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func securityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Manage security definitions (groups, ACL, record rules)",
	}

	cmd.AddCommand(securityLoadCmd())
	cmd.AddCommand(securityExportCmd())
	cmd.AddCommand(securityDiffCmd())
	cmd.AddCommand(securityValidateCmd())
	cmd.AddCommand(securityHistoryCmd())

	return cmd
}

func securityLoadCmd() *cobra.Command {
	var force bool
	return &cobra.Command{
		Use:   "load [module]",
		Short: "Load securities/*.json into database",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDBForCLI()
			if err != nil {
				return err
			}
			loader := infraModule.NewSecurityLoader(db)
			moduleDir := envOrDefault("MODULE_DIR", "modules")

			if len(args) > 0 {
				modName := args[0]
				secDir := filepath.Join(moduleDir, modName, "securities")
				if err := loader.LoadFromDirectory(secDir, modName); err != nil {
					return fmt.Errorf("failed to load securities for %s: %w", modName, err)
				}
				fmt.Printf("Loaded securities for module: %s\n", modName)
				return nil
			}

			entries, err := os.ReadDir(moduleDir)
			if err != nil {
				return fmt.Errorf("cannot read module dir: %w", err)
			}
			loaded := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				secDir := filepath.Join(moduleDir, entry.Name(), "securities")
				if _, err := os.Stat(secDir); os.IsNotExist(err) {
					continue
				}
				if err := loader.LoadFromDirectory(secDir, entry.Name()); err != nil {
					log.Printf("[WARN] %s: %v", entry.Name(), err)
					continue
				}
				loaded++
				fmt.Printf("Loaded: %s\n", entry.Name())
			}
			fmt.Printf("Done. Loaded securities from %d modules.\n", loaded)
			_ = force
			return nil
		},
	}
}

func securityExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [module]",
		Short: "Export security definitions from database to JSON files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Export: not yet implemented. Use admin UI for now.")
			return nil
		},
	}
}

func securityDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [module]",
		Short: "Show differences between database and JSON files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleDir := envOrDefault("MODULE_DIR", "modules")
			modName := ""
			if len(args) > 0 {
				modName = args[0]
			}

			entries, err := os.ReadDir(moduleDir)
			if err != nil {
				return fmt.Errorf("cannot read module dir: %w", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if modName != "" && entry.Name() != modName {
					continue
				}
				secDir := filepath.Join(moduleDir, entry.Name(), "securities")
				secEntries, err := os.ReadDir(secDir)
				if err != nil {
					continue
				}
				for _, se := range secEntries {
					if se.IsDir() || !strings.HasSuffix(se.Name(), ".json") {
						continue
					}
					path := filepath.Join(secDir, se.Name())
					data, err := os.ReadFile(path)
					if err != nil {
						continue
					}
					secDef, err := parser.ParseSecurity(data)
					if err != nil {
						fmt.Printf("  [ERROR] %s: %v\n", se.Name(), err)
						continue
					}
					fmt.Printf("  %s: %s (%s)\n", entry.Name(), secDef.Name, se.Name())
				}
			}
			fmt.Println("Full diff comparison: not yet implemented.")
			return nil
		},
	}
}

func securityValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [module]",
		Short: "Validate security JSON files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleDir := envOrDefault("MODULE_DIR", "modules")
			modName := ""
			if len(args) > 0 {
				modName = args[0]
			}

			entries, err := os.ReadDir(moduleDir)
			if err != nil {
				return fmt.Errorf("cannot read module dir: %w", err)
			}

			errors := 0
			valid := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if modName != "" && entry.Name() != modName {
					continue
				}
				secDir := filepath.Join(moduleDir, entry.Name(), "securities")
				secEntries, err := os.ReadDir(secDir)
				if err != nil {
					continue
				}
				for _, se := range secEntries {
					if se.IsDir() || !strings.HasSuffix(se.Name(), ".json") {
						continue
					}
					path := filepath.Join(secDir, se.Name())
					data, err := os.ReadFile(path)
					if err != nil {
						fmt.Printf("  [ERROR] %s/%s: cannot read: %v\n", entry.Name(), se.Name(), err)
						errors++
						continue
					}
					_, err = parser.ParseSecurity(data)
					if err != nil {
						fmt.Printf("  [INVALID] %s/%s: %v\n", entry.Name(), se.Name(), err)
						errors++
						continue
					}
					valid++
					fmt.Printf("  [OK] %s/%s\n", entry.Name(), se.Name())
				}
			}
			fmt.Printf("\nValidation complete: %d valid, %d errors\n", valid, errors)
			if errors > 0 {
				return fmt.Errorf("%d validation errors found", errors)
			}
			return nil
		},
	}
}

func securityHistoryCmd() *cobra.Command {
	var entity string
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show security change history",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDBForCLI()
			if err != nil {
				return err
			}

			query := db.Table("ir_security_histories").Order("created_at DESC").Limit(50)
			if entity != "" {
				query = query.Where("entity_name = ?", entity)
			}

			var history []map[string]any
			if err := query.Find(&history).Error; err != nil {
				return fmt.Errorf("failed to query history: %w", err)
			}

			if len(history) == 0 {
				fmt.Println("No security changes recorded.")
				return nil
			}

			fmt.Printf("%-20s %-20s %-8s %-8s %s\n", "DATE", "ENTITY", "ACTION", "SOURCE", "CHANGES")
			fmt.Println(strings.Repeat("-", 80))
			for _, h := range history {
				date := fmt.Sprintf("%v", h["created_at"])
				if len(date) > 19 {
					date = date[:19]
				}
				entityName := fmt.Sprintf("%v", h["entity_name"])
				action := fmt.Sprintf("%v", h["action"])
				source := fmt.Sprintf("%v", h["source"])
				changes := fmt.Sprintf("%v", h["changes"])
				if changes == "<nil>" {
					changes = "(new)"
				}
				if len(changes) > 30 {
					changes = changes[:30] + "..."
				}
				fmt.Printf("%-20s %-20s %-8s %-8s %s\n", date, entityName, action, source, changes)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&entity, "entity", "", "Filter by entity name")
	return cmd
}

func openDBForCLI() (*gorm.DB, error) {
	configPath := ""
	if _, err := os.Stat("bitcode.yaml"); err == nil {
		configPath = "bitcode.yaml"
	}
	cfg, err := internal.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	db, err := persistence.NewDatabase(cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	return db, nil
}
