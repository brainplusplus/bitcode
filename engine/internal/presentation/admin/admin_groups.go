package admin

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (a *AdminPanel) listGroups(c *fiber.Ctx) error {
	var groups []map[string]any
	a.db.Table("groups").
		Select("groups.id, groups.name, groups.display_name, groups.category, groups.share, groups.module, groups.modified_source, (SELECT COUNT(*) FROM user_groups WHERE user_groups.group_id = groups.id) as user_count").
		Order("category, name").
		Find(&groups)

	var html strings.Builder
	html.WriteString(a.pageHeader("Groups", "groups"))

	html.WriteString(fmt.Sprintf(`<div class="list-toolbar"><div class="list-count text-muted">%d groups</div></div>`, len(groups)))

	html.WriteString(`<div class="card"><table><thead><tr><th>Name</th><th>Label</th><th>Category</th><th>Share</th><th>Module</th><th>Users</th><th>Source</th></tr></thead><tbody>`)
	for _, g := range groups {
		name := fmt.Sprintf("%v", g["name"])
		displayName := fmt.Sprintf("%v", g["display_name"])
		category := fmt.Sprintf("%v", g["category"])
		module := fmt.Sprintf("%v", g["module"])
		source := fmt.Sprintf("%v", g["modified_source"])
		userCount := fmt.Sprintf("%v", g["user_count"])
		share := g["share"]
		shareBadge := `<span class="text-muted">No</span>`
		if share == true || share == int64(1) || share == "1" {
			shareBadge = `<span class="badge blue">Yes</span>`
		}
		sourceBadge := fmt.Sprintf(`<span class="badge muted">%s</span>`, source)
		if source == "ui" {
			sourceBadge = `<span class="badge green">ui</span>`
		}

		html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/groups/%s" class="fw-500">%s</a></td><td>%s</td><td><span class="badge muted">%s</span></td><td>%s</td><td><span class="badge muted">%s</span></td><td>%s</td><td>%s</td></tr>`,
			name, name, displayName, category, shareBadge, module, userCount, sourceBadge))
	}
	if len(groups) == 0 {
		html.WriteString(`<tr><td colspan="7" class="empty-state">No groups defined. Load modules with securities/*.json.</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) viewGroup(c *fiber.Ctx) error {
	groupName := c.Params("name")
	tab := c.Query("tab", "users")

	var group map[string]any
	if err := a.db.Table("groups").Where("name = ?", groupName).First(&group).Error; err != nil {
		return c.Status(404).SendString("Group not found")
	}

	groupID := fmt.Sprintf("%v", group["id"])
	displayName := fmt.Sprintf("%v", group["display_name"])
	category := fmt.Sprintf("%v", group["category"])

	breadcrumb := fmt.Sprintf(`<div class="breadcrumb"><a href="/admin">Admin</a> <span class="sep">/</span> <a href="/admin/groups">Groups</a> <span class="sep">/</span> <span class="fw-500">%s</span></div>`, displayName)

	tabs := fmt.Sprintf(`<div class="tabs"><a href="/admin/groups/%s?tab=users" class="tab%s">Users</a><a href="/admin/groups/%s?tab=inherited" class="tab%s">Inherited</a><a href="/admin/groups/%s?tab=menus" class="tab%s">Menus</a><a href="/admin/groups/%s?tab=pages" class="tab%s">Pages</a><a href="/admin/groups/%s?tab=access" class="tab%s">Access Rights</a><a href="/admin/groups/%s?tab=rules" class="tab%s">Record Rules</a><a href="/admin/groups/%s?tab=notes" class="tab%s">Notes</a></div>`,
		groupName, activeClass(tab, "users"),
		groupName, activeClass(tab, "inherited"),
		groupName, activeClass(tab, "menus"),
		groupName, activeClass(tab, "pages"),
		groupName, activeClass(tab, "access"),
		groupName, activeClass(tab, "rules"),
		groupName, activeClass(tab, "notes"))

	share := group["share"]
	shareCheck := ""
	if share == true || share == int64(1) {
		shareCheck = " checked"
	}

	meta := fmt.Sprintf(`<div class="model-header"><div class="model-name">%s</div><div class="model-meta"><span class="badge muted">%s</span> <code>%s</code></div></div>`, displayName, category, groupName)

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s - BitCode Admin</title>
%s</head><body>
<div class="layout">%s<div class="main"><div class="topbar">%s</div><div class="content">%s%s`,
		displayName, cssBlock(), a.sidebarHTML("groups"), breadcrumb, meta, tabs))

	switch tab {
	case "inherited":
		a.renderGroupInherited(&html, groupID)
	case "menus":
		a.renderGroupMenus(&html, groupID)
	case "pages":
		a.renderGroupPages(&html, groupID)
	case "access":
		a.renderGroupAccess(&html, groupID)
	case "rules":
		a.renderGroupRules(&html, groupID)
	case "notes":
		comment := fmt.Sprintf("%v", group["comment"])
		if comment == "<nil>" {
			comment = ""
		}
		html.WriteString(fmt.Sprintf(`<div class="card"><div class="card-title">Notes</div><div style="padding:16px;white-space:pre-wrap;font-size:13px">%s</div></div>`, comment))
	default:
		a.renderGroupUsers(&html, groupID)
	}

	_ = shareCheck
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) renderGroupUsers(html *strings.Builder, groupID string) {
	var users []map[string]any
	a.db.Table("users").
		Select("users.id, users.username, users.email, users.active").
		Joins("INNER JOIN user_groups ON user_groups.user_id = users.id").
		Where("user_groups.group_id = ?", groupID).
		Find(&users)

	html.WriteString(`<div class="card"><div class="card-title">Users</div>`)
	html.WriteString(`<table><thead><tr><th>Username</th><th>Email</th><th>Active</th></tr></thead><tbody>`)
	for _, u := range users {
		active := `<span class="text-muted">✖</span>`
		if u["active"] == true || u["active"] == int64(1) {
			active = `<span class="dot green-dot"></span>`
		}
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">%v</td><td>%v</td><td>%s</td></tr>`, u["username"], u["email"], active))
	}
	if len(users) == 0 {
		html.WriteString(`<tr><td colspan="3" class="empty-state">No users in this group</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
}

func (a *AdminPanel) renderGroupInherited(html *strings.Builder, groupID string) {
	var implied []map[string]any
	a.db.Table("groups").
		Select("groups.name, groups.display_name").
		Joins("INNER JOIN group_implies ON group_implies.implied_group_id = groups.id").
		Where("group_implies.group_id = ?", groupID).
		Find(&implied)

	html.WriteString(`<div class="card"><div class="card-title">Inherited Groups</div><p style="padding:8px 16px;font-size:12px;color:var(--text-muted)">Users added to this group are automatically added to the following groups.</p>`)
	html.WriteString(`<table><thead><tr><th>Group Name</th></tr></thead><tbody>`)
	for _, g := range implied {
		html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/groups/%v">%v</a></td></tr>`, g["name"], g["display_name"]))
	}
	if len(implied) == 0 {
		html.WriteString(`<tr><td class="empty-state">No inherited groups</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
}

func (a *AdminPanel) renderGroupMenus(html *strings.Builder, groupID string) {
	var menus []map[string]any
	a.db.Table("group_menus").Where("group_id = ?", groupID).Find(&menus)

	html.WriteString(`<div class="card"><div class="card-title">Menus</div>`)
	html.WriteString(`<table><thead><tr><th>Menu Item</th><th>Module</th></tr></thead><tbody>`)
	for _, m := range menus {
		html.WriteString(fmt.Sprintf(`<tr><td>%v</td><td><span class="badge muted">%v</span></td></tr>`, m["menu_item_id"], m["module"]))
	}
	if len(menus) == 0 {
		html.WriteString(`<tr><td colspan="2" class="empty-state">No menu items assigned</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
}

func (a *AdminPanel) renderGroupPages(html *strings.Builder, groupID string) {
	var pages []map[string]any
	a.db.Table("group_pages").Where("group_id = ?", groupID).Find(&pages)

	html.WriteString(`<div class="card"><div class="card-title">Pages</div>`)
	html.WriteString(`<table><thead><tr><th>Page Name</th><th>Module</th></tr></thead><tbody>`)
	for _, p := range pages {
		html.WriteString(fmt.Sprintf(`<tr><td>%v</td><td><span class="badge muted">%v</span></td></tr>`, p["page_name"], p["module"]))
	}
	if len(pages) == 0 {
		html.WriteString(`<tr><td colspan="2" class="empty-state">No pages assigned</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
}

func (a *AdminPanel) renderGroupAccess(html *strings.Builder, groupID string) {
	var acls []map[string]any
	a.db.Table("model_accesses").Where("group_id = ?", groupID).Order("model_name").Find(&acls)

	check := func(v any) string {
		if v == true || v == int64(1) || v == "1" {
			return "✔"
		}
		return `<span class="text-muted">✖</span>`
	}

	html.WriteString(`<div class="card"><div class="card-title">Access Rights</div>`)
	html.WriteString(`<table><thead><tr><th>Name</th><th>Model</th><th>Se</th><th>Re</th><th>Wr</th><th>Cr</th><th>De</th><th>Pr</th><th>Em</th><th>Rp</th><th>Ex</th><th>Im</th><th>Mk</th><th>Cl</th></tr></thead><tbody>`)
	for _, acl := range acls {
		html.WriteString(fmt.Sprintf(`<tr><td>%v</td><td><code>%v</code></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			acl["name"], acl["model_name"],
			check(acl["can_select"]), check(acl["can_read"]), check(acl["can_write"]), check(acl["can_create"]), check(acl["can_delete"]),
			check(acl["can_print"]), check(acl["can_email"]), check(acl["can_report"]),
			check(acl["can_export"]), check(acl["can_import"]), check(acl["can_mask"]), check(acl["can_clone"])))
	}
	if len(acls) == 0 {
		html.WriteString(`<tr><td colspan="14" class="empty-state">No access rights defined</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
	html.WriteString(`<div style="font-size:11px;color:var(--text-muted);margin-top:4px">Se=Select Re=Read Wr=Write Cr=Create De=Delete Pr=Print Em=Email Rp=Report Ex=Export Im=Import Mk=Mask Cl=Clone</div>`)
}

func (a *AdminPanel) renderGroupRules(html *strings.Builder, groupID string) {
	var rules []map[string]any
	a.db.Table("record_rules").
		Select("record_rules.name, record_rules.model_name, record_rules.domain_filter, record_rules.can_read, record_rules.can_write, record_rules.can_create, record_rules.can_delete").
		Joins("INNER JOIN record_rule_groups ON record_rule_groups.record_rule_id = record_rules.id").
		Where("record_rule_groups.group_id = ?", groupID).
		Find(&rules)

	check := func(v any) string {
		if v == true || v == int64(1) || v == "1" {
			return "✔"
		}
		return `<span class="text-muted">✖</span>`
	}

	html.WriteString(`<div class="card"><div class="card-title">Record Rules</div>`)
	html.WriteString(`<table><thead><tr><th>Name</th><th>Model</th><th>Domain</th><th>R</th><th>W</th><th>C</th><th>D</th></tr></thead><tbody>`)
	for _, r := range rules {
		domain := fmt.Sprintf("%v", r["domain_filter"])
		if len(domain) > 60 {
			domain = domain[:60] + "..."
		}
		html.WriteString(fmt.Sprintf(`<tr><td>%v</td><td><code>%v</code></td><td><code style="font-size:11px">%s</code></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			r["name"], r["model_name"], domain,
			check(r["can_read"]), check(r["can_write"]), check(r["can_create"]), check(r["can_delete"])))
	}
	if len(rules) == 0 {
		html.WriteString(`<tr><td colspan="7" class="empty-state">No record rules defined</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
}

func (a *AdminPanel) apiListGroups(c *fiber.Ctx) error {
	var groups []map[string]any
	a.db.Table("groups").Order("category, name").Find(&groups)
	return c.JSON(fiber.Map{"data": groups})
}

func (a *AdminPanel) apiGetGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	var group map[string]any
	if err := a.db.Table("groups").Where("id = ?", id).First(&group).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "group not found"})
	}
	return c.JSON(fiber.Map{"data": group})
}

func (a *AdminPanel) apiCreateGroup(c *fiber.Ctx) error {
	var body map[string]any
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["modified_source"] = "ui"
	if err := a.db.Table("groups").Create(body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"ok": true, "data": body})
}

func (a *AdminPanel) apiUpdateGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	var body map[string]any
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	body["modified_source"] = "ui"
	if err := a.db.Table("groups").Where("id = ?", id).Updates(body).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (a *AdminPanel) apiDeleteGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	a.db.Table("group_implies").Where("group_id = ? OR implied_group_id = ?", id, id).Delete(nil)
	a.db.Table("user_groups").Where("group_id = ?", id).Delete(nil)
	a.db.Table("model_accesses").Where("group_id = ?", id).Delete(nil)
	a.db.Table("record_rule_groups").Where("group_id = ?", id).Delete(nil)
	a.db.Table("group_menus").Where("group_id = ?", id).Delete(nil)
	a.db.Table("group_pages").Where("group_id = ?", id).Delete(nil)
	if err := a.db.Table("groups").Where("id = ?", id).Delete(nil).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}
