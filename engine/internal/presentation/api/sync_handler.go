package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

type SyncHandler struct {
	db       *gorm.DB
	modelReg *domainModel.Registry
}

func NewSyncHandler(db *gorm.DB, modelReg *domainModel.Registry) *SyncHandler {
	return &SyncHandler{db: db, modelReg: modelReg}
}

type syncVersionRow struct {
	ID            int64  `gorm:"column:id"`
	TableName     string `gorm:"column:table_name"`
	RecordID      string `gorm:"column:record_id"`
	Operation     string `gorm:"column:operation"`
	Version       int64  `gorm:"column:version"`
	ChangedFields string `gorm:"column:changed_fields"`
	ChangedBy     string `gorm:"column:changed_by"`
}

type registerDeviceRequest struct {
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	StoreID    string `json:"store_id,omitempty"`
}

func (h *SyncHandler) RegisterDevice(c *fiber.Ctx) error {
	var req registerDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if req.Platform == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_platform",
			"message": "platform is required",
		})
	}

	deviceID := generateDeviceID()
	devicePrefix := generateDevicePrefix()
	now := time.Now().UTC()

	err := h.db.Exec(
		`INSERT INTO _sync_devices (device_id, device_prefix, platform, app_version, store_id, registered_at, last_sync_version, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		deviceID, devicePrefix, req.Platform, req.AppVersion, nullIfEmpty(req.StoreID), now, true,
	).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "registration_failed",
			"message": fmt.Sprintf("Failed to register device: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"device_id":     deviceID,
		"device_prefix": devicePrefix,
		"registered_at": now.Format(time.RFC3339),
	})
}

type pushEnvelopeRequest struct {
	EnvelopeID string          `json:"envelope_id"`
	DeviceID   string          `json:"device_id"`
	Operations []pushOperation `json:"operations"`
}

type pushOperation struct {
	TableName      string                 `json:"table_name"`
	RecordID       string                 `json:"record_id"`
	Operation      string                 `json:"operation"`
	Payload        map[string]interface{} `json:"payload"`
	IdempotencyKey string                 `json:"idempotency_key"`
}

func (h *SyncHandler) PushEnvelope(c *fiber.Ctx) error {
	var req pushEnvelopeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if req.EnvelopeID == "" || req.DeviceID == "" || len(req.Operations) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_fields",
			"message": "envelope_id, device_id, and operations are required",
		})
	}

	var deviceActive bool
	h.db.Raw("SELECT is_active FROM _sync_devices WHERE device_id = ?", req.DeviceID).Scan(&deviceActive)
	if !deviceActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "device_deactivated",
			"message": "Device has been deactivated",
		})
	}

	var existingStatus string
	err := h.db.Raw("SELECT status FROM _sync_log WHERE envelope_id = ?", req.EnvelopeID).Scan(&existingStatus).Error
	if err == nil && existingStatus != "" {
		return c.JSON(fiber.Map{
			"envelope_id": req.EnvelopeID,
			"status":      existingStatus,
			"message":     "already processed",
		})
	}

	startTime := time.Now()

	tx := h.db.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "transaction_failed",
			"message": "Failed to begin transaction",
		})
	}

	var maxVersion int64
	for _, op := range req.Operations {
		if !isValidTableName(op.TableName) {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_table",
				"message": fmt.Sprintf("Invalid table name: %s", op.TableName),
			})
		}

		switch strings.ToUpper(op.Operation) {
		case "CREATE":
			if err := applyCreate(tx, op); err != nil {
				tx.Rollback()
				logSyncEnvelope(h.db, req.EnvelopeID, req.DeviceID, "ERROR", len(req.Operations), time.Since(startTime), err.Error())
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"envelope_id": req.EnvelopeID,
					"status":      "error",
					"message":     err.Error(),
				})
			}
		case "UPDATE":
			if err := applyUpdate(tx, op); err != nil {
				tx.Rollback()
				logSyncEnvelope(h.db, req.EnvelopeID, req.DeviceID, "ERROR", len(req.Operations), time.Since(startTime), err.Error())
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"envelope_id": req.EnvelopeID,
					"status":      "error",
					"message":     err.Error(),
				})
			}
		case "DELETE":
			if err := applyDelete(tx, op); err != nil {
				tx.Rollback()
				logSyncEnvelope(h.db, req.EnvelopeID, req.DeviceID, "ERROR", len(req.Operations), time.Since(startTime), err.Error())
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"envelope_id": req.EnvelopeID,
					"status":      "error",
					"message":     err.Error(),
				})
			}
		default:
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_operation",
				"message": fmt.Sprintf("Unknown operation: %s", op.Operation),
			})
		}

		version, err := recordSyncVersion(tx, op.TableName, op.RecordID, op.Operation, req.DeviceID)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "version_failed",
				"message": err.Error(),
			})
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	if err := tx.Commit().Error; err != nil {
		logSyncEnvelope(h.db, req.EnvelopeID, req.DeviceID, "ERROR", len(req.Operations), time.Since(startTime), err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "commit_failed",
			"message": "Failed to commit transaction",
		})
	}

	logSyncEnvelope(h.db, req.EnvelopeID, req.DeviceID, "APPLIED", len(req.Operations), time.Since(startTime), "")

	h.db.Exec("UPDATE _sync_devices SET last_sync_at = ?, last_sync_version = ? WHERE device_id = ?",
		time.Now().UTC(), maxVersion, req.DeviceID)

	return c.JSON(fiber.Map{
		"envelope_id": req.EnvelopeID,
		"status":      "applied",
		"version":     maxVersion,
		"operations":  len(req.Operations),
	})
}

func (h *SyncHandler) PullChanges(c *fiber.Ctx) error {
	sinceVersion := c.QueryInt("since_version", 0)
	deviceID := c.Query("device_id", "")
	limit := c.QueryInt("limit", 1000)

	if limit > 5000 {
		limit = 5000
	}

	var versions []syncVersionRow
	query := h.db.Table("_sync_versions").
		Where("version > ?", sinceVersion).
		Order("version ASC").
		Limit(limit)

	if deviceID != "" {
		query = query.Where("changed_by != ?", deviceID)
	}

	if err := query.Find(&versions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "query_failed",
			"message": err.Error(),
		})
	}

	type recordKey struct {
		Table    string
		RecordID string
	}

	latestIdx := make(map[recordKey]int)
	for i, v := range versions {
		latestIdx[recordKey{v.TableName, v.RecordID}] = i
	}

	type changeEntry struct {
		TableName     string                 `json:"table_name"`
		RecordID      string                 `json:"record_id"`
		Operation     string                 `json:"operation"`
		Data          map[string]interface{} `json:"data"`
		Version       int64                  `json:"version"`
		ChangedFields []string               `json:"changed_fields,omitempty"`
	}

	changes := make([]changeEntry, 0, len(latestIdx))
	var maxVersion int64

	for i, v := range versions {
		if v.Version > maxVersion {
			maxVersion = v.Version
		}

		if latestIdx[recordKey{v.TableName, v.RecordID}] != i {
			continue
		}

		if !isValidTableName(v.TableName) {
			continue
		}

		entry := changeEntry{
			TableName: v.TableName,
			RecordID:  v.RecordID,
			Operation: v.Operation,
			Version:   v.Version,
		}

		if v.Operation == "DELETE" {
			entry.Data = map[string]interface{}{"id": v.RecordID}
		} else {
			var row map[string]interface{}
			if err := h.db.Table(v.TableName).Where("id = ?", v.RecordID).Take(&row).Error; err != nil {
				continue
			}

			if v.Operation == "UPDATE" && v.ChangedFields != "" {
				var fieldNames []string
				if err := json.Unmarshal([]byte(v.ChangedFields), &fieldNames); err == nil && len(fieldNames) > 0 {
					entry.ChangedFields = fieldNames
					delta := map[string]interface{}{"id": v.RecordID}
					for _, fn := range fieldNames {
						if val, ok := row[fn]; ok {
							delta[fn] = val
						}
					}
					entry.Data = delta
				} else {
					entry.Data = row
				}
			} else {
				entry.Data = row
			}
		}

		changes = append(changes, entry)
	}

	return c.JSON(fiber.Map{
		"changes":     changes,
		"max_version": maxVersion,
		"count":       len(changes),
	})
}

type updateDeviceRequest struct {
	DeviceName *string `json:"device_name,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}

func (h *SyncHandler) UpdateDevice(c *fiber.Ctx) error {
	deviceID := c.Params("device_id")
	if deviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_device_id",
			"message": "device_id path parameter is required",
		})
	}

	var count int64
	h.db.Table("_sync_devices").Where("device_id = ?", deviceID).Count(&count)
	if count == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "device_not_found",
			"message": "Device not registered",
		})
	}

	var req updateDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	updates := make(map[string]interface{})

	if req.DeviceName != nil {
		updates["device_name"] = *req.DeviceName
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		if !*req.IsActive {
			now := time.Now().UTC()
			updates["deactivated_at"] = now
			updates["deactivated_reason"] = req.Reason
		} else {
			updates["deactivated_at"] = nil
			updates["deactivated_reason"] = nil
		}
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "no_updates",
			"message": "No fields to update",
		})
	}

	if err := h.db.Table("_sync_devices").Where("device_id = ?", deviceID).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"device_id": deviceID,
		"updated":   updates,
	})
}

func (h *SyncHandler) CacheAuth(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "Auth cache endpoint — coming in Phase 5",
	})
}

func (h *SyncHandler) DeviceStatus(c *fiber.Ctx) error {
	deviceID := c.Query("device_id", "")
	if deviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_device_id",
			"message": "device_id query parameter is required",
		})
	}

	type deviceRow struct {
		DeviceID        string  `gorm:"column:device_id" json:"device_id"`
		DevicePrefix    string  `gorm:"column:device_prefix" json:"device_prefix"`
		Platform        string  `gorm:"column:platform" json:"platform"`
		LastSyncAt      *string `gorm:"column:last_sync_at" json:"last_sync_at"`
		LastSyncVersion int64   `gorm:"column:last_sync_version" json:"last_sync_version"`
		IsActive        bool    `gorm:"column:is_active" json:"is_active"`
	}

	var device deviceRow
	err := h.db.Table("_sync_devices").Where("device_id = ?", deviceID).Take(&device).Error
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "device_not_found",
			"message": "Device not registered",
		})
	}

	var pendingConflicts int64
	h.db.Table("_sync_conflicts").Where("device_id = ? AND reviewed_at IS NULL", deviceID).Count(&pendingConflicts)

	return c.JSON(fiber.Map{
		"device":            device,
		"pending_conflicts": pendingConflicts,
	})
}

type schemaField struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"`
	Model    string   `json:"model,omitempty"`
}

type schemaModel struct {
	Name       string        `json:"name"`
	Module     string        `json:"module"`
	TableName  string        `json:"table_name"`
	Fields     []schemaField `json:"fields"`
	PrimaryKey interface{}   `json:"primary_key,omitempty"`
}

func (h *SyncHandler) GetSchema(c *fiber.Ctx) error {
	allModels := h.modelReg.List()
	var offlineModels []schemaModel

	for _, m := range allModels {
		if !m.OfflineModule {
			continue
		}

		fields := make([]schemaField, 0, len(m.Fields))
		for name, f := range m.Fields {
			sf := schemaField{
				Name:     name,
				Type:     string(f.Type),
				Required: f.Required,
			}
			if len(f.Options) > 0 {
				sf.Options = f.Options
			}
			if f.Model != "" {
				sf.Model = f.Model
			}
			fields = append(fields, sf)
		}
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})

		tableName := h.modelReg.TableName(m.Name)
		if tableName == "" {
			tableName = m.Name
		}

		sm := schemaModel{
			Name:      m.Name,
			Module:    m.Module,
			TableName: tableName,
			Fields:    fields,
		}
		if m.PrimaryKey != nil {
			sm.PrimaryKey = m.PrimaryKey
		}

		offlineModels = append(offlineModels, sm)
	}

	return c.JSON(fiber.Map{
		"models": offlineModels,
	})
}

func generateDeviceID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "DEV-" + hex.EncodeToString(b)
}

func generateDevicePrefix() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%03d-%s", b[0]%100, string(rune('A'+int(b[1]%26))))
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

var validTableNameChars = func() [256]bool {
	var t [256]bool
	for c := 'a'; c <= 'z'; c++ {
		t[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		t[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		t[c] = true
	}
	t['_'] = true
	return t
}()

func isValidTableName(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !validTableNameChars[name[i]] {
			return false
		}
	}
	return true
}

func applyCreate(tx *gorm.DB, op pushOperation) error {
	if len(op.Payload) == 0 {
		return fmt.Errorf("empty payload for CREATE on %s", op.TableName)
	}

	columns := make([]string, 0, len(op.Payload))
	placeholders := make([]string, 0, len(op.Payload))
	values := make([]interface{}, 0, len(op.Payload))

	for k, v := range op.Payload {
		if !isValidTableName(k) {
			return fmt.Errorf("invalid column name: %s", k)
		}
		columns = append(columns, k)
		placeholders = append(placeholders, "?")
		values = append(values, v)
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		op.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return tx.Exec(sql, values...).Error
}

func applyUpdate(tx *gorm.DB, op pushOperation) error {
	if len(op.Payload) == 0 {
		return fmt.Errorf("empty payload for UPDATE on %s", op.TableName)
	}

	setParts := make([]string, 0, len(op.Payload))
	values := make([]interface{}, 0, len(op.Payload))

	for k, v := range op.Payload {
		if k == "id" {
			continue
		}
		if !isValidTableName(k) {
			return fmt.Errorf("invalid column name: %s", k)
		}
		setParts = append(setParts, k+" = ?")
		values = append(values, v)
	}

	if len(setParts) == 0 {
		return nil
	}

	values = append(values, op.RecordID)
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", op.TableName, strings.Join(setParts, ", "))

	return tx.Exec(sql, values...).Error
}

func applyDelete(tx *gorm.DB, op pushOperation) error {
	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?", op.TableName)
	return tx.Exec(sql, op.RecordID).Error
}

func recordSyncVersion(tx *gorm.DB, tableName, recordID, operation, changedBy string) (int64, error) {
	dialect := persistence.DetectDialect(tx)
	now := time.Now().UTC()

	if dialect == persistence.DialectPostgres {
		var newVersion int64
		err := tx.Raw(
			`INSERT INTO _sync_versions (table_name, record_id, operation, changed_by, created_at)
			 VALUES ($1, $2, $3, $4, $5) RETURNING version`,
			tableName, recordID, operation, changedBy, now,
		).Scan(&newVersion).Error
		return newVersion, err
	}

	err := tx.Exec(
		`INSERT INTO _sync_versions (table_name, record_id, operation, version, changed_by, created_at)
		 SELECT ?, ?, ?, COALESCE(MAX(version), 0) + 1, ?, ? FROM _sync_versions`,
		tableName, recordID, operation, changedBy, now,
	).Error
	if err != nil {
		return 0, err
	}

	var newVersion int64
	tx.Raw("SELECT MAX(version) FROM _sync_versions WHERE table_name = ? AND record_id = ? AND changed_by = ?",
		tableName, recordID, changedBy).Scan(&newVersion)
	return newVersion, nil
}

func logSyncEnvelope(db *gorm.DB, envelopeID, deviceID, status string, opsCount int, duration time.Duration, errMsg string) {
	db.Exec(
		"INSERT INTO _sync_log (envelope_id, device_id, received_at, status, operations_count, processing_time_ms, error_message) VALUES (?, ?, ?, ?, ?, ?, ?)",
		envelopeID, deviceID, time.Now().UTC(), status, opsCount, duration.Milliseconds(), nullIfEmpty(errMsg),
	)
}
