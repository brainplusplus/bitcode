package security

import (
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
)

type ModelAccess struct {
	ddd.BaseEntity
	Name           string `json:"name" gorm:"size:200"`
	ModelName      string `json:"model_name" gorm:"size:100;uniqueIndex:idx_model_access_model_group"`
	GroupID        string `json:"group_id" gorm:"size:100;uniqueIndex:idx_model_access_model_group"`
	CanSelect      bool   `json:"can_select" gorm:"default:false"`
	CanRead        bool   `json:"can_read" gorm:"default:false"`
	CanWrite       bool   `json:"can_write" gorm:"default:false"`
	CanCreate      bool   `json:"can_create" gorm:"default:false"`
	CanDelete      bool   `json:"can_delete" gorm:"default:false"`
	CanPrint       bool   `json:"can_print" gorm:"default:false"`
	CanEmail       bool   `json:"can_email" gorm:"default:false"`
	CanReport      bool   `json:"can_report" gorm:"default:false"`
	CanExport      bool   `json:"can_export" gorm:"default:false"`
	CanImport      bool   `json:"can_import" gorm:"default:false"`
	CanMask        bool   `json:"can_mask" gorm:"default:false"`
	CanClone       bool   `json:"can_clone" gorm:"default:false"`
	Module         string `json:"module" gorm:"size:100;index"`
	ModifiedSource string `json:"modified_source" gorm:"size:20;default:'json'"`
}

func NewModelAccess(id, name, modelName, groupID, module string) *ModelAccess {
	return &ModelAccess{
		BaseEntity: ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:       name,
		ModelName:  modelName,
		GroupID:    groupID,
		Module:     module,
	}
}

func (m *ModelAccess) IsGlobal() bool {
	return m.GroupID == ""
}

func (m *ModelAccess) HasPermission(operation string) bool {
	switch operation {
	case "select":
		return m.CanSelect
	case "read":
		return m.CanRead
	case "write":
		return m.CanWrite
	case "create":
		return m.CanCreate
	case "delete":
		return m.CanDelete
	case "print":
		return m.CanPrint
	case "email":
		return m.CanEmail
	case "report":
		return m.CanReport
	case "export":
		return m.CanExport
	case "import":
		return m.CanImport
	case "mask":
		return m.CanMask
	case "clone":
		return m.CanClone
	default:
		return false
	}
}

func (m *ModelAccess) SetAll(value bool) {
	m.CanSelect = value
	m.CanRead = value
	m.CanWrite = value
	m.CanCreate = value
	m.CanDelete = value
	m.CanPrint = value
	m.CanEmail = value
	m.CanReport = value
	m.CanExport = value
	m.CanImport = value
	m.CanMask = value
	m.CanClone = value
}

func (m *ModelAccess) SetFromList(perms []string) {
	m.SetAll(false)
	for _, p := range perms {
		switch p {
		case "select":
			m.CanSelect = true
		case "read":
			m.CanRead = true
		case "write":
			m.CanWrite = true
		case "create":
			m.CanCreate = true
		case "delete":
			m.CanDelete = true
		case "print":
			m.CanPrint = true
		case "email":
			m.CanEmail = true
		case "report":
			m.CanReport = true
		case "export":
			m.CanExport = true
		case "import":
			m.CanImport = true
		case "mask":
			m.CanMask = true
		case "clone":
			m.CanClone = true
		}
	}
}

func (m *ModelAccess) AllPermissions() []string {
	var perms []string
	if m.CanSelect {
		perms = append(perms, "select")
	}
	if m.CanRead {
		perms = append(perms, "read")
	}
	if m.CanWrite {
		perms = append(perms, "write")
	}
	if m.CanCreate {
		perms = append(perms, "create")
	}
	if m.CanDelete {
		perms = append(perms, "delete")
	}
	if m.CanPrint {
		perms = append(perms, "print")
	}
	if m.CanEmail {
		perms = append(perms, "email")
	}
	if m.CanReport {
		perms = append(perms, "report")
	}
	if m.CanExport {
		perms = append(perms, "export")
	}
	if m.CanImport {
		perms = append(perms, "import")
	}
	if m.CanMask {
		perms = append(perms, "mask")
	}
	if m.CanClone {
		perms = append(perms, "clone")
	}
	return perms
}
