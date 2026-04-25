package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitcode-engine/engine/embedded"
	"github.com/bitcode-engine/engine/internal/infrastructure/module"
	"github.com/spf13/cobra"
)

func publishCmd() *cobra.Command {
	var force bool
	var dryRun bool
	var models, apis, views, templates, processes, data bool

	cmd := &cobra.Command{
		Use:   "publish [module|air.toml] [file]",
		Short: "Publish embedded module files or config templates to project",
		Long: `Extract embedded module files or generate config templates.
Similar to Laravel's artisan vendor:publish.

Examples:
  bitcode publish base                    # Publish entire base module
  bitcode publish base --models           # Publish only models
  bitcode publish base models/user.json   # Publish single file
  bitcode publish air.toml                # Generate .air.toml config
  bitcode publish --list                  # List publishable modules`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			listFlag, _ := cmd.Flags().GetBool("list")
			if listFlag {
				return listPublishable()
			}

			if len(args) == 0 {
				return fmt.Errorf("module name or 'air.toml' required. Use --list to see available modules")
			}

			if args[0] == "air.toml" {
				return publishAirToml(force, dryRun)
			}

			moduleName := args[0]
			moduleDir := envOrDefault("MODULE_DIR", "modules")
			targetDir := filepath.Join(moduleDir, moduleName)

			src := module.NewEmbedFSFromEmbed(embedded.ModulesFS, "modules/"+moduleName)

			if !src.Exists("module.json") {
				return fmt.Errorf("module %q not found in embedded modules", moduleName)
			}

			var specificFile string
			if len(args) == 2 {
				specificFile = args[1]
			}

			resourceFilter := buildResourceFilter(models, apis, views, templates, processes, data)

			published := 0
			skipped := 0

			err := publishFiles(src, targetDir, specificFile, resourceFilter, force, dryRun, &published, &skipped)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Printf("[DRY RUN] Would publish %d files to %s (%d skipped)\n", published, targetDir, skipped)
			} else {
				fmt.Printf("Published %d files to %s (%d skipped)\n", published, targetDir, skipped)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without writing")
	cmd.Flags().Bool("list", false, "List publishable modules")
	cmd.Flags().BoolVar(&models, "models", false, "Publish only models")
	cmd.Flags().BoolVar(&apis, "apis", false, "Publish only APIs")
	cmd.Flags().BoolVar(&views, "views", false, "Publish only views")
	cmd.Flags().BoolVar(&templates, "templates", false, "Publish only templates")
	cmd.Flags().BoolVar(&processes, "processes", false, "Publish only processes")
	cmd.Flags().BoolVar(&data, "data", false, "Publish only seed data")

	return cmd
}

func buildResourceFilter(models, apis, views, templates, processes, data bool) []string {
	if !models && !apis && !views && !templates && !processes && !data {
		return nil
	}
	var filters []string
	if models {
		filters = append(filters, "models/")
	}
	if apis {
		filters = append(filters, "apis/")
	}
	if views {
		filters = append(filters, "views/")
	}
	if templates {
		filters = append(filters, "templates/")
	}
	if processes {
		filters = append(filters, "processes/")
	}
	if data {
		filters = append(filters, "data/")
	}
	return filters
}

func publishFiles(src module.ModuleFS, targetDir, specificFile string, resourceFilter []string, force, dryRun bool, published, skipped *int) error {
	if specificFile != "" {
		return publishSingleFile(src, targetDir, specificFile, force, dryRun, published, skipped)
	}
	return publishRecursive(src, ".", targetDir, resourceFilter, force, dryRun, published, skipped)
}

func publishSingleFile(src module.ModuleFS, targetDir, file string, force, dryRun bool, published, skipped *int) error {
	fileData, err := src.ReadFile(file)
	if err != nil {
		return fmt.Errorf("file %q not found in embedded module", file)
	}

	targetPath := filepath.Join(targetDir, filepath.FromSlash(file))
	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("  SKIP %s (exists, use --force to overwrite)\n", file)
			*skipped++
			return nil
		}
	}

	if dryRun {
		fmt.Printf("  PUBLISH %s\n", file)
		*published++
		return nil
	}

	os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err := os.WriteFile(targetPath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", targetPath, err)
	}
	fmt.Printf("  PUBLISH %s\n", file)
	*published++
	return nil
}

func publishRecursive(src module.ModuleFS, dir, targetDir string, resourceFilter []string, force, dryRun bool, published, skipped *int) error {
	entries, err := src.ListDir(dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		path := entry
		if dir != "." {
			path = dir + "/" + entry
		}

		if len(resourceFilter) > 0 {
			matched := false
			for _, f := range resourceFilter {
				if strings.HasPrefix(path, f) || strings.HasPrefix(path+"/", f) {
					matched = true
					break
				}
			}
			if !matched && path != "module.json" {
				continue
			}
		}

		fileData, err := src.ReadFile(path)
		if err != nil {
			publishRecursive(src, path, targetDir, resourceFilter, force, dryRun, published, skipped)
			continue
		}

		targetPath := filepath.Join(targetDir, filepath.FromSlash(path))
		if !force {
			if _, err := os.Stat(targetPath); err == nil {
				fmt.Printf("  SKIP %s (exists)\n", path)
				*skipped++
				continue
			}
		}

		if dryRun {
			fmt.Printf("  PUBLISH %s\n", path)
			*published++
			continue
		}

		os.MkdirAll(filepath.Dir(targetPath), 0755)
		if err := os.WriteFile(targetPath, fileData, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetPath, err)
		}
		fmt.Printf("  PUBLISH %s\n", path)
		*published++
	}
	return nil
}

func listPublishable() error {
	embedFS := module.NewEmbedFSFromEmbed(embedded.ModulesFS, "modules")
	entries, err := embedFS.ListDir(".")
	if err != nil {
		fmt.Println("No embedded modules found.")
		return nil
	}

	fmt.Println("Publishable modules:")
	for _, entry := range entries {
		if embedFS.Exists(entry + "/module.json") {
			data, _ := embedFS.ReadFile(entry + "/module.json")
			fmt.Printf("  %s (%d bytes)\n", entry, len(data))
		}
	}
	fmt.Println("\nConfig templates:")
	fmt.Println("  air.toml (Air hot-reload config)")
	return nil
}

func publishAirToml(force, dryRun bool) error {
	targetPath := ".air.toml"

	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf(".air.toml already exists. Use --force to overwrite")
		}
	}

	isEngineRepo, enginePath := detectEngineRepoWithPath()

	var template string
	if isEngineRepo {
		template = airTomlEngineTemplate(enginePath)
	} else {
		template = airTomlAppTemplate()
	}

	if dryRun {
		fmt.Println("[DRY RUN] Would create .air.toml:")
		fmt.Println(template)
		return nil
	}

	if err := os.WriteFile(targetPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write .air.toml: %w", err)
	}

	mode := "app project"
	if isEngineRepo {
		mode = "engine repository"
	}
	fmt.Printf("Published .air.toml (%s mode)\n", mode)
	return nil
}

func detectEngineRepoWithPath() (bool, string) {
	if _, err := os.Stat("go.mod"); err == nil {
		data, err := os.ReadFile("go.mod")
		if err == nil && strings.Contains(string(data), "github.com/bitcode-engine/engine") {
			return true, "."
		}
	}

	if _, err := os.Stat("../../engine/go.mod"); err == nil {
		data, err := os.ReadFile("../../engine/go.mod")
		if err == nil && strings.Contains(string(data), "github.com/bitcode-engine/engine") {
			return true, "../../engine"
		}
	}

	return false, ""
}

func airTomlEngineTemplate(enginePath string) string {
	cmdPath := strings.ReplaceAll(filepath.Join(enginePath, "cmd/bitcode/"), "\\", "/")
	if enginePath == "." {
		cmdPath = "./cmd/bitcode/"
	}
	
	includeDirs := `["cmd", "internal", "pkg", "modules"]`
	if enginePath != "." {
		enginePathUnix := strings.ReplaceAll(enginePath, "\\", "/")
		includeDirs = fmt.Sprintf(`["%s", "modules"]`, enginePathUnix)
	}

	return fmt.Sprintf(`root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/bitcode %s"
  bin = "./tmp/bitcode serve"
  include_ext = ["go", "json", "html", "yaml", "toml"]
  include_dir = %s
  exclude_dir = ["tmp", "vendor", "node_modules", "uploads", "packages", ".git"]
  exclude_regex = ["_test\\.go$"]
  delay = 1000
  stop_on_error = true
  send_interrupt = true
  kill_delay = 3000

[log]
  time = false

[misc]
  clean_on_exit = true
`, cmdPath, includeDirs)
}

func airTomlAppTemplate() string {
	moduleDir := envOrDefault("MODULE_DIR", "modules")
	return fmt.Sprintf(`root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/bitcode github.com/bitcode-engine/engine/cmd/bitcode"
  bin = "./tmp/bitcode serve"
  include_ext = ["json", "html", "yaml", "toml"]
  include_dir = ["%s"]
  exclude_dir = ["tmp", "vendor", "node_modules", "uploads"]
  delay = 1000
  stop_on_error = true
  send_interrupt = true
  kill_delay = 3000

[log]
  time = false

[misc]
  clean_on_exit = true
`, moduleDir)
}
