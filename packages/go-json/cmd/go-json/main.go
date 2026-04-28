package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	goio "github.com/bitcode-framework/go-json/io"
	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/stdlib"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		cmdRun(os.Args[2:])
	case "check":
		cmdCheck(os.Args[2:])
	case "test":
		cmdTest(os.Args[2:])
	case "ast":
		cmdAST(os.Args[2:])
	case "codegen":
		cmdCodegen(os.Args[2:])
	case "migrate":
		cmdMigrate(os.Args[2:])
	case "--version", "-v", "version":
		fmt.Printf("go-json %s\n", version)
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`go-json — JSON/JSONC programming language engine

Usage: go-json <command> [options]

Commands:
  run       Execute a go-json program
  check     Validate a program (compile check, no execution)
  test      Run test files
  ast       Export program AST as JSON
  codegen   Generate Go/JS/Python code from program
  migrate   Migrate deprecated syntax

Flags:
  --version   Print version
  --help      Print this help`)
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	inputJSON := fs.String("input", "", "Inline JSON input")
	inputFile := fs.String("input-file", "", "Read input from file")
	timeout := fs.String("timeout", "30s", "Execution timeout")
	maxDepth := fs.Int("max-depth", 0, "Override default call depth limit")
	ioModules := fs.String("io", "", "Enable I/O modules (http,fs,sql,exec or 'all')")
	trace := fs.Bool("trace", false, "Print execution trace")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json run <program.json> [options]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)

	if *inputJSON != "" && *inputFile != "" {
		fmt.Fprintln(os.Stderr, "Error: cannot use both --input and --input-file")
		os.Exit(1)
	}

	var input map[string]any
	if *inputJSON != "" {
		if err := json.Unmarshal([]byte(*inputJSON), &input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid JSON input: %s\n", err.Error())
			os.Exit(1)
		}
	} else if *inputFile != "" {
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot read input file: %s\n", err.Error())
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid JSON in input file: %s\n", err.Error())
			os.Exit(1)
		}
	}

	dur, err := time.ParseDuration(*timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid timeout: %s\n", err.Error())
		os.Exit(1)
	}

	reg := stdlib.DefaultRegistry()
	opts := []runtime.Option{
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
		runtime.WithLimits(runtime.Limits{Timeout: dur}),
		runtime.WithRuntimeTrace(*trace),
	}

	if *maxDepth > 0 {
		opts = append(opts, runtime.WithLimits(runtime.Limits{
			Timeout:  dur,
			MaxDepth: *maxDepth,
		}))
	}

	if *ioModules != "" {
		sec := goio.DefaultSecurityConfig()
		if *ioModules == "all" {
			opts = append(opts, runtime.WithIO(goio.All(sec)...))
		} else {
			for _, mod := range strings.Split(*ioModules, ",") {
				mod = strings.TrimSpace(mod)
				switch mod {
				case "http":
					opts = append(opts, runtime.WithIO(goio.HTTP(sec)))
				case "fs":
					opts = append(opts, runtime.WithIO(goio.FS(sec)))
				case "sql":
					opts = append(opts, runtime.WithIO(goio.SQL(sec)))
				case "exec":
					opts = append(opts, runtime.WithIO(goio.Exec(sec)))
				default:
					fmt.Fprintf(os.Stderr, "Error: unknown I/O module: %s\n", mod)
					os.Exit(1)
				}
			}
		}
	}

	rt := runtime.NewRuntime(opts...)

	compiled, err := rt.CompileFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	result, err := rt.Execute(compiled, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	if result.Value != nil {
		out, _ := json.MarshalIndent(result.Value, "", "  ")
		fmt.Println(string(out))
	}

	if *trace && result.Trace != nil {
		fmt.Fprintln(os.Stderr, "\n--- Trace ---")
		traceOut, _ := json.MarshalIndent(result.Trace, "", "  ")
		fmt.Fprintln(os.Stderr, string(traceOut))
	}
}

func cmdCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show program metadata")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json check <program.json>")
		os.Exit(1)
	}

	programPath := fs.Arg(0)

	reg := stdlib.DefaultRegistry()
	rt := runtime.NewRuntime(
		runtime.WithStdlib(reg.All()),
		runtime.WithStdlibEnv(reg.EnvVars()),
	)

	compiled, err := rt.CompileFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("OK")

	if *verbose {
		fmt.Printf("Name: %s\n", compiled.Name)
		if len(compiled.Functions) > 0 {
			names := make([]string, 0, len(compiled.Functions))
			for n := range compiled.Functions {
				names = append(names, n)
			}
			fmt.Printf("Functions: %s\n", strings.Join(names, ", "))
		}
		if len(compiled.Structs) > 0 {
			names := make([]string, 0, len(compiled.Structs))
			for n := range compiled.Structs {
				names = append(names, n)
			}
			fmt.Printf("Structs: %s\n", strings.Join(names, ", "))
		}
		if compiled.AST != nil && len(compiled.AST.Imports) > 0 {
			for _, imp := range compiled.AST.Imports {
				fmt.Printf("Import: %s → %s\n", imp.Alias, imp.Path)
			}
		}
	}
}

func cmdAST(args []string) {
	fs := flag.NewFlagSet("ast", flag.ExitOnError)
	output := fs.String("output", "", "Write to file (default: stdout)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json ast <program.json> [--output ast.json]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)
	data, err := os.ReadFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	program, err := lang.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	astJSON, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	if *output != "" {
		if err := os.WriteFile(*output, astJSON, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("AST written to %s\n", *output)
	} else {
		fmt.Println(string(astJSON))
	}
}

func cmdCodegen(args []string) {
	fs := flag.NewFlagSet("codegen", flag.ExitOnError)
	target := fs.String("target", "", "Target language: go, js, python (required)")
	output := fs.String("output", "", "Write to file (default: stdout)")
	pkg := fs.String("package", "main", "Go package name (only for --target go)")
	fs.Parse(args)

	if fs.NArg() < 1 || *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: go-json codegen <program.json> --target go|js|python [--output file]")
		os.Exit(1)
	}

	_ = pkg

	fmt.Fprintf(os.Stderr, "Code generation for target '%s' not yet implemented\n", *target)
	if *output != "" {
		_ = output
	}
	os.Exit(1)
}

func cmdMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	from := fs.String("from", "", "Source version (auto-detect if omitted)")
	to := fs.String("to", "v2", "Target version")
	output := fs.String("output", "", "Write to file (default: stdout)")
	dryRun := fs.Bool("dry-run", false, "Show changes without applying")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go-json migrate <program.json> [--from v1] [--to v2] [--dry-run]")
		os.Exit(1)
	}

	programPath := fs.Arg(0)
	data, err := os.ReadFile(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	migrated, changes := migrateProgram(string(data), *from, *to)

	if len(changes) == 0 {
		fmt.Println("Program is already current — no changes needed")
		return
	}

	if *dryRun {
		fmt.Println("Changes that would be applied:")
		for _, c := range changes {
			fmt.Printf("  - %s\n", c)
		}
		return
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(migrated), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("Migrated program written to %s\n", *output)
	} else {
		fmt.Println(migrated)
	}

	fmt.Fprintf(os.Stderr, "Applied %d changes\n", len(changes))
}

func migrateProgram(source, from, to string) (string, []string) {
	var changes []string
	result := source

	renames := map[string]string{
		"unique":     "uniq",
		"startsWith": "hasPrefix",
		"endsWith":   "hasSuffix",
	}

	for old, new := range renames {
		if strings.Contains(result, old) {
			result = strings.ReplaceAll(result, `"`+old+`"`, `"`+new+`"`)
			changes = append(changes, fmt.Sprintf("renamed '%s' → '%s'", old, new))
		}
	}

	return result, changes
}
