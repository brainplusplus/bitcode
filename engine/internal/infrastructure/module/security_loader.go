package module

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type TableNameFunc func(modelName string) string

type SecurityLoader struct {
	db        *gorm.DB
	tableName TableNameFunc
}

func NewSecurityLoader(db *gorm.DB) *SecurityLoader {
	return &SecurityLoader{db: db, tableName: func(name string) string { return name }}
}

func (l *SecurityLoader) SetTableNameResolver(fn TableNameFunc) {
	l.tableName = fn
}

func (l *SecurityLoader) tn(model string) string {
	return l.tableName(model)
}

func (l *SecurityLoader) LoadFromDirectory(secDir string, moduleName string) error {
	entries, err := os.ReadDir(secDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(secDir, entry.Name())
		if err := l.LoadFile(path, moduleName); err != nil {
			log.Printf("[WARN] failed to load security file %s: %v", path, err)
		}
	}
	return nil
}

func (l *SecurityLoader) LoadFile(path string, moduleName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	secDef, err := parser.ParseSecurity(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	return l.SyncToDB(secDef, moduleName)
}

func (l *SecurityLoader) SyncToDB(secDef *parser.SecurityDefinition, moduleName string) error {
	groupID, err := l.syncGroup(secDef, moduleName)
	if err != nil {
		return fmt.Errorf("sync group %s: %w", secDef.Name, err)
	}

	if err := l.syncImpliedGroups(groupID, secDef.Implies); err != nil {
		return fmt.Errorf("sync implies for %s: %w", secDef.Name, err)
	}

	if err := l.syncModelAccess(groupID, secDef.Access, moduleName); err != nil {
		return fmt.Errorf("sync access for %s: %w", secDef.Name, err)
	}

	if err := l.syncRecordRules(groupID, secDef.Rules, moduleName); err != nil {
		return fmt.Errorf("sync rules for %s: %w", secDef.Name, err)
	}

	if err := l.syncGroupMenus(groupID, secDef.Menus, moduleName); err != nil {
		return fmt.Errorf("sync menus for %s: %w", secDef.Name, err)
	}

	if err := l.syncGroupPages(groupID, secDef.Pages, moduleName); err != nil {
		return fmt.Errorf("sync pages for %s: %w", secDef.Name, err)
	}

	return nil
}

func (l *SecurityLoader) syncGroup(secDef *parser.SecurityDefinition, moduleName string) (string, error) {
	var existingID string
	err := l.db.Table(l.tn("group")).Select("id").Where("name = ?", secDef.Name).Pluck("id", &existingID).Error

	if err != nil || existingID == "" {
		groupID := uuid.New().String()
		now := time.Now()
		return groupID, l.db.Table(l.tn("group")).Create(map[string]any{
			"id":              groupID,
			"name":            secDef.Name,
			"display_name":    secDef.Label,
			"category":        secDef.Category,
			"share":           secDef.Share,
			"comment":         secDef.Comment,
			"module":          moduleName,
			"modified_source": "json",
			"created_at":      now,
			"updated_at":      now,
		}).Error
	}

	var modifiedSource string
	l.db.Table(l.tn("group")).Select("modified_source").Where("id = ?", existingID).Pluck("modified_source", &modifiedSource)

	if modifiedSource == "ui" {
		log.Printf("[SECURITY] skipping group %s (modified via UI)", secDef.Name)
		return existingID, nil
	}

	return existingID, l.db.Table(l.tn("group")).Where("id = ?", existingID).Updates(map[string]any{
		"display_name":    secDef.Label,
		"category":        secDef.Category,
		"share":           secDef.Share,
		"comment":         secDef.Comment,
		"module":          moduleName,
		"modified_source": "json",
		"updated_at":      time.Now(),
	}).Error
}

func (l *SecurityLoader) syncImpliedGroups(groupID string, implies []string) error {
	l.db.Table(l.tn("group") + "_implies").Where("group_id = ?", groupID).Delete(nil)

	for _, impliedName := range implies {
		var impliedID string
		l.db.Table(l.tn("group")).Select("id").Where("name = ?", impliedName).Pluck("id", &impliedID)
		if impliedID == "" {
			log.Printf("[WARN] implied group %q not found, skipping", impliedName)
			continue
		}
		l.db.Table(l.tn("group") + "_implies").Create(map[string]any{
			"group_id":         groupID,
			"implied_group_id": impliedID,
		})
	}
	return nil
}

func (l *SecurityLoader) syncModelAccess(groupID string, access map[string]parser.SecurityACL, moduleName string) error {
	l.db.Table(l.tn("model_access")).Where("group_id = ? AND module = ? AND modified_source = ?", groupID, moduleName, "json").Delete(nil)

	allPerms := []string{"select", "read", "write", "create", "delete", "print", "email", "report", "export", "import", "mask", "clone"}

	for modelName, perms := range access {
		permSet := make(map[string]bool)
		for _, p := range perms {
			permSet[p] = true
		}

		row := map[string]any{
			"id":              uuid.New().String(),
			"name":            fmt.Sprintf("%s %s access", modelName, groupID[:8]),
			"model_name":      modelName,
			"group_id":        groupID,
			"module":          moduleName,
			"modified_source": "json",
			"created_at":      time.Now(),
			"updated_at":      time.Now(),
		}
		for _, p := range allPerms {
			row["can_"+p] = permSet[p]
		}

		if err := l.db.Table(l.tn("model_access")).Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (l *SecurityLoader) syncRecordRules(groupID string, rules []parser.SecurityRuleDefinition, moduleName string) error {
	for _, rule := range rules {
		domainJSON := "[]"
		if len(rule.Domain) > 0 {
			if b, err := json.Marshal(rule.Domain); err == nil {
				domainJSON = string(b)
			}
		}

		var existingID string
		l.db.Table(l.tn("record_rule")).Select("id").Where("name = ?", rule.Name).Pluck("id", &existingID)

		ruleID := existingID
		if ruleID == "" {
			ruleID = uuid.New().String()
			now := time.Now()
			l.db.Table(l.tn("record_rule")).Create(map[string]any{
				"id":              ruleID,
				"name":            rule.Name,
				"model_name":      rule.Model,
				"domain_filter":   domainJSON,
				"can_read":        rule.IsPermRead(),
				"can_create":      rule.IsPermCreate(),
				"can_write":       rule.IsPermWrite(),
				"can_delete":      rule.IsPermDelete(),
				"is_global":       rule.Global,
				"active":          true,
				"module":          moduleName,
				"modified_source": "json",
				"created_at":      now,
				"updated_at":      now,
			})
		} else {
			var modSrc string
			l.db.Table(l.tn("record_rule")).Select("modified_source").Where("id = ?", ruleID).Pluck("modified_source", &modSrc)
			if modSrc != "ui" {
				l.db.Table(l.tn("record_rule")).Where("id = ?", ruleID).Updates(map[string]any{
					"model_name":      rule.Model,
					"domain_filter":   domainJSON,
					"can_read":        rule.IsPermRead(),
					"can_create":      rule.IsPermCreate(),
					"can_write":       rule.IsPermWrite(),
					"can_delete":      rule.IsPermDelete(),
					"is_global":       rule.Global,
					"active":          true,
					"module":          moduleName,
					"modified_source": "json",
					"updated_at":      time.Now(),
				})
			}
		}

		l.db.Table(l.tn("record_rule") + "_groups").Where("record_rule_id = ?", ruleID).Delete(nil)
		l.db.Table(l.tn("record_rule") + "_groups").Create(map[string]any{
			"record_rule_id": ruleID,
			"group_id":       groupID,
		})
	}
	return nil
}

func (l *SecurityLoader) syncGroupMenus(groupID string, menus []string, moduleName string) error {
	l.db.Table(l.tn("group") + "_menus").Where("group_id = ? AND module = ?", groupID, moduleName).Delete(nil)
	for _, menu := range menus {
		l.db.Table(l.tn("group") + "_menus").Create(map[string]any{
			"group_id":     groupID,
			"menu_item_id": menu,
			"module":       moduleName,
		})
	}
	return nil
}

func (l *SecurityLoader) syncGroupPages(groupID string, pages []string, moduleName string) error {
	l.db.Table(l.tn("group") + "_pages").Where("group_id = ? AND module = ?", groupID, moduleName).Delete(nil)
	for _, page := range pages {
		l.db.Table(l.tn("group") + "_pages").Create(map[string]any{
			"group_id":  groupID,
			"page_name": page,
			"module":    moduleName,
		})
	}
	return nil
}
