package admin

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"

	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
)

func (a *AdminPanel) healthPage(c *fiber.Ctx) error {
	models := a.modelRegistry.List()
	modules := a.moduleRegistry.List()

	var html strings.Builder
	html.WriteString(a.pageHeader("Health", "health"))

	statusColor := "var(--green)"
	statusText := "Healthy"

	html.WriteString(fmt.Sprintf(`<div class="card"><div class="card-title">System Status</div><div style="padding:20px 16px;display:flex;align-items:center;gap:12px"><div class="health-dot" style="background:%s"></div><span style="font-size:16px;font-weight:600">%s</span><span class="text-muted">v%s</span></div></div>`, statusColor, statusText, a.health.Version))

	html.WriteString(`<div class="stats-grid">`)
	html.WriteString(statCard("Models", fmt.Sprintf("%d", len(models)), "var(--blue)"))
	html.WriteString(statCard("Modules", fmt.Sprintf("%d", len(modules)), "var(--green)"))
	html.WriteString(statCard("Views", fmt.Sprintf("%d", len(a.views())), "var(--amber)"))
	html.WriteString(statCard("Processes", fmt.Sprintf("%d", len(a.health.Processes)), "var(--primary)"))
	html.WriteString(`</div>`)

	html.WriteString(`<div class="conn-grid">`)

	html.WriteString(`<div class="card"><div class="card-title">Infrastructure</div><table>`)
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Engine</td><td>bitcode</td></tr>`))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Version</td><td>%s</td></tr>`, a.health.Version))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Database</td><td><span class="badge muted">%s</span></td></tr>`, a.health.DBDriver))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Cache</td><td><span class="badge muted">%s</span></td></tr>`, a.health.CacheDriver))
	html.WriteString(`</table></div>`)

	html.WriteString(`<div class="card"><div class="card-title">Installed Modules</div><div class="conn-list">`)
	for _, m := range modules {
		html.WriteString(fmt.Sprintf(`<div class="conn-item"><div class="conn-model">%s</div><div class="conn-detail">v%s <span class="badge green">%s</span></div></div>`,
			m.Definition.Name, m.Definition.Version, m.State))
	}
	html.WriteString(`</div></div>`)

	html.WriteString(`</div>`)

	if len(a.health.Processes) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Registered Processes</div><div style="padding:12px 16px;display:flex;flex-wrap:wrap;gap:6px">`)
		sort.Strings(a.health.Processes)
		for _, p := range a.health.Processes {
			html.WriteString(fmt.Sprintf(`<span class="badge muted">%s</span>`, p))
		}
		html.WriteString(`</div></div>`)
	}

	html.WriteString(fmt.Sprintf(`<div style="margin-top:8px"><a href="/health" target="_blank" class="btn-sm">View JSON API &rarr;</a></div>`))
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) loginHistoryPage(c *fiber.Ctx) error {
	var html strings.Builder
	html.WriteString(a.pageHeader("Login History", "login-history"))

	results, _ := a.auditLogRepo.FindLoginHistory(200)

	html.WriteString(`<div class="card"><div class="card-title">Login / Logout / Register Activity</div>`)
	html.WriteString(`<table class="list-table"><thead><tr><th>Time</th><th>Action</th><th>User</th><th>IP Address</th><th>User Agent</th></tr></thead><tbody>`)

	for _, r := range results {
		action, _ := r["action"].(string)
		badgeClass := "blue"
		if action == "logout" {
			badgeClass = "muted"
		} else if action == "register" {
			badgeClass = "green"
		}
		userID := fmt.Sprintf("%v", r["user_id"])
		ip := fmt.Sprintf("%v", r["ip_address"])
		ua := fmt.Sprintf("%v", r["user_agent"])
		if len(ua) > 80 {
			ua = ua[:80] + "..."
		}
		createdAt := fmt.Sprintf("%v", r["created_at"])

		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted" style="white-space:nowrap">%s</td><td><span class="badge %s">%s</span></td><td>%s</td><td><code>%s</code></td><td style="font-size:11px;color:var(--text-muted);max-width:300px;overflow:hidden;text-overflow:ellipsis">%s</td></tr>`,
			createdAt, badgeClass, action, userID, ip, template.HTMLEscapeString(ua)))
	}

	html.WriteString(`</tbody></table></div>`)

	return c.Type("html").SendString(html.String())
}

func (a *AdminPanel) requestLogPage(c *fiber.Ctx) error {
	methodFilter := c.Query("method", "")

	var html strings.Builder
	html.WriteString(a.pageHeader("API Request Log", "request-log"))

	html.WriteString(`<div style="margin-bottom:12px;display:flex;gap:4px">`)
	methods := []string{"", "GET", "POST", "PUT", "DELETE", "PATCH"}
	labels := []string{"All", "GET", "POST", "PUT", "DELETE", "PATCH"}
	for i, m := range methods {
		active := ""
		if m == methodFilter {
			active = " active"
		}
		href := "/admin/audit/request-log"
		if m != "" {
			href += "?method=" + m
		}
		html.WriteString(fmt.Sprintf(`<a href="%s" class="filter-pill%s">%s</a>`, href, active, labels[i]))
	}
	html.WriteString(`</div>`)

	results, _ := a.auditLogRepo.FindRequests(200, methodFilter)

	html.WriteString(`<div class="card"><div class="card-title">Request Log</div>`)
	html.WriteString(`<table class="list-table"><thead><tr><th>Time</th><th>Method</th><th>Path</th><th>Status</th><th>Duration</th><th>User</th><th>IP</th></tr></thead><tbody>`)

	for _, r := range results {
		method := fmt.Sprintf("%v", r["request_method"])
		path := fmt.Sprintf("%v", r["request_path"])
		status := fmt.Sprintf("%v", r["status_code"])
		duration := fmt.Sprintf("%v", r["duration_ms"])
		userID := fmt.Sprintf("%v", r["user_id"])
		ip := fmt.Sprintf("%v", r["ip_address"])
		createdAt := fmt.Sprintf("%v", r["created_at"])

		methodBadge := "blue"
		if method == "POST" {
			methodBadge = "green"
		} else if method == "PUT" || method == "PATCH" {
			methodBadge = "yellow"
		} else if method == "DELETE" {
			methodBadge = "red"
		}

		statusColor := "var(--green)"
		if status >= "400" {
			statusColor = "var(--red)"
		} else if status >= "300" {
			statusColor = "var(--yellow)"
		}

		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted" style="white-space:nowrap;font-size:11px">%s</td><td><span class="badge %s">%s</span></td><td style="font-family:monospace;font-size:12px">%s</td><td style="color:%s;font-weight:500">%s</td><td>%sms</td><td>%s</td><td><code style="font-size:11px">%s</code></td></tr>`,
			createdAt, methodBadge, method, template.HTMLEscapeString(path), statusColor, status, duration, userID, ip))
	}

	html.WriteString(`</tbody></table></div>`)

	return c.Type("html").SendString(html.String())
}

func (a *AdminPanel) apiImpersonate(c *fiber.Ctx) error {
	adminToken := c.Get("Authorization")
	if adminToken == "" {
		adminToken = c.Cookies("token")
	} else if len(adminToken) > 7 && adminToken[:7] == "Bearer " {
		adminToken = adminToken[7:]
	}
	if adminToken == "" {
		return c.Status(401).JSON(fiber.Map{"error": "authentication required"})
	}

	adminClaims, err := security.ValidateToken(a.jwtConfig, adminToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}

	if !containsRole(adminClaims.Roles, "admin") {
		return c.Status(403).JSON(fiber.Map{"error": "only admin users can impersonate"})
	}

	if adminClaims.ImpersonatedBy != "" {
		return c.Status(400).JSON(fiber.Map{"error": "cannot impersonate while already impersonating"})
	}

	targetUserID := c.Params("user_id")
	if targetUserID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "user_id required"})
	}

	repo := persistence.NewGenericRepository(a.db, a.modelRegistry.TableName("user"))
	targetUser, err := repo.FindByID(c.Context(), targetUserID)
	if err != nil || targetUser == nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	targetUsername, _ := targetUser["username"].(string)

	targetRoles := loadUserRoles(a.db, targetUserID, a.modelRegistry)
	if containsRole(targetRoles, "admin") {
		return c.Status(403).JSON(fiber.Map{"error": "cannot impersonate another admin user"})
	}

	targetGroups := loadUserGroups(a.db, targetUserID, a.modelRegistry)

	token, err := security.GenerateToken(
		a.jwtConfig,
		targetUserID,
		targetUsername,
		targetRoles,
		targetGroups,
		security.WithImpersonatedBy(adminClaims.UserID),
		security.WithExpiration(1*time.Hour),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate impersonation token"})
	}

	a.auditLogRepo.WriteAsync(persistence.AuditLogEntry{
		UserID:         adminClaims.UserID,
		Action:         "impersonate_start",
		ModelName:      "user",
		RecordID:       targetUserID,
		ImpersonatedBy: adminClaims.UserID,
		IPAddress:      c.IP(),
		UserAgent:      c.Get("User-Agent"),
		RequestMethod:  c.Method(),
		RequestPath:    c.Path(),
	})

	return c.JSON(fiber.Map{
		"token": token,
		"impersonating": fiber.Map{
			"user_id":  targetUserID,
			"username": targetUsername,
		},
		"admin": fiber.Map{
			"user_id":  adminClaims.UserID,
			"username": adminClaims.Username,
		},
		"expires_in": 3600,
	})
}

func (a *AdminPanel) apiStopImpersonate(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")
	if tokenStr == "" {
		tokenStr = c.Cookies("token")
	} else if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}
	if tokenStr == "" {
		return c.Status(401).JSON(fiber.Map{"error": "authentication required"})
	}

	claims, err := security.ValidateToken(a.jwtConfig, tokenStr)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}

	if claims.ImpersonatedBy == "" {
		return c.Status(400).JSON(fiber.Map{"error": "not currently impersonating"})
	}

	adminID := claims.ImpersonatedBy
	repo := persistence.NewGenericRepository(a.db, a.modelRegistry.TableName("user"))
	adminUser, err := repo.FindByID(c.Context(), adminID)
	if err != nil || adminUser == nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load admin user"})
	}

	adminUsername, _ := adminUser["username"].(string)
	adminRoles := loadUserRoles(a.db, adminID, a.modelRegistry)
	adminGroups := loadUserGroups(a.db, adminID, a.modelRegistry)

	newToken, err := security.GenerateToken(
		a.jwtConfig,
		adminID,
		adminUsername,
		adminRoles,
		adminGroups,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate admin token"})
	}

	a.auditLogRepo.WriteAsync(persistence.AuditLogEntry{
		UserID:         adminID,
		Action:         "impersonate_stop",
		ModelName:      "user",
		RecordID:       claims.UserID,
		ImpersonatedBy: adminID,
		IPAddress:      c.IP(),
		UserAgent:      c.Get("User-Agent"),
		RequestMethod:  c.Method(),
		RequestPath:    c.Path(),
	})

	return c.JSON(fiber.Map{
		"token":    newToken,
		"user_id":  adminID,
		"username": adminUsername,
	})
}

func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func loadUserRoles(db *gorm.DB, userID string, reg *domainModel.Registry) []string {
	roleTable := reg.TableName("role")
	userRoleTable := reg.TableName("user") + "_roles"
	var roles []string
	db.Raw(fmt.Sprintf(`SELECT r.name FROM %s r
		INNER JOIN %s ur ON ur.role_id = r.id
		WHERE ur.user_id = ?`, roleTable, userRoleTable), userID).Scan(&roles)
	return roles
}

func loadUserGroups(db *gorm.DB, userID string, reg *domainModel.Registry) []string {
	groupTable := reg.TableName("group")
	userGroupTable := reg.TableName("user") + "_groups"
	var groups []string
	db.Raw(fmt.Sprintf(`SELECT g.name FROM %s g
		INNER JOIN %s ug ON ug.group_id = g.id
		WHERE ug.user_id = ?`, groupTable, userGroupTable), userID).Scan(&groups)
	return groups
}
