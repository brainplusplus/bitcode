package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	domainstorage "github.com/bitcode-framework/bitcode/internal/domain/storage"
	"gorm.io/gorm"
)

type AttachmentRepository struct {
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) *AttachmentRepository {
	return &AttachmentRepository{db: db}
}

func AutoMigrateAttachments(db *gorm.DB) error {
	return db.AutoMigrate(&domainstorage.Attachment{})
}

func (r *AttachmentRepository) Create(att *domainstorage.Attachment) error {
	if att.ID == "" {
		att.ID = uuid.New().String()
	}
	if att.CreatedAt.IsZero() {
		att.CreatedAt = time.Now()
	}
	att.UpdatedAt = time.Now()
	return r.db.Create(att).Error
}

func (r *AttachmentRepository) FindByID(id string) (*domainstorage.Attachment, error) {
	var att domainstorage.Attachment
	if err := r.db.Where("id = ? AND active = ?", id, true).First(&att).Error; err != nil {
		return nil, fmt.Errorf("attachment not found: %w", err)
	}
	return &att, nil
}

func (r *AttachmentRepository) FindByIDIncludeInactive(id string) (*domainstorage.Attachment, error) {
	var att domainstorage.Attachment
	if err := r.db.Where("id = ?", id).First(&att).Error; err != nil {
		return nil, fmt.Errorf("attachment not found: %w", err)
	}
	return &att, nil
}

func (r *AttachmentRepository) FindByHash(hash, model, recordID, fieldName string) (*domainstorage.Attachment, error) {
	var att domainstorage.Attachment
	query := r.db.Where("hash = ? AND active = ?", hash, true)
	if model != "" {
		query = query.Where("model = ?", model)
	}
	if recordID != "" {
		query = query.Where("record_id = ?", recordID)
	}
	if fieldName != "" {
		query = query.Where("field_name = ?", fieldName)
	}
	if err := query.First(&att).Error; err != nil {
		return nil, err
	}
	return &att, nil
}

func (r *AttachmentRepository) FindByModelRecord(model, recordID, fieldName string, page, pageSize int) ([]domainstorage.Attachment, int64, error) {
	query := r.db.Where("active = ?", true)
	if model != "" {
		query = query.Where("model = ?", model)
	}
	if recordID != "" {
		query = query.Where("record_id = ?", recordID)
	}
	if fieldName != "" {
		query = query.Where("field_name = ?", fieldName)
	}

	var total int64
	if err := query.Model(&domainstorage.Attachment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var attachments []domainstorage.Attachment
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&attachments).Error; err != nil {
		return nil, 0, err
	}

	return attachments, total, nil
}

func (r *AttachmentRepository) FindVersions(parentID string) ([]domainstorage.Attachment, error) {
	var attachments []domainstorage.Attachment
	if err := r.db.Where("(parent_id = ? OR id = ?) AND active = ?", parentID, parentID, true).
		Order("version DESC").Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *AttachmentRepository) GetLatestVersion(model, recordID, fieldName, parentID string) (int, error) {
	var maxVersion int
	query := r.db.Model(&domainstorage.Attachment{}).Where("active = ?", true)
	if parentID != "" {
		query = query.Where("parent_id = ? OR id = ?", parentID, parentID)
	} else {
		query = query.Where("model = ? AND record_id = ? AND field_name = ?", model, recordID, fieldName)
	}
	query.Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)
	return maxVersion, nil
}

func (r *AttachmentRepository) Update(att *domainstorage.Attachment) error {
	att.UpdatedAt = time.Now()
	return r.db.Save(att).Error
}

func (r *AttachmentRepository) SoftDelete(id string) error {
	return r.db.Model(&domainstorage.Attachment{}).Where("id = ?", id).
		Updates(map[string]any{"active": false, "updated_at": time.Now()}).Error
}

func (r *AttachmentRepository) HardDelete(id string) error {
	return r.db.Where("id = ?", id).Delete(&domainstorage.Attachment{}).Error
}

func (r *AttachmentRepository) FindOrphans(olderThan time.Time) ([]domainstorage.Attachment, error) {
	var attachments []domainstorage.Attachment
	if err := r.db.Where("record_id = '' AND created_at < ? AND active = ?", olderThan, true).
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *AttachmentRepository) CleanupVersions(parentID string, keepN int) error {
	if keepN <= 0 {
		return nil
	}

	var count int64
	r.db.Model(&domainstorage.Attachment{}).Where("parent_id = ? AND active = ?", parentID, true).Count(&count)

	if count <= int64(keepN) {
		return nil
	}

	var cutoff domainstorage.Attachment
	if err := r.db.Where("parent_id = ? AND active = ?", parentID, true).
		Order("version DESC").Offset(keepN).First(&cutoff).Error; err != nil {
		return nil
	}

	return r.db.Model(&domainstorage.Attachment{}).
		Where("parent_id = ? AND version <= ? AND active = ?", parentID, cutoff.Version, true).
		Updates(map[string]any{"active": false, "updated_at": time.Now()}).Error
}
