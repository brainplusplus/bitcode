package security

import (
	"encoding/json"
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
)

type SecurityHistory struct {
	ddd.BaseEntity
	EntityType string `json:"entity_type" gorm:"size:50;index"`
	EntityID   string `json:"entity_id" gorm:"size:100;index"`
	EntityName string `json:"entity_name" gorm:"size:200"`
	Action     string `json:"action" gorm:"size:20"`
	Changes    string `json:"changes" gorm:"type:text"`
	Snapshot   string `json:"snapshot" gorm:"type:text"`
	UserID     string `json:"user_id" gorm:"size:100;index"`
	Source     string `json:"source" gorm:"size:50"`
	Module     string `json:"module" gorm:"size:100"`
}

func NewSecurityHistory(id, entityType, entityID, entityName, action string, changes, snapshot any, userID, source, module string) *SecurityHistory {
	changesStr := ""
	if changes != nil {
		if b, err := json.Marshal(changes); err == nil {
			changesStr = string(b)
		}
	}

	snapshotStr := ""
	if snapshot != nil {
		if b, err := json.Marshal(snapshot); err == nil {
			snapshotStr = string(b)
		}
	}

	return &SecurityHistory{
		BaseEntity: ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EntityType: entityType,
		EntityID:   entityID,
		EntityName: entityName,
		Action:     action,
		Changes:    changesStr,
		Snapshot:   snapshotStr,
		UserID:     userID,
		Source:     source,
		Module:     module,
	}
}
