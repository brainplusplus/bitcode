package persistence

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SequenceEngine struct {
	db *gorm.DB
}

func NewSequenceEngine(db *gorm.DB) *SequenceEngine {
	return &SequenceEngine{db: db}
}

func (e *SequenceEngine) MigrateSequenceTable() error {
	var sql string

	switch DetectDialect(e.db) {
	case DialectPostgres:
		sql = `CREATE TABLE IF NOT EXISTS sequences (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			model_name VARCHAR(100) NOT NULL,
			field_name VARCHAR(100) NOT NULL,
			sequence_key VARCHAR(500) NOT NULL,
			next_value BIGINT NOT NULL DEFAULT 1,
			step INT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(model_name, field_name, sequence_key)
		)`
	case DialectMySQL:
		sql = `CREATE TABLE IF NOT EXISTS sequences (
			id CHAR(36) PRIMARY KEY,
			model_name VARCHAR(100) NOT NULL,
			field_name VARCHAR(100) NOT NULL,
			sequence_key VARCHAR(500) NOT NULL,
			next_value BIGINT NOT NULL DEFAULT 1,
			step INT NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uq_seq (model_name, field_name, sequence_key)
		)`
	default:
		sql = `CREATE TABLE IF NOT EXISTS sequences (
			id TEXT PRIMARY KEY,
			model_name VARCHAR(100) NOT NULL,
			field_name VARCHAR(100) NOT NULL,
			sequence_key VARCHAR(500) NOT NULL,
			next_value INTEGER NOT NULL DEFAULT 1,
			step INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(model_name, field_name, sequence_key)
		)`
	}

	if err := e.db.Exec(sql).Error; err != nil {
		return fmt.Errorf("failed to migrate sequences table: %w", err)
	}

	return nil
}

func (e *SequenceEngine) NextValue(modelName, fieldName, sequenceKey string, step int) (int64, error) {
	if step <= 0 {
		step = 1
	}

	initialNextValue := int64(step + 1)
	currentTime := time.Now()

	switch DetectDialect(e.db) {
	case DialectPostgres:
		var value int64
		err := e.db.Raw(`
			INSERT INTO sequences (model_name, field_name, sequence_key, next_value, step, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (model_name, field_name, sequence_key)
			DO UPDATE SET
				next_value = sequences.next_value + EXCLUDED.step,
				step = EXCLUDED.step,
				updated_at = EXCLUDED.updated_at
			RETURNING next_value - step
		`, modelName, fieldName, sequenceKey, initialNextValue, step, currentTime, currentTime).Scan(&value).Error
		if err != nil {
			return 0, fmt.Errorf("failed to get next sequence value: %w", err)
		}
		return value, nil
	case DialectMySQL:
		var value int64
		err := e.db.Transaction(func(tx *gorm.DB) error {
			insertErr := tx.Exec(`
				INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE
					next_value = next_value + VALUES(step),
					step = VALUES(step),
					updated_at = VALUES(updated_at)
			`, uuid.New().String(), modelName, fieldName, sequenceKey, initialNextValue, step, currentTime, currentTime).Error
			if insertErr != nil {
				return insertErr
			}

			selectErr := tx.Raw(`
				SELECT next_value - step
				FROM sequences
				WHERE model_name = ? AND field_name = ? AND sequence_key = ?
			`, modelName, fieldName, sequenceKey).Scan(&value).Error
			if selectErr != nil {
				return selectErr
			}

			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("failed to get next sequence value: %w", err)
		}
		return value, nil
	default:
		var value int64
		err := e.db.Raw(`
			INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(model_name, field_name, sequence_key)
			DO UPDATE SET
				next_value = sequences.next_value + excluded.step,
				step = excluded.step,
				updated_at = excluded.updated_at
			RETURNING next_value - step
		`, uuid.New().String(), modelName, fieldName, sequenceKey, initialNextValue, step, currentTime, currentTime).Scan(&value).Error
		if err != nil {
			return 0, fmt.Errorf("failed to get next sequence value: %w", err)
		}
		return value, nil
	}
}

func BuildSequenceKey(modelName, fieldName, reset string) string {
	base := fmt.Sprintf("%s:%s", modelName, fieldName)
	now := time.Now()

	switch reset {
	case "", "never":
		return base
	case "yearly":
		return fmt.Sprintf("%s:%04d", base, now.Year())
	case "monthly":
		return fmt.Sprintf("%s:%04d-%02d", base, now.Year(), now.Month())
	case "daily":
		return fmt.Sprintf("%s:%04d-%02d-%02d", base, now.Year(), now.Month(), now.Day())
	case "hourly":
		return fmt.Sprintf("%s:%04d-%02d-%02dT%02d", base, now.Year(), now.Month(), now.Day(), now.Hour())
	case "minutely":
		return fmt.Sprintf("%s:%04d-%02d-%02dT%02d:%02d", base, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())
	default:
		return base
	}
}
