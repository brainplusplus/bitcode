package persistence

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseConfig struct {
	Driver     string // "sqlite" (default), "postgres", "mysql"
	Host       string
	Port       int
	User       string
	Password   string
	DBName     string
	SSLMode    string
	SQLitePath string
	Schema     string // Postgres only, default "public"
}

func NewDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	if cfg.Driver == "" {
		cfg.Driver = "sqlite"
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch cfg.Driver {
	case "sqlite":
		return openSQLite(cfg, gormCfg)
	case "postgres":
		return openPostgres(cfg, gormCfg)
	case "mysql":
		return openMySQL(cfg, gormCfg)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (use sqlite, postgres, or mysql)", cfg.Driver)
	}
}

func openSQLite(cfg DatabaseConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
	path := cfg.SQLitePath
	if path == "" {
		path = cfg.DBName
	}
	if path == "" {
		path = "bitcode.db"
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}

	db, err := gorm.Open(sqlite.Open(path), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database %s: %w", path, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.Exec("PRAGMA journal_mode=WAL")
	sqlDB.Exec("PRAGMA foreign_keys=ON")

	return db, nil
}

func openPostgres(cfg DatabaseConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	schema := cfg.Schema
	if schema == "" {
		schema = "public"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	if schema != "public" {
		dsn += fmt.Sprintf(" search_path=%s", schema)
	}

	db, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if schema != "public" {
		db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema))
		db.Exec(fmt.Sprintf("SET search_path TO %s", schema))
	}

	return db, nil
}

func openMySQL(cfg DatabaseConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
	if cfg.Port == 0 {
		cfg.Port = 3306
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)

	db, err := gorm.Open(mysql.Open(dsn), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	return db, nil
}
