package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bitcode-engine/engine/internal"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/infrastructure/watcher"
	"github.com/bitcode-engine/engine/pkg/security"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	root := &cobra.Command{
		Use:   "bitcode",
		Short: "BitCode Engine CLI",
	}

	root.AddCommand(serveCmd())
	root.AddCommand(devCmd())
	root.AddCommand(initCmd())
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

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start production server",
		Long:  "Start the BitCode engine server. Loads config, initializes app, loads modules, and serves HTTP.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return fmt.Errorf("failed to initialize app: %w", err)
			}

			if err := app.LoadModules(); err != nil {
				return fmt.Errorf("failed to load modules: %w", err)
			}

			serverErr := make(chan error, 1)
			go func() {
				if err := app.Start(); err != nil {
					serverErr <- err
				}
			}()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			select {
			case <-quit:
				fmt.Println("Shutting down...")
			case err := <-serverErr:
				if err == nil {
					return nil
				}
				msg := err.Error()
				if strings.Contains(msg, "server closed") || strings.Contains(msg, "use of closed network connection") {
					return nil
				}
				return fmt.Errorf("server error: %w", err)
			}

			if err := app.Shutdown(); err != nil {
				log.Printf("shutdown error: %v", err)
			}
			return nil
		},
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
	var forceEngine bool
	var forceApp bool

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start development server with hot reload",
		Long: `Start BitCode in development mode with automatic reload.

Auto-detects context:
  - Engine repo: uses Air for Go hot reload (rebuilds on .go changes)
  - App project: watches module files (JSON, HTML, templates) and reloads in-process

Override with --engine or --no-engine flags.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			engineMode := forceEngine
			if !forceEngine && !forceApp {
				engineMode = detectEngineRepo()
			}

			if engineMode {
				return runEngineDevMode()
			}
			return runAppDevMode()
		},
	}

	cmd.Flags().BoolVar(&forceEngine, "engine", false, "Force engine development mode (Air hot reload for Go code)")
	cmd.Flags().BoolVar(&forceApp, "no-engine", false, "Force app development mode (module file watcher only)")

	return cmd
}

func detectEngineRepo() bool {
	if _, err := os.Stat("go.mod"); err == nil {
		data, err := os.ReadFile("go.mod")
		if err == nil && strings.Contains(string(data), "github.com/bitcode-engine/engine") {
			return true
		}
	}

	if _, err := os.Stat("../../engine/go.mod"); err == nil {
		data, err := os.ReadFile("../../engine/go.mod")
		if err == nil && strings.Contains(string(data), "github.com/bitcode-engine/engine") {
			return true
		}
	}

	return false
}

func runEngineDevMode() error {
	if _, err := exec.LookPath("air"); err != nil {
		fmt.Println("[DEV] Air not found. Install it for Go hot reload:")
		fmt.Println("      go install github.com/air-verse/air@latest")
		fmt.Println()
		fmt.Println("[DEV] Falling back to app dev mode (module watcher only)...")
		fmt.Println()
		return runAppDevMode()
	}

	engineDir := "."
	if _, err := os.Stat("../../engine/go.mod"); err == nil {
		engineDir = "../../engine"
	}

	moduleDir := envOrDefault("MODULE_DIR", "modules")

	binName := "./tmp/bitcode"
	if runtime.GOOS == "windows" {
		binName = "./tmp/bitcode.exe"
	}

	buildCmd := fmt.Sprintf("go build -o %s ./cmd/bitcode/", binName)

	airArgs := []string{
		"--build.cmd", buildCmd,
		"--build.bin", binName + " serve",
		"--build.include_ext", "go,json,html,yaml,toml",
		"--build.exclude_dir", "tmp,vendor,node_modules,uploads,packages,.git",
		"--build.exclude_regex", "_test\\.go$",
		"--build.delay", "1000",
		"--build.stop_on_error", "true",
		"--build.send_interrupt", "true",
		"--build.kill_delay", "3000",
		"--misc.clean_on_exit", "true",
	}

	if engineDir != "." {
		airArgs = append(airArgs, "--build.include_dir", engineDir+","+moduleDir)
	}

	fmt.Println("[DEV] Engine development mode (Air hot reload)")
	fmt.Println("      Watching: *.go, *.json, *.html, *.yaml, *.toml")
	fmt.Println()

	airCmd := exec.Command("air", airArgs...)
	airCmd.Stdout = os.Stdout
	airCmd.Stderr = os.Stderr
	airCmd.Stdin = os.Stdin
	airCmd.Env = os.Environ()

	return airCmd.Run()
}

func runAppDevMode() error {
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

	fmt.Println("[DEV] App development mode (module watcher)")
	fmt.Println("      Watching:", moduleDir)
	fmt.Println()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	w.Stop()
	fmt.Println("Shutting down...")
	stopApp()
	return nil
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
			if err := app.LoadModules(); err != nil {
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

			hash, err := security.HashPassword(password)
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}

			tableName := app.ModelRegistry.TableName("user")
			record := map[string]any{
				"id":            uuid.New().String(),
				"username":      username,
				"email":         email,
				"password_hash": hash,
				"active":        true,
			}
			if err := app.DB.Table(tableName).Create(&record).Error; err != nil {
				return fmt.Errorf("failed to create user: %w", err)
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
			if err := app.LoadModules(); err != nil {
				return err
			}

			tableName := app.ModelRegistry.TableName("user")
			var results []map[string]any
			app.DB.Table(tableName).Select("id, username, email, active").Find(&results)

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
