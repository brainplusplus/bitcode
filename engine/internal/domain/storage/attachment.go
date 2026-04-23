package storage

import (
	"time"
)

type Attachment struct {
	ID            string    `gorm:"primaryKey;type:text" json:"id"`
	TenantID      string    `gorm:"type:text;index:idx_att_tenant" json:"tenant_id,omitempty"`
	UserID        string    `gorm:"type:text;index:idx_att_user" json:"user_id,omitempty"`
	Model         string    `gorm:"type:text;index:idx_att_model_record" json:"model,omitempty"`
	RecordID      string    `gorm:"type:text;index:idx_att_model_record" json:"record_id,omitempty"`
	FieldName     string    `gorm:"type:text;index:idx_att_model_record" json:"field_name,omitempty"`
	Name          string    `gorm:"type:text;not null" json:"name"`
	Path          string    `gorm:"type:text;not null" json:"path"`
	URL           string    `gorm:"type:text;not null" json:"url"`
	Storage       string    `gorm:"type:text;not null" json:"storage"`
	Size          int64     `gorm:"not null;default:0" json:"size"`
	MimeType      string    `gorm:"type:text;not null" json:"mime_type"`
	Ext           string    `gorm:"type:text;not null" json:"ext"`
	Hash          string    `gorm:"type:text;not null;index:idx_att_hash" json:"hash"`
	IsPublic      bool      `gorm:"not null;default:false" json:"is_public"`
	Version       int       `gorm:"not null;default:1" json:"version"`
	ParentID      string    `gorm:"type:text;index:idx_att_parent" json:"parent_id,omitempty"`
	ThumbnailPath string    `gorm:"type:text" json:"thumbnail_path,omitempty"`
	Metadata      string    `gorm:"type:text" json:"metadata,omitempty"`
	Active        bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Attachment) TableName() string {
	return "attachments"
}
