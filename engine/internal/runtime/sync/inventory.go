package sync

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type InventoryDelta struct {
	TableName string  `json:"table_name"`
	RecordID  string  `json:"record_id"`
	Field     string  `json:"field"`
	Delta     float64 `json:"qty_delta"`
	DeviceID  string  `json:"device_id"`
}

type OversellAlert struct {
	ID            int64   `json:"id"`
	TableName     string  `json:"table_name"`
	RecordID      string  `json:"record_id"`
	Field         string  `json:"field"`
	PreviousQty   float64 `json:"previous_qty"`
	DeltaApplied  float64 `json:"delta_applied"`
	ResultingQty  float64 `json:"resulting_qty"`
	DeviceID      string  `json:"device_id"`
	EnvelopeID    string  `json:"envelope_id"`
	Acknowledged  bool    `json:"acknowledged"`
	AcknowledgedBy string `json:"acknowledged_by,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

func CreateOversellAlertsTable(db *gorm.DB) error {
	sql := `CREATE TABLE IF NOT EXISTS _sync_oversell_alerts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  field TEXT NOT NULL,
  previous_qty REAL NOT NULL,
  delta_applied REAL NOT NULL,
  resulting_qty REAL NOT NULL,
  device_id TEXT NOT NULL,
  envelope_id TEXT NOT NULL DEFAULT '',
  acknowledged INTEGER NOT NULL DEFAULT 0,
  acknowledged_by TEXT,
  acknowledged_at TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

	dialect := detectDialectFromDB(db)
	if dialect == "postgres" {
		sql = `CREATE TABLE IF NOT EXISTS _sync_oversell_alerts (
  id SERIAL PRIMARY KEY,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  field TEXT NOT NULL,
  previous_qty DOUBLE PRECISION NOT NULL,
  delta_applied DOUBLE PRECISION NOT NULL,
  resulting_qty DOUBLE PRECISION NOT NULL,
  device_id TEXT NOT NULL,
  envelope_id TEXT NOT NULL DEFAULT '',
  acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
  acknowledged_by TEXT,
  acknowledged_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`
	} else if dialect == "mysql" {
		sql = `CREATE TABLE IF NOT EXISTS _sync_oversell_alerts (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  table_name TEXT NOT NULL,
  record_id VARCHAR(255) NOT NULL,
  field VARCHAR(255) NOT NULL,
  previous_qty DOUBLE NOT NULL,
  delta_applied DOUBLE NOT NULL,
  resulting_qty DOUBLE NOT NULL,
  device_id VARCHAR(255) NOT NULL,
  envelope_id VARCHAR(255) NOT NULL DEFAULT '',
  acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
  acknowledged_by VARCHAR(255),
  acknowledged_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}

	return db.Exec(sql).Error
}

// ApplyInventoryDelta applies a quantity delta to a record field and returns an OversellAlert if stock goes negative.
func ApplyInventoryDelta(tx *gorm.DB, delta InventoryDelta, envelopeID string) (*OversellAlert, error) {
	if !isValidTableName(delta.TableName) {
		return nil, fmt.Errorf("invalid table name: %s", delta.TableName)
	}
	if !isValidTableName(delta.Field) {
		return nil, fmt.Errorf("invalid field name: %s", delta.Field)
	}

	var currentQty float64
	err := tx.Raw(
		fmt.Sprintf("SELECT COALESCE(%s, 0) FROM %s WHERE id = ?", delta.Field, delta.TableName),
		delta.RecordID,
	).Scan(&currentQty).Error
	if err != nil {
		return nil, fmt.Errorf("failed to read current %s: %w", delta.Field, err)
	}

	newQty := currentQty + delta.Delta

	err = tx.Exec(
		fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", delta.TableName, delta.Field),
		newQty, delta.RecordID,
	).Error
	if err != nil {
		return nil, fmt.Errorf("failed to apply delta to %s: %w", delta.Field, err)
	}

	if newQty < 0 {
		alert := &OversellAlert{
			TableName:    delta.TableName,
			RecordID:     delta.RecordID,
			Field:        delta.Field,
			PreviousQty:  currentQty,
			DeltaApplied: delta.Delta,
			ResultingQty: newQty,
			DeviceID:     delta.DeviceID,
			EnvelopeID:   envelopeID,
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		}

		err = tx.Exec(
			`INSERT INTO _sync_oversell_alerts (table_name, record_id, field, previous_qty, delta_applied, resulting_qty, device_id, envelope_id, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			alert.TableName, alert.RecordID, alert.Field,
			alert.PreviousQty, alert.DeltaApplied, alert.ResultingQty,
			alert.DeviceID, alert.EnvelopeID, alert.CreatedAt,
		).Error
		if err != nil {
			return alert, fmt.Errorf("delta applied but failed to record oversell alert: %w", err)
		}

		return alert, nil
	}

	return nil, nil
}

// DetectInventoryFields checks if a payload contains fields that look like inventory deltas.
// Convention: fields ending with "_delta" (e.g. "qty_delta", "stock_delta") are treated as deltas.
func DetectInventoryFields(payload map[string]interface{}) map[string]float64 {
	deltas := make(map[string]float64)
	for k, v := range payload {
		if len(k) > 6 && k[len(k)-6:] == "_delta" {
			targetField := k[:len(k)-6]
			switch val := v.(type) {
			case float64:
				deltas[targetField] = val
			case int:
				deltas[targetField] = float64(val)
			case int64:
				deltas[targetField] = float64(val)
			}
		}
	}
	return deltas
}

func detectDialectFromDB(db *gorm.DB) string {
	name := db.Dialector.Name()
	switch name {
	case "postgres":
		return "postgres"
	case "mysql":
		return "mysql"
	default:
		return "sqlite"
	}
}
