package security

import (
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
)

type Group struct {
	ddd.BaseEntity
	Name          string  `json:"name" gorm:"uniqueIndex;size:100"`
	DisplayName   string  `json:"display_name" gorm:"size:200"`
	Category      string  `json:"category" gorm:"size:100"`
	ImpliedGroups []Group `json:"implied_groups" gorm:"many2many:group_implies;"`
}

func NewGroup(id string, name string, displayName string, category string) *Group {
	return &Group{
		BaseEntity:  ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:        name,
		DisplayName: displayName,
		Category:    category,
	}
}

func (g *Group) AllGroupNames() []string {
	seen := make(map[string]bool)
	g.collectGroups(seen)
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result
}

func (g *Group) collectGroups(seen map[string]bool) {
	if seen[g.Name] {
		return
	}
	seen[g.Name] = true
	for _, implied := range g.ImpliedGroups {
		implied.collectGroups(seen)
	}
}
