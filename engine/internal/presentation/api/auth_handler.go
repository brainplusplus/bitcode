package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"github.com/bitcode-engine/engine/pkg/security"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db        *gorm.DB
	jwtCfg    security.JWTConfig
	auditRepo *persistence.AuditLogRepository
}

func NewAuthHandler(db *gorm.DB, jwtCfg security.JWTConfig) *AuthHandler {
	return &AuthHandler{db: db, jwtCfg: jwtCfg}
}

func NewAuthHandlerWithAudit(db *gorm.DB, jwtCfg security.JWTConfig, auditRepo *persistence.AuditLogRepository) *AuthHandler {
	return &AuthHandler{db: db, jwtCfg: jwtCfg, auditRepo: auditRepo}
}

func (h *AuthHandler) Register(app *fiber.App) {
	auth := app.Group("/auth")
	auth.Post("/login", h.Login)
	auth.Post("/register", h.RegisterUser)
	auth.Post("/logout", h.Logout)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if body.Username == "" || body.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username and password required"})
	}

	repo := persistence.NewGenericRepository(h.db, "users")
	users, _, err := repo.FindAll(c.Context(), [][]any{{"username", "=", body.Username}}, 1, 1)
	if err != nil || len(users) == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	user := users[0]
	hash, _ := user["password_hash"].(string)
	if !security.CheckPassword(body.Password, hash) {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	userID, _ := user["id"].(string)
	username, _ := user["username"].(string)

	token, err := security.GenerateToken(h.jwtCfg, userID, username, nil, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
	}

	repo.Update(c.Context(), userID, map[string]any{"last_login": time.Now()})

	h.writeAuditLog(c, userID, "login", "user", userID)

	return c.JSON(fiber.Map{
		"token":    token,
		"user_id":  userID,
		"username": username,
	})
}

func (h *AuthHandler) RegisterUser(c *fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if body.Username == "" || body.Email == "" || body.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username, email, and password required"})
	}

	hash, err := security.HashPassword(body.Password)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	repo := persistence.NewGenericRepository(h.db, "users")

	existing, _, _ := repo.FindAll(c.Context(), [][]any{{"username", "=", body.Username}}, 1, 1)
	if len(existing) > 0 {
		return c.Status(409).JSON(fiber.Map{"error": "username already exists"})
	}

	existingEmail, _, _ := repo.FindAll(c.Context(), [][]any{{"email", "=", body.Email}}, 1, 1)
	if len(existingEmail) > 0 {
		return c.Status(409).JSON(fiber.Map{"error": "email already exists"})
	}

	record := map[string]any{
		"id":            uuid.New().String(),
		"username":      body.Username,
		"email":         body.Email,
		"password_hash": hash,
		"active":        true,
	}

	result, err := repo.Create(c.Context(), record)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	token, _ := security.GenerateToken(h.jwtCfg, record["id"].(string), body.Username, nil, nil)

	h.writeAuditLog(c, record["id"].(string), "register", "user", record["id"].(string))

	return c.Status(201).JSON(fiber.Map{
		"user_id":  result["id"],
		"username": body.Username,
		"email":    body.Email,
		"token":    token,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	h.writeAuditLog(c, userID, "logout", "user", userID)

	return c.JSON(fiber.Map{
		"ok":      true,
		"message": "logged out",
	})
}

func (h *AuthHandler) writeAuditLog(c *fiber.Ctx, userID, action, modelName, recordID string) {
	if h.auditRepo == nil {
		return
	}
	h.auditRepo.WriteAsync(persistence.AuditLogEntry{
		UserID:        userID,
		Action:        action,
		ModelName:     modelName,
		RecordID:      recordID,
		IPAddress:     c.IP(),
		UserAgent:     c.Get("User-Agent"),
		RequestMethod: c.Method(),
		RequestPath:   c.Path(),
		StatusCode:    c.Response().StatusCode(),
	})
}
