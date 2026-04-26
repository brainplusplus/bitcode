package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type MigrationSourceType string

const (
	SourceJSON MigrationSourceType = "json"
	SourceCSV  MigrationSourceType = "csv"
	SourceXLSX MigrationSourceType = "xlsx"
	SourceXML  MigrationSourceType = "xml"
)

type MigrationConflictMode string

const (
	ConflictSkip   MigrationConflictMode = "skip"
	ConflictUpsert MigrationConflictMode = "upsert"
	ConflictError  MigrationConflictMode = "error"
)

type MigrationDownStrategy string

const (
	DownNone         MigrationDownStrategy = "none"
	DownDeleteSeeded MigrationDownStrategy = "delete_seeded"
	DownTruncate     MigrationDownStrategy = "truncate"
	DownCustom       MigrationDownStrategy = "custom"
)

type MigrationSource struct {
	Type    MigrationSourceType    `json:"type"`
	File    string                 `json:"file"`
	Options MigrationSourceOptions `json:"options,omitempty"`
}

type MigrationSourceOptions struct {
	Sheet       string            `json:"sheet,omitempty"`
	HeaderRow   int               `json:"header_row,omitempty"`
	Delimiter   string            `json:"delimiter,omitempty"`
	RootElement string            `json:"root_element,omitempty"`
	Encoding    string            `json:"encoding,omitempty"`
	SkipRows    int               `json:"skip_rows,omitempty"`
	FieldTypes  map[string]string `json:"field_types,omitempty"`
}

type MigrationProcessor struct {
	Type    string     `json:"type"`
	Script  *ScriptRef `json:"script,omitempty"`
	Process string     `json:"process,omitempty"`
}

type MigrationOptions struct {
	BatchSize     int                   `json:"batch_size,omitempty"`
	OnConflict    MigrationConflictMode `json:"on_conflict,omitempty"`
	UniqueFields  []string              `json:"unique_fields,omitempty"`
	UpdateFields  []string              `json:"update_fields,omitempty"`
	GenerateID    *bool                 `json:"generate_id,omitempty"`
	HashPasswords *bool                 `json:"hash_passwords,omitempty"`
	SetTimestamps *bool                 `json:"set_timestamps,omitempty"`
	NoUpdate      bool                  `json:"noupdate,omitempty"`
	DryRun        bool                  `json:"dry_run,omitempty"`
}

type MigrationDown struct {
	Strategy MigrationDownStrategy `json:"strategy,omitempty"`
	Process  string                `json:"process,omitempty"`
	Script   *ScriptRef            `json:"script,omitempty"`
}

type MigrationDefinition struct {
	Name         string              `json:"name"`
	Module       string              `json:"module,omitempty"`
	Model        string              `json:"model"`
	Description  string              `json:"description,omitempty"`
	Source       MigrationSource     `json:"source"`
	Processor    *MigrationProcessor `json:"processor,omitempty"`
	Options      MigrationOptions    `json:"options,omitempty"`
	FieldMapping map[string]string   `json:"field_mapping,omitempty"`
	Defaults     map[string]any      `json:"defaults,omitempty"`
	Down         *MigrationDown      `json:"down,omitempty"`
	Priority     int                 `json:"priority,omitempty"`
	DependsOn    []string            `json:"depends_on,omitempty"`
}

func (m *MigrationDefinition) ShouldGenerateID() bool {
	if m.Options.GenerateID == nil {
		return true
	}
	return *m.Options.GenerateID
}

func (m *MigrationDefinition) ShouldHashPasswords() bool {
	if m.Options.HashPasswords == nil {
		return true
	}
	return *m.Options.HashPasswords
}

func (m *MigrationDefinition) ShouldSetTimestamps() bool {
	if m.Options.SetTimestamps == nil {
		return true
	}
	return *m.Options.SetTimestamps
}

func (m *MigrationDefinition) GetBatchSize() int {
	if m.Options.BatchSize <= 0 {
		return 100
	}
	return m.Options.BatchSize
}

func (m *MigrationDefinition) GetConflictMode() MigrationConflictMode {
	if m.Options.OnConflict == "" {
		return ConflictSkip
	}
	return m.Options.OnConflict
}

func (m *MigrationDefinition) GetDownStrategy() MigrationDownStrategy {
	if m.Down == nil || m.Down.Strategy == "" {
		return DownNone
	}
	return m.Down.Strategy
}

func ParseMigration(data []byte) (*MigrationDefinition, error) {
	var mig MigrationDefinition
	if err := json.Unmarshal(data, &mig); err != nil {
		return nil, fmt.Errorf("invalid migration JSON: %w", err)
	}

	if mig.Name == "" {
		return nil, fmt.Errorf("migration name is required")
	}
	if mig.Model == "" {
		return nil, fmt.Errorf("migration model is required")
	}
	if mig.Source.Type == "" {
		return nil, fmt.Errorf("migration source type is required")
	}
	if mig.Source.File == "" {
		return nil, fmt.Errorf("migration source file is required")
	}

	switch mig.Source.Type {
	case SourceJSON, SourceCSV, SourceXLSX, SourceXML:
	default:
		return nil, fmt.Errorf("unsupported source type: %s (use json, csv, xlsx, xml)", mig.Source.Type)
	}

	if mig.Processor != nil {
		switch mig.Processor.Type {
		case "script":
			if mig.Processor.Script == nil {
				return nil, fmt.Errorf("processor type 'script' requires script configuration")
			}
		case "process":
			if mig.Processor.Process == "" {
				return nil, fmt.Errorf("processor type 'process' requires process name")
			}
		default:
			return nil, fmt.Errorf("unsupported processor type: %s (use script or process)", mig.Processor.Type)
		}
	}

	if mig.Options.OnConflict == ConflictUpsert && len(mig.Options.UniqueFields) == 0 {
		return nil, fmt.Errorf("upsert conflict mode requires unique_fields")
	}

	return &mig, nil
}

func ParseMigrationFile(path string) (*MigrationDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read migration file %s: %w", path, err)
	}
	return ParseMigration(data)
}

var migrationFilePattern = regexp.MustCompile(`^(\d{8}_\d{6})_(.+)\.json$`)

type MigrationFile struct {
	Path      string
	Timestamp string
	Name      string
	Def       *MigrationDefinition
}

func DiscoverMigrations(basePath string, patterns []string) ([]*MigrationFile, error) {
	var files []*MigrationFile

	for _, pattern := range patterns {
		fullPattern := filepath.Join(basePath, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			base := filepath.Base(match)

			parts := migrationFilePattern.FindStringSubmatch(base)
			var timestamp, name string
			if parts != nil {
				timestamp = parts[1]
				name = parts[2]
			} else {
				name = strings.TrimSuffix(base, filepath.Ext(base))
				timestamp = "00000000_000000"
			}

			def, err := ParseMigrationFile(match)
			if err != nil {
				return nil, fmt.Errorf("failed to parse migration %s: %w", match, err)
			}

			if def.Name == "" {
				def.Name = name
			}

			files = append(files, &MigrationFile{
				Path:      match,
				Timestamp: timestamp,
				Name:      def.Name,
				Def:       def,
			})
		}
	}

	sorted, err := topoSortMigrations(files)
	if err != nil {
		return nil, err
	}

	return sorted, nil
}

func topoSortMigrations(files []*MigrationFile) ([]*MigrationFile, error) {
	hasDeps := false
	for _, f := range files {
		if len(f.Def.DependsOn) > 0 {
			hasDeps = true
			break
		}
	}

	if !hasDeps {
		sort.Slice(files, func(i, j int) bool {
			if files[i].Timestamp != files[j].Timestamp {
				return files[i].Timestamp < files[j].Timestamp
			}
			if files[i].Def.Priority != files[j].Def.Priority {
				return files[i].Def.Priority < files[j].Def.Priority
			}
			return files[i].Name < files[j].Name
		})
		return files, nil
	}

	byName := make(map[string]*MigrationFile, len(files))
	for _, f := range files {
		byName[f.Name] = f
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var result []*MigrationFile

	var visit func(name string) error
	visit = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("circular dependency detected: %s", name)
		}
		if visited[name] {
			return nil
		}
		inStack[name] = true

		f, ok := byName[name]
		if !ok {
			inStack[name] = false
			return nil
		}

		for _, dep := range f.Def.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visited[name] = true
		inStack[name] = false
		result = append(result, f)
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Timestamp != files[j].Timestamp {
			return files[i].Timestamp < files[j].Timestamp
		}
		if files[i].Def.Priority != files[j].Def.Priority {
			return files[i].Def.Priority < files[j].Def.Priority
		}
		return files[i].Name < files[j].Name
	})

	for _, f := range files {
		if err := visit(f.Name); err != nil {
			return nil, err
		}
	}

	return result, nil
}
