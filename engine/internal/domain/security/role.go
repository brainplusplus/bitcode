package security

import (
	"time"

	"github.com/bitcode-engine/engine/pkg/ddd"
)

type Role struct {
	ddd.BaseEntity
	Name        string       `json:"name" gorm:"uniqueIndex;size:100"`
	DisplayName string       `json:"display_name" gorm:"size:200"`
	Permissions []Permission `json:"permissions" gorm:"many2many:role_permissions;"`
	Inherits    []Role       `json:"inherits" gorm:"many2many:role_inherits;"`
}

func NewRole(id string, name string, displayName string) *Role {
	return &Role{
		BaseEntity:  ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:        name,
		DisplayName: displayName,
	}
}

func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p.Name == permission {
			return true
		}
	}
	for _, parent := range r.Inherits {
		if parent.HasPermission(permission) {
			return true
		}
	}
	return false
}

func (r *Role) AddPermission(p Permission) {
	for _, existing := range r.Permissions {
		if existing.Name == p.Name {
			return
		}
	}
	r.Permissions = append(r.Permissions, p)
	r.UpdatedAt = time.Now()
}

func (r *Role) AllPermissions() []string {
	seen := make(map[string]bool)
	r.collectPermissions(seen)
	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	return result
}

func (r *Role) collectPermissions(seen map[string]bool) {
	for _, p := range r.Permissions {
		seen[p.Name] = true
	}
	for _, parent := range r.Inherits {
		parent.collectPermissions(seen)
	}
}
