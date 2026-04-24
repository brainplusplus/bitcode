package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bitcode-engine/engine/internal"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/infrastructure/watcher"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	root := &cobra.Command{
		Use:   "bitcode",
		Short: "BitCode Engine CLI",
	}

	root.AddCommand(initCmd())
	root.AddCommand(devCmd())
	root.AddCommand(validateCmd())
	root.AddCommand(versionCmd())
	root.AddCommand(moduleCmd())
	root.AddCommand(userCmd())
	root.AddCommand(dbCmd())
	root.AddCommand(publishCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [project-name]",
		Short: "Create a new bitcode project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dirs := []string{
				name + "/modules",
				name + "/scripts",
				name + "/templates",
			}
			for _, d := range dirs {
				if err := os.MkdirAll(d, 0755); err != nil {
					return fmt.Errorf("failed to create %s: %w", d, err)
				}
			}

			config := fmt.Sprintf("name: %s\nversion: 0.1.0\nport: 8080\ndatabase:\n  host: localhost\n  port: 5432\n  name: %s\n  user: postgres\n  password: postgres\n", name, name)
			if err := os.WriteFile(name+"/bitcode.yaml", []byte(config), 0644); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Printf("Project %s created.\n", name)
			fmt.Println("Next steps:")
			fmt.Println("  cd " + name)
			fmt.Println("  bitcode dev")
			return nil
		},
	}
}

func devCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dev",
		Short: "Start development server with hot reload",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				mu         sync.Mutex
				currentApp *internal.App
			)

			startApp := func() error {
				app, err := buildApp()
				if err != nil {
					return err
				}
				if err := app.LoadModules(); err != nil {
					return fmt.Errorf("failed to load modules: %w", err)
				}
				mu.Lock()
				currentApp = app
				mu.Unlock()

				go func() {
					if err := app.Start(); err != nil {
						errMsg := err.Error()
						if strings.Contains(errMsg, "server closed") || strings.Contains(errMsg, "use of closed network connection") {
							return
						}
						log.Printf("[DEV] server error: %v", err)
					}
				}()
				return nil
			}

			stopApp := func() {
				mu.Lock()
				app := currentApp
				currentApp = nil
				mu.Unlock()
				if app != nil {
					app.Shutdown()
				}
			}

			if err := startApp(); err != nil {
				return err
			}

			moduleDir := envOrDefault("MODULE_DIR", "modules")
			w := watcher.New(moduleDir, 2*time.Second, func() {
				log.Println("[DEV] changes detected, restarting server...")
				stopApp()
				time.Sleep(100 * time.Millisecond)
				if err := startApp(); err != nil {
					log.Printf("[DEV] restart failed: %v", err)
				} else {
					log.Println("[DEV] server restarted")
				}
			})
			go w.Start()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit

			w.Stop()
			fmt.Println("Shutting down...")
			stopApp()
			return nil
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate all JSON definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleDir := envOrDefault("MODULE_DIR", "modules")
			moduleDirs, _ := filepath.Glob(filepath.Join(moduleDir, "*/module.json"))

			errors := 0
			for _, modFile := range moduleDirs {
				modDir := filepath.Dir(modFile)
				modDef, err := parser.ParseModuleFile(modFile)
				if err != nil {
					fmt.Printf("  FAIL %s: %v\n", modFile, err)
					errors++
					continue
				}
				fmt.Printf("  OK   module: %s (%s)\n", modDef.Name, modDef.Version)

				for _, pattern := range modDef.Models {
					matches, _ := filepath.Glob(filepath.Join(modDir, pattern))
					for _, m := range matches {
						if _, err := parser.ParseModelFile(m); err != nil {
							fmt.Printf("  FAIL %s: %v\n", m, err)
							errors++
						} else {
							fmt.Printf("  OK   model: %s\n", filepath.Base(m))
						}
					}
				}

				for _, pattern := range modDef.APIs {
					matches, _ := filepath.Glob(filepath.Join(modDir, pattern))
					for _, a := range matches {
						if _, err := parser.ParseAPIFile(a); err != nil {
							fmt.Printf("  FAIL %s: %v\n", a, err)
							errors++
						} else {
							fmt.Printf("  OK   api: %s\n", filepath.Base(a))
						}
					}
				}
			}

			if errors > 0 {
				return fmt.Errorf("%d validation error(s)", errors)
			}
			fmt.Println("All definitions valid.")
			return nil
		},
	}
}

func moduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Module management",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleDir := envOrDefault("MODULE_DIR", "modules")
			moduleDirs, _ := filepath.Glob(filepath.Join(moduleDir, "*/module.json"))

			fmt.Printf("%-15s %-10s %-20s %s\n", "NAME", "VERSION", "LABEL", "DEPENDS")
			fmt.Println("--------------------------------------------------------------")
			for _, modFile := range moduleDirs {
				modDef, err := parser.ParseModuleFile(modFile)
				if err != nil {
					continue
				}
				deps := ""
				for i, d := range modDef.Depends {
					if i > 0 {
						deps += ", "
					}
					deps += d
				}
				fmt.Printf("%-15s %-10s %-20s %s\n", modDef.Name, modDef.Version, modDef.Label, deps)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create [name]",
		Short: "Scaffold a new module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			moduleDir := envOrDefault("MODULE_DIR", "modules")
			base := filepath.Join(moduleDir, name)

			dirs := []string{
				filepath.Join(base, "models"),
				filepath.Join(base, "apis"),
				filepath.Join(base, "processes"),
				filepath.Join(base, "views"),
				filepath.Join(base, "templates"),
				filepath.Join(base, "scripts"),
				filepath.Join(base, "data"),
				filepath.Join(base, "i18n"),
			}
			for _, d := range dirs {
				os.MkdirAll(d, 0755)
			}

			moduleJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "label": "%s",
  "depends": ["base"],
  "category": "",
  "models": ["models/*.json"],
  "apis": ["apis/*.json"],
  "processes": ["processes/*.json"],
  "views": ["views/*.json"],
  "permissions": {},
  "groups": {}
}`, name, name)

			if err := os.WriteFile(filepath.Join(base, "module.json"), []byte(moduleJSON), 0644); err != nil {
				return err
			}

			fmt.Printf("Module %s created at %s\n", name, base)
			return nil
		},
	})

	return cmd
}

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create [username] [email]",
		Short: "Create a new user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}

			username := args[0]
			email := args[1]

			fmt.Printf("Enter password for %s: ", username)
			var password string
			fmt.Scanln(&password)

			if password == "" {
				password = "changeme123"
				fmt.Println("Using default password: changeme123")
			}

			result := app.DB.Exec(
				"INSERT INTO users (id, username, email, password_hash, active, created_at, updated_at) VALUES (gen_random_uuid(), ?, ?, crypt(?, gen_salt('bf')), true, NOW(), NOW())",
				username, email, password,
			)
			if result.Error != nil {
				return fmt.Errorf("failed to create user: %w", result.Error)
			}

			fmt.Printf("User %s (%s) created.\n", username, email)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}

			var results []map[string]any
			app.DB.Table("users").Select("id, username, email, active").Find(&results)

			fmt.Printf("%-36s %-20s %-30s %s\n", "ID", "USERNAME", "EMAIL", "ACTIVE")
			fmt.Println("------------------------------------------------------------------------------------")
			for _, r := range results {
				fmt.Printf("%-36v %-20v %-30v %v\n", r["id"], r["username"], r["email"], r["active"])
			}
			return nil
		},
	})

	return cmd
}

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}
			fmt.Println("Migrations complete.")
			return nil
		},
	})

	cmd.AddCommand(backupCmd())
	cmd.AddCommand(restoreCmd())

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("bitcode %s\n", version)
		},
	}
}

func buildApp() (*internal.App, error) {
	configPath := ""
	if _, err := os.Stat("bitcode.yaml"); err == nil {
		configPath = "bitcode.yaml"
	}
	cfg, err := internal.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return internal.NewApp(cfg)
}

func envOrDefault(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
