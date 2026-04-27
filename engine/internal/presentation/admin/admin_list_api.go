package admin

import (
	"fmt"
	"sort"

	"github.com/gofiber/fiber/v2"
)

func (a *AdminPanel) apiListModelsData(c *fiber.Ctx) error {
	models := a.modelRegistry.List()
	sort.Slice(models, func(i, j int) bool {
		if models[i].Module != models[j].Module {
			return models[i].Module < models[j].Module
		}
		return models[i].Name < models[j].Name
	})

	var data []map[string]any
	for _, m := range models {
		moduleName := m.Module
		if moduleName == "" {
			moduleName = "base"
		}
		label := m.Label
		if label == "" {
			label = m.Name
		}
		inherit := ""
		if m.Inherit != "" {
			inherit = m.Inherit
		}
		data = append(data, map[string]any{
			"id":      moduleName + "/" + m.Name,
			"name":    m.Name,
			"module":  moduleName,
			"label":   label,
			"fields":  len(m.Fields),
			"inherit": inherit,
		})
	}

	return c.JSON(fiber.Map{"data": data, "total": len(data), "page": 1, "page_size": len(data), "total_pages": 1})
}

func (a *AdminPanel) apiListModulesData(c *fiber.Ctx) error {
	modules := a.moduleRegistry.List()

	var data []map[string]any
	for _, m := range modules {
		label := m.Definition.Label
		if label == "" {
			label = m.Definition.Name
		}
		deps := ""
		if len(m.Definition.Depends) > 0 {
			for i, d := range m.Definition.Depends {
				if i > 0 {
					deps += ", "
				}
				deps += d
			}
		}
		data = append(data, map[string]any{
			"id":           m.Definition.Name,
			"name":         m.Definition.Name,
			"version":      m.Definition.Version,
			"label":        label,
			"category":     m.Definition.Category,
			"dependencies": deps,
			"status":       m.State,
		})
	}

	return c.JSON(fiber.Map{"data": data, "total": len(data), "page": 1, "page_size": len(data), "total_pages": 1})
}

func (a *AdminPanel) apiListViewsData(c *fiber.Ctx) error {
	views := a.views()
	keys := make([]string, 0, len(views))
	for k := range views {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var data []map[string]any
	for _, key := range keys {
		v := views[key]
		editable := "embedded"
		if v.Editable {
			editable = "editable"
		}
		data = append(data, map[string]any{
			"id":     key,
			"name":   v.Def.Name,
			"type":   string(v.Def.Type),
			"model":  v.Def.Model,
			"title":  v.Def.Title,
			"module": v.Module,
			"status": editable,
		})
	}

	return c.JSON(fiber.Map{"data": data, "total": len(data), "page": 1, "page_size": len(data), "total_pages": 1})
}

func (a *AdminPanel) apiListGroupsData(c *fiber.Ctx) error {
	gt := a.modelRegistry.TableName("group")
	ut := a.modelRegistry.TableName("user")
	ugt := ut + "_" + gt

	var groups []map[string]any
	a.db.Table(gt).
		Select(fmt.Sprintf("%s.id, %s.name, %s.display_name, %s.category, %s.share, %s.module, %s.modified_source, (SELECT COUNT(*) FROM %s WHERE %s.group_id = %s.id) as user_count",
			gt, gt, gt, gt, gt, gt, gt, ugt, ugt, gt)).
		Order("category, name").
		Find(&groups)

	var data []map[string]any
	for _, g := range groups {
		share := false
		if g["share"] == true || g["share"] == int64(1) {
			share = true
		}
		data = append(data, map[string]any{
			"id":         g["id"],
			"name":       g["name"],
			"label":      g["display_name"],
			"category":   g["category"],
			"share":      share,
			"module":     g["module"],
			"users":      g["user_count"],
			"source":     g["modified_source"],
		})
	}

	return c.JSON(fiber.Map{"data": data, "total": len(data), "page": 1, "page_size": len(data), "total_pages": 1})
}
