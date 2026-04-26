package persistence

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type MigrationRecord struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"size:255;not null;index"`
	Module    string    `gorm:"size:100;not null;index"`
	Batch     int       `gorm:"not null;index"`
	Model     string    `gorm:"size:255"`
	Source    string    `gorm:"size:50"`
	Records   int       `gorm:"default:0"`
	Status    string    `gorm:"size:20;default:completed"`
	Error     string    `gorm:"type:text"`
	Duration  int64     `gorm:"default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (MigrationRecord) TableName() string {
	return "ir_migration"
}

type MigrationTracker struct {
	db *gorm.DB
}

func NewMigrationTracker(db *gorm.DB) *MigrationTracker {
	return &MigrationTracker{db: db}
}

func AutoMigrateMigrationTracker(db *gorm.DB) error {
	return db.AutoMigrate(&MigrationRecord{})
}

func (t *MigrationTracker) HasRun(module, name string) bool {
	var count int64
	t.db.Model(&MigrationRecord{}).
		Where("module = ? AND name = ? AND status = ?", module, name, "completed").
		Count(&count)
	return count > 0
}

func (t *MigrationTracker) NextBatch() int {
	var maxBatch int
	t.db.Model(&MigrationRecord{}).Select("COALESCE(MAX(batch), 0)").Scan(&maxBatch)
	return maxBatch + 1
}

func (t *MigrationTracker) CurrentBatch() int {
	var maxBatch int
	t.db.Model(&MigrationRecord{}).Select("COALESCE(MAX(batch), 0)").Scan(&maxBatch)
	return maxBatch
}

func (t *MigrationTracker) Record(module, name, model, source string, records int, batch int, duration time.Duration, err error) error {
	rec := MigrationRecord{
		Name:     name,
		Module:   module,
		Batch:    batch,
		Model:    model,
		Source:   source,
		Records:  records,
		Status:   "completed",
		Duration: duration.Milliseconds(),
	}
	if err != nil {
		rec.Status = "failed"
		rec.Error = err.Error()
	}
	return t.db.Create(&rec).Error
}

func (t *MigrationTracker) RemoveBatch(batch int) error {
	return t.db.Where("batch = ?", batch).Delete(&MigrationRecord{}).Error
}

func (t *MigrationTracker) RemoveByName(module, name string) error {
	return t.db.Where("module = ? AND name = ?", module, name).Delete(&MigrationRecord{}).Error
}

func (t *MigrationTracker) GetByBatch(batch int) ([]MigrationRecord, error) {
	var records []MigrationRecord
	err := t.db.Where("batch = ?", batch).Order("id DESC").Find(&records).Error
	return records, err
}

func (t *MigrationTracker) GetAll() ([]MigrationRecord, error) {
	var records []MigrationRecord
	err := t.db.Order("batch ASC, id ASC").Find(&records).Error
	return records, err
}

func (t *MigrationTracker) GetByModule(module string) ([]MigrationRecord, error) {
	var records []MigrationRecord
	err := t.db.Where("module = ?", module).Order("batch ASC, id ASC").Find(&records).Error
	return records, err
}

func (t *MigrationTracker) GetPending(module string, allNames []string) []string {
	var ran []string
	t.db.Model(&MigrationRecord{}).
		Where("module = ? AND status = ?", module, "completed").
		Pluck("name", &ran)

	ranMap := make(map[string]bool, len(ran))
	for _, n := range ran {
		ranMap[n] = true
	}

	var pending []string
	for _, name := range allNames {
		if !ranMap[name] {
			pending = append(pending, name)
		}
	}
	return pending
}

func (t *MigrationTracker) Reset(module string) error {
	if module == "" {
		return t.db.Where("1 = 1").Delete(&MigrationRecord{}).Error
	}
	return t.db.Where("module = ?", module).Delete(&MigrationRecord{}).Error
}

func (t *MigrationTracker) Status() ([]MigrationStatusEntry, error) {
	var records []MigrationRecord
	if err := t.db.Order("batch ASC, id ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	entries := make([]MigrationStatusEntry, len(records))
	for i, r := range records {
		entries[i] = MigrationStatusEntry{
			Name:      r.Name,
			Module:    r.Module,
			Batch:     r.Batch,
			Model:     r.Model,
			Source:    r.Source,
			Records:   r.Records,
			Status:    r.Status,
			Duration:  fmt.Sprintf("%dms", r.Duration),
			RanAt:     r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return entries, nil
}

type MigrationStatusEntry struct {
	Name     string
	Module   string
	Batch    int
	Model    string
	Source   string
	Records  int
	Status   string
	Duration string
	RanAt    string
}
