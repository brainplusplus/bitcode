package main

import (
	"fmt"
	"os"

	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/spf13/cobra"
)

func backupCmd() *cobra.Command {
	var gzipFlag bool

	cmd := &cobra.Command{
		Use:   "backup [output-path]",
		Short: "Backup the database",
		Long:  "Create a backup of the database. SQLite: file copy. PostgreSQL: pg_dump. MySQL: mysqldump.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadDBConfig()
			db, err := persistence.NewDatabase(cfg)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			outputPath := ""
			if len(args) > 0 {
				outputPath = args[0]
			}

			opts := persistence.BackupOptions{
				OutputPath: outputPath,
				Gzip:       gzipFlag,
			}

			path, err := persistence.Backup(db, cfg, opts)
			if err != nil {
				return fmt.Errorf("backup failed: %w", err)
			}

			fmt.Printf("Backup created: %s\n", path)
			if _, err := os.Stat(path + ".meta.json"); err == nil {
				fmt.Printf("Metadata: %s\n", path+".meta.json")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&gzipFlag, "gzip", false, "Compress backup with gzip")
	return cmd
}

func restoreCmd() *cobra.Command {
	var forceFlag bool

	cmd := &cobra.Command{
		Use:   "restore [backup-path]",
		Short: "Restore the database from a backup",
		Long:  "Restore the database from a backup file. WARNING: This will overwrite the current database.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath := args[0]

			if !forceFlag {
				fmt.Printf("WARNING: This will overwrite the current database.\n")
				fmt.Printf("Restore from: %s\n", inputPath)
				fmt.Printf("Use --force to skip this confirmation.\n")
				fmt.Print("Continue? [y/N]: ")

				var answer string
				fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" {
					fmt.Println("Restore cancelled.")
					return nil
				}
			}

			cfg := loadDBConfig()
			db, err := persistence.NewDatabase(cfg)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			opts := persistence.RestoreOptions{
				InputPath: inputPath,
				Force:     forceFlag,
			}

			if err := persistence.Restore(db, cfg, opts); err != nil {
				return fmt.Errorf("restore failed: %w", err)
			}

			fmt.Println("Database restored successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&forceFlag, "force", false, "Skip confirmation prompt")
	return cmd
}

func loadDBConfig() persistence.DatabaseConfig {
	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		driver = "sqlite"
	}

	port := 5432
	if driver == "mysql" {
		port = 3306
	}

	sqlitePath := os.Getenv("DB_SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "bitcode.db"
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "bitcode"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "bitcode"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "bitcode"
	}

	return persistence.DatabaseConfig{
		Driver:     driver,
		Host:       host,
		Port:       port,
		User:       user,
		Password:   password,
		DBName:     dbName,
		SQLitePath: sqlitePath,
	}
}
