package security

import (
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
)

type Permission struct {
	ddd.BaseEntity
	Name        string `json:"name" gorm:"uniqueIndex;size:100"`
	DisplayName string `json:"display_name" gorm:"size:200"`
	Module      string `json:"module" gorm:"size:100;index"`
}

func NewPermission(id string, name string, displayName string, module string) *Permission {
	return &Permission{
		BaseEntity:  ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:        name,
		DisplayName: displayName,
		Module:      module,
	}
}
