package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ViewRevision struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	ViewKey   string    `gorm:"type:text;index:idx_vr_key;index:idx_vr_key_ver" json:"view_key"`
	Version   int       `gorm:"index:idx_vr_key_ver" json:"version"`
	Content   string    `gorm:"type:text" json:"content"`
	CreatedBy string    `gorm:"type:text" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type ViewRevisionRepository struct {
	db *gorm.DB
}

func NewViewRevisionRepository(db *gorm.DB) *ViewRevisionRepository {
	return &ViewRevisionRepository{db: db}
}

func AutoMigrateViewRevisions(db *gorm.DB) error {
	return db.AutoMigrate(&ViewRevision{})
}

func (r *ViewRevisionRepository) Create(viewKey, content, createdBy string) (*ViewRevision, error) {
	var maxVersion int
	r.db.Model(&ViewRevision{}).Where("view_key = ?", viewKey).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)

	rev := &ViewRevision{
		ID:        uuid.New().String(),
		ViewKey:   viewKey,
		Version:   maxVersion + 1,
		Content:   content,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	if err := r.db.Create(rev).Error; err != nil {
		return nil, err
	}
	return rev, nil
}

func (r *ViewRevisionRepository) ListByViewKey(viewKey string, limit int) ([]ViewRevision, error) {
	var revisions []ViewRevision
	q := r.db.Where("view_key = ?", viewKey).Order("version DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&revisions).Error; err != nil {
		return nil, err
	}
	return revisions, nil
}

func (r *ViewRevisionRepository) GetByVersion(viewKey string, version int) (*ViewRevision, error) {
	var rev ViewRevision
	if err := r.db.Where("view_key = ? AND version = ?", viewKey, version).First(&rev).Error; err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *ViewRevisionRepository) GetLatest(viewKey string) (*ViewRevision, error) {
	var rev ViewRevision
	if err := r.db.Where("view_key = ?", viewKey).Order("version DESC").First(&rev).Error; err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *ViewRevisionRepository) Cleanup(viewKey string, keepN int) error {
	if keepN <= 0 {
		return nil
	}

	var count int64
	r.db.Model(&ViewRevision{}).Where("view_key = ?", viewKey).Count(&count)

	if count <= int64(keepN) {
		return nil
	}

	var cutoff ViewRevision
	if err := r.db.Where("view_key = ?", viewKey).Order("version DESC").Offset(keepN).First(&cutoff).Error; err != nil {
		return nil
	}

	return r.db.Where("view_key = ? AND version <= ?", viewKey, cutoff.Version).Delete(&ViewRevision{}).Error
}

func (r *ViewRevisionRepository) Count(viewKey string) int64 {
	var count int64
	r.db.Model(&ViewRevision{}).Where("view_key = ?", viewKey).Count(&count)
	return count
}
