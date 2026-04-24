package persistence

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DataRevision struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	ModelName string    `gorm:"type:text;index:idx_dr_model_record;index:idx_dr_model_record_ver" json:"model_name"`
	RecordID  string    `gorm:"type:text;index:idx_dr_model_record;index:idx_dr_model_record_ver" json:"record_id"`
	Version   int       `gorm:"index:idx_dr_model_record_ver" json:"version"`
	Action    string    `gorm:"type:text" json:"action"`
	Snapshot  string    `gorm:"type:text" json:"snapshot"`
	Changes   string    `gorm:"type:text" json:"changes"`
	UserID    string    `gorm:"type:text" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type DataRevisionRepository struct {
	db *gorm.DB
}

func NewDataRevisionRepository(db *gorm.DB) *DataRevisionRepository {
	return &DataRevisionRepository{db: db}
}

func AutoMigrateDataRevisions(db *gorm.DB) error {
	return db.AutoMigrate(&DataRevision{})
}

func (r *DataRevisionRepository) Create(modelName, recordID, action string, snapshot map[string]any, changes map[string]any, userID string) (*DataRevision, error) {
	var maxVersion int
	r.db.Model(&DataRevision{}).
		Where("model_name = ? AND record_id = ?", modelName, recordID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion)

	snapshotJSON, _ := json.Marshal(snapshot)
	changesJSON, _ := json.Marshal(changes)

	rev := &DataRevision{
		ID:        uuid.New().String(),
		ModelName: modelName,
		RecordID:  recordID,
		Version:   maxVersion + 1,
		Action:    action,
		Snapshot:  string(snapshotJSON),
		Changes:   string(changesJSON),
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	if err := r.db.Create(rev).Error; err != nil {
		return nil, err
	}
	return rev, nil
}

func (r *DataRevisionRepository) ListByRecord(modelName, recordID string, limit int) ([]DataRevision, error) {
	var revisions []DataRevision
	q := r.db.Where("model_name = ? AND record_id = ?", modelName, recordID).Order("version DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&revisions).Error; err != nil {
		return nil, err
	}
	return revisions, nil
}

func (r *DataRevisionRepository) GetByVersion(modelName, recordID string, version int) (*DataRevision, error) {
	var rev DataRevision
	if err := r.db.Where("model_name = ? AND record_id = ? AND version = ?", modelName, recordID, version).First(&rev).Error; err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *DataRevisionRepository) GetLatest(modelName, recordID string) (*DataRevision, error) {
	var rev DataRevision
	if err := r.db.Where("model_name = ? AND record_id = ?", modelName, recordID).Order("version DESC").First(&rev).Error; err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *DataRevisionRepository) Cleanup(modelName, recordID string, keepN int) error {
	if keepN <= 0 {
		return nil
	}

	var count int64
	r.db.Model(&DataRevision{}).Where("model_name = ? AND record_id = ?", modelName, recordID).Count(&count)

	if count <= int64(keepN) {
		return nil
	}

	var cutoff DataRevision
	if err := r.db.Where("model_name = ? AND record_id = ?", modelName, recordID).
		Order("version DESC").Offset(keepN).First(&cutoff).Error; err != nil {
		return nil
	}

	return r.db.Where("model_name = ? AND record_id = ? AND version <= ?", modelName, recordID, cutoff.Version).
		Delete(&DataRevision{}).Error
}

func (r *DataRevisionRepository) Count(modelName, recordID string) int64 {
	var count int64
	r.db.Model(&DataRevision{}).Where("model_name = ? AND record_id = ?", modelName, recordID).Count(&count)
	return count
}

func (r *DataRevisionRepository) GetSnapshotMap(rev *DataRevision) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(rev.Snapshot), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func ComputeChanges(before, after map[string]any) map[string]any {
	changes := make(map[string]any)
	for key, newVal := range after {
		oldVal, exists := before[key]
		if !exists {
			changes[key] = map[string]any{"old": nil, "new": newVal}
		} else {
			oldStr := jsonString(oldVal)
			newStr := jsonString(newVal)
			if oldStr != newStr {
				changes[key] = map[string]any{"old": oldVal, "new": newVal}
			}
		}
	}
	for key, oldVal := range before {
		if _, exists := after[key]; !exists {
			changes[key] = map[string]any{"old": oldVal, "new": nil}
		}
	}
	return changes
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
