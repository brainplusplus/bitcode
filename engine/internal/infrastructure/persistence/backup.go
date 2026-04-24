package persistence

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

type BackupMeta struct {
	EngineVersion string    `json:"engine_version"`
	Driver        string    `json:"driver"`
	CreatedAt     time.Time `json:"created_at"`
	Compressed    bool      `json:"compressed"`
}

type BackupOptions struct {
	OutputPath string
	Gzip       bool
}

type RestoreOptions struct {
	InputPath string
	Force     bool
}

func Backup(db *gorm.DB, cfg DatabaseConfig, opts BackupOptions) (string, error) {
	dialect := DetectDialect(db)

	if opts.OutputPath == "" {
		ts := time.Now().Format("20060102-150405")
		switch dialect {
		case DialectSQLite:
			opts.OutputPath = fmt.Sprintf("backup-%s.db", ts)
		default:
			ext := ".sql"
			if opts.Gzip {
				ext = ".sql.gz"
			}
			opts.OutputPath = fmt.Sprintf("backup-%s%s", ts, ext)
		}
	}

	switch dialect {
	case DialectSQLite:
		return backupSQLite(cfg, opts)
	case DialectPostgres:
		return backupPostgres(cfg, opts)
	case DialectMySQL:
		return backupMySQL(cfg, opts)
	default:
		return "", fmt.Errorf("unsupported database driver for backup: %s", dialect)
	}
}

func Restore(db *gorm.DB, cfg DatabaseConfig, opts RestoreOptions) error {
	if _, err := os.Stat(opts.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", opts.InputPath)
	}

	dialect := DetectDialect(db)

	switch dialect {
	case DialectSQLite:
		return restoreSQLite(cfg, opts)
	case DialectPostgres:
		return restorePostgres(cfg, opts)
	case DialectMySQL:
		return restoreMySQL(cfg, opts)
	default:
		return fmt.Errorf("unsupported database driver for restore: %s", dialect)
	}
}

func backupSQLite(cfg DatabaseConfig, opts BackupOptions) (string, error) {
	srcPath := cfg.SQLitePath
	if srcPath == "" {
		srcPath = "bitcode.db"
	}

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", fmt.Errorf("SQLite database not found: %s", srcPath)
	}

	if err := os.MkdirAll(filepath.Dir(opts.OutputPath), 0755); err != nil && filepath.Dir(opts.OutputPath) != "." {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if opts.Gzip {
		gz := gzip.NewWriter(dst)
		defer gz.Close()
		if _, err := io.Copy(gz, src); err != nil {
			return "", fmt.Errorf("failed to compress backup: %w", err)
		}
	} else {
		if _, err := io.Copy(dst, src); err != nil {
			return "", fmt.Errorf("failed to copy database: %w", err)
		}
	}

	writeBackupMeta(opts.OutputPath, "sqlite", opts.Gzip)
	return opts.OutputPath, nil
}

func backupPostgres(cfg DatabaseConfig, opts BackupOptions) (string, error) {
	args := []string{
		"-h", cfg.Host,
		"-p", fmt.Sprintf("%d", cfg.Port),
		"-U", cfg.User,
		"-d", cfg.DBName,
		"-F", "p",
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))

	output, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer output.Close()

	if opts.Gzip {
		gz := gzip.NewWriter(output)
		defer gz.Close()
		cmd.Stdout = gz
	} else {
		cmd.Stdout = output
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(opts.OutputPath)
		return "", fmt.Errorf("pg_dump failed: %w", err)
	}

	writeBackupMeta(opts.OutputPath, "postgres", opts.Gzip)
	return opts.OutputPath, nil
}

func backupMySQL(cfg DatabaseConfig, opts BackupOptions) (string, error) {
	args := []string{
		"-h", cfg.Host,
		"-P", fmt.Sprintf("%d", cfg.Port),
		"-u", cfg.User,
		fmt.Sprintf("-p%s", cfg.Password),
		cfg.DBName,
	}

	cmd := exec.Command("mysqldump", args...)

	output, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer output.Close()

	if opts.Gzip {
		gz := gzip.NewWriter(output)
		defer gz.Close()
		cmd.Stdout = gz
	} else {
		cmd.Stdout = output
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(opts.OutputPath)
		return "", fmt.Errorf("mysqldump failed: %w", err)
	}

	writeBackupMeta(opts.OutputPath, "mysql", opts.Gzip)
	return opts.OutputPath, nil
}

func restoreSQLite(cfg DatabaseConfig, opts RestoreOptions) error {
	dstPath := cfg.SQLitePath
	if dstPath == "" {
		dstPath = "bitcode.db"
	}

	src, err := os.Open(opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create database file: %w", err)
	}
	defer dst.Close()

	var reader io.Reader = src
	if isGzipped(opts.InputPath) {
		gz, err := gzip.NewReader(src)
		if err != nil {
			return fmt.Errorf("failed to decompress backup: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	if _, err := io.Copy(dst, reader); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	return nil
}

func restorePostgres(cfg DatabaseConfig, opts RestoreOptions) error {
	args := []string{
		"-h", cfg.Host,
		"-p", fmt.Sprintf("%d", cfg.Port),
		"-U", cfg.User,
		"-d", cfg.DBName,
	}

	cmd := exec.Command("psql", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))

	input, err := os.Open(opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer input.Close()

	if isGzipped(opts.InputPath) {
		gz, err := gzip.NewReader(input)
		if err != nil {
			return fmt.Errorf("failed to decompress backup: %w", err)
		}
		defer gz.Close()
		cmd.Stdin = gz
	} else {
		cmd.Stdin = input
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore failed: %w", err)
	}
	return nil
}

func restoreMySQL(cfg DatabaseConfig, opts RestoreOptions) error {
	args := []string{
		"-h", cfg.Host,
		"-P", fmt.Sprintf("%d", cfg.Port),
		"-u", cfg.User,
		fmt.Sprintf("-p%s", cfg.Password),
		cfg.DBName,
	}

	cmd := exec.Command("mysql", args...)

	input, err := os.Open(opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer input.Close()

	if isGzipped(opts.InputPath) {
		gz, err := gzip.NewReader(input)
		if err != nil {
			return fmt.Errorf("failed to decompress backup: %w", err)
		}
		defer gz.Close()
		cmd.Stdin = gz
	} else {
		cmd.Stdin = input
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql restore failed: %w", err)
	}
	return nil
}

func isGzipped(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 2)
	if _, err := f.Read(buf); err != nil {
		return false
	}
	return buf[0] == 0x1f && buf[1] == 0x8b
}

func writeBackupMeta(backupPath, driver string, compressed bool) {
	meta := BackupMeta{
		EngineVersion: "0.1.0",
		Driver:        driver,
		CreatedAt:     time.Now(),
		Compressed:    compressed,
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := backupPath + ".meta.json"
	os.WriteFile(metaPath, data, 0644)
}
