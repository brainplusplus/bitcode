package security

import (
	"fmt"
	"strings"
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
)

type RecordRule struct {
	ddd.BaseEntity
	Name         string `json:"name" gorm:"uniqueIndex;size:100"`
	ModelName    string `json:"model_name" gorm:"size:100;index"`
	GroupNames   string `json:"group_names" gorm:"size:500"`
	DomainFilter string `json:"domain_filter" gorm:"type:jsonb"`
	CanRead      bool   `json:"can_read" gorm:"default:true"`
	CanCreate    bool   `json:"can_create" gorm:"default:true"`
	CanWrite     bool   `json:"can_write" gorm:"default:true"`
	CanDelete    bool   `json:"can_delete" gorm:"default:false"`
	Global       bool   `json:"global" gorm:"default:false"`
	Active       bool   `json:"active" gorm:"default:true"`
}

func NewRecordRule(id string, name string, modelName string, groups []string) *RecordRule {
	return &RecordRule{
		BaseEntity: ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:       name,
		ModelName:  modelName,
		GroupNames: strings.Join(groups, ","),
		CanRead:    true,
		CanCreate:  true,
		CanWrite:   true,
		CanDelete:  false,
		Active:     true,
	}
}

func (r *RecordRule) GetGroups() []string {
	if r.GroupNames == "" {
		return nil
	}
	return strings.Split(r.GroupNames, ",")
}

func (r *RecordRule) AppliesToGroup(userGroups []string) bool {
	if r.Global {
		return true
	}
	ruleGroups := r.GetGroups()
	for _, rg := range ruleGroups {
		for _, ug := range userGroups {
			if rg == ug {
				return true
			}
		}
	}
	return false
}

func (r *RecordRule) AppliesToOperation(operation string) bool {
	switch operation {
	case "read":
		return r.CanRead
	case "create":
		return r.CanCreate
	case "write":
		return r.CanWrite
	case "delete":
		return r.CanDelete
	default:
		return false
	}
}

type Condition struct {
	Field    string
	Operator string
	Value    string
}

func InterpolateDomain(domain string, vars map[string]string) string {
	result := domain
	for key, val := range vars {
		result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", key), val)
	}
	return result
}
