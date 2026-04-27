package persistence

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type ModelPermissions struct {
	CanSelect bool `json:"can_select"`
	CanRead   bool `json:"can_read"`
	CanWrite  bool `json:"can_write"`
	CanCreate bool `json:"can_create"`
	CanDelete bool `json:"can_delete"`
	CanPrint  bool `json:"can_print"`
	CanEmail  bool `json:"can_email"`
	CanReport bool `json:"can_report"`
	CanExport bool `json:"can_export"`
	CanImport bool `json:"can_import"`
	CanMask   bool `json:"can_mask"`
	CanClone  bool `json:"can_clone"`
}

func (p *ModelPermissions) Has(operation string) bool {
	switch operation {
	case "select":
		return p.CanSelect
	case "read":
		return p.CanRead
	case "write":
		return p.CanWrite
	case "create":
		return p.CanCreate
	case "delete":
		return p.CanDelete
	case "print":
		return p.CanPrint
	case "email":
		return p.CanEmail
	case "report":
		return p.CanReport
	case "export":
		return p.CanExport
	case "import":
		return p.CanImport
	case "mask":
		return p.CanMask
	case "clone":
		return p.CanClone
	default:
		return false
	}
}

type PermissionService struct {
	db        *gorm.DB
	tableName func(string) string
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db, tableName: func(n string) string { return n }}
}

func (s *PermissionService) SetTableNameResolver(fn func(string) string) {
	s.tableName = fn
}

func (s *PermissionService) tn(model string) string {
	return s.tableName(model)
}

func (s *PermissionService) UserHasPermission(userID string, permission string) (bool, error) {
	parts := strings.SplitN(permission, ".", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid permission format: %q (expected model.operation)", permission)
	}
	modelName := parts[0]
	operation := parts[1]

	perms, err := s.GetModelPermissions(userID, modelName)
	if err != nil {
		return false, err
	}
	return perms.Has(operation), nil
}

func (p *ModelPermissions) ToMap() map[string]bool {
	return map[string]bool{
		"can_select": p.CanSelect,
		"can_read":   p.CanRead,
		"can_write":  p.CanWrite,
		"can_create": p.CanCreate,
		"can_delete": p.CanDelete,
		"can_print":  p.CanPrint,
		"can_email":  p.CanEmail,
		"can_report": p.CanReport,
		"can_export": p.CanExport,
		"can_import": p.CanImport,
		"can_mask":   p.CanMask,
		"can_clone":  p.CanClone,
	}
}

func (s *PermissionService) GetModelPermissions(userID string, modelName string) (*ModelPermissions, error) {
	var user struct {
		IsSuperuser bool
	}
	if err := s.db.Table(s.tn("user")).Select("is_superuser").Where("id = ?", userID).First(&user).Error; err != nil {
		return &ModelPermissions{}, nil
	}
	if user.IsSuperuser {
		return &ModelPermissions{
			CanSelect: true, CanRead: true, CanWrite: true, CanCreate: true,
			CanDelete: true, CanPrint: true, CanEmail: true, CanReport: true,
			CanExport: true, CanImport: true, CanMask: true, CanClone: true,
		}, nil
	}

	groupIDs, err := s.ResolveUserGroupIDs(userID)
	if err != nil {
		return &ModelPermissions{}, err
	}

	var accessList []struct {
		CanSelect bool
		CanRead   bool
		CanWrite  bool
		CanCreate bool
		CanDelete bool
		CanPrint  bool
		CanEmail  bool
		CanReport bool
		CanExport bool
		CanImport bool
		CanMask   bool
		CanClone  bool
	}

	query := s.db.Table(s.tn("model_access")).
		Select("can_select, can_read, can_write, can_create, can_delete, can_print, can_email, can_report, can_export, can_import, can_mask, can_clone").
		Where("model_name = ?", modelName)

	if len(groupIDs) > 0 {
		query = query.Where("group_id = '' OR group_id IN ?", groupIDs)
	} else {
		query = query.Where("group_id = ''")
	}

	if err := query.Find(&accessList).Error; err != nil {
		return &ModelPermissions{}, err
	}

	result := &ModelPermissions{}
	for _, a := range accessList {
		if a.CanSelect {
			result.CanSelect = true
		}
		if a.CanRead {
			result.CanRead = true
		}
		if a.CanWrite {
			result.CanWrite = true
		}
		if a.CanCreate {
			result.CanCreate = true
		}
		if a.CanDelete {
			result.CanDelete = true
		}
		if a.CanPrint {
			result.CanPrint = true
		}
		if a.CanEmail {
			result.CanEmail = true
		}
		if a.CanReport {
			result.CanReport = true
		}
		if a.CanExport {
			result.CanExport = true
		}
		if a.CanImport {
			result.CanImport = true
		}
		if a.CanMask {
			result.CanMask = true
		}
		if a.CanClone {
			result.CanClone = true
		}
	}

	return result, nil
}

func (s *PermissionService) ResolveUserGroupIDs(userID string) ([]string, error) {
	var directGroupIDs []string
	if err := s.db.Table(s.tn("user") + "_" + s.tn("group")).
		Select("group_id").
		Where("user_id = ?", userID).
		Pluck("group_id", &directGroupIDs).Error; err != nil {
		return nil, err
	}

	if len(directGroupIDs) == 0 {
		return nil, nil
	}

	allGroupIDs := make(map[string]bool)
	queue := make([]string, len(directGroupIDs))
	copy(queue, directGroupIDs)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if allGroupIDs[current] {
			continue
		}
		allGroupIDs[current] = true

		var impliedIDs []string
		if err := s.db.Table(s.tn("group") + "_implies").
			Select("implied_group_id").
			Where("group_id = ?", current).
			Pluck("implied_group_id", &impliedIDs).Error; err != nil {
			continue
		}
		queue = append(queue, impliedIDs...)
	}

	result := make([]string, 0, len(allGroupIDs))
	for id := range allGroupIDs {
		result = append(result, id)
	}
	return result, nil
}
