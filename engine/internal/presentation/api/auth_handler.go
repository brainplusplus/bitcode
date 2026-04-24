package api

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/bitcode-engine/engine/internal/infrastructure/cache"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"github.com/bitcode-engine/engine/pkg/email"
	"github.com/bitcode-engine/engine/pkg/security"
	"gorm.io/gorm"
)

type otpEntry struct {
	Code     string
	Attempts int
}

type AuthHandler struct {
	db          *gorm.DB
	jwtCfg      security.JWTConfig
	auditRepo   *persistence.AuditLogRepository
	cache       cache.Cache
	emailSender email.Sender
}

func NewAuthHandler(db *gorm.DB, jwtCfg security.JWTConfig) *AuthHandler {
	return &AuthHandler{db: db, jwtCfg: jwtCfg}
}

func NewAuthHandlerWithAudit(db *gorm.DB, jwtCfg security.JWTConfig, auditRepo *persistence.AuditLogRepository) *AuthHandler {
	return &AuthHandler{db: db, jwtCfg: jwtCfg, auditRepo: auditRepo}
}

func NewAuthHandlerFull(db *gorm.DB, jwtCfg security.JWTConfig, auditRepo *persistence.AuditLogRepository, appCache cache.Cache, sender email.Sender) *AuthHandler {
	return &AuthHandler{db: db, jwtCfg: jwtCfg, auditRepo: auditRepo, cache: appCache, emailSender: sender}
}

func (h *AuthHandler) Register(app *fiber.App) {
	auth := app.Group("/auth")
	auth.Post("/login", h.Login)
	auth.Post("/register", h.RegisterUser)
	auth.Post("/logout", h.Logout)
	auth.Post("/2fa/enable", h.Enable2FA)
	auth.Post("/2fa/disable", h.Disable2FA)
	auth.Post("/2fa/validate", h.Validate2FA)
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
		return c.Status(400).JSON(fiber.Map{"error": "username or email and password required"})
	}

	repo := persistence.NewGenericRepository(h.db, "users")
	loginQuery := persistence.NewQuery().Where("username", "=", body.Username)
	users, _, err := repo.FindAll(c.Context(), loginQuery, 1, 1)
	if (err != nil || len(users) == 0) && strings.Contains(body.Username, "@") {
		emailQuery := persistence.NewQuery().Where("email", "=", body.Username)
		users, _, err = repo.FindAll(c.Context(), emailQuery, 1, 1)
	}
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
	userEmail, _ := user["email"].(string)

	totpEnabled, _ := user["totp_enabled"].(bool)
	if !totpEnabled {
		if te, ok := user["totp_enabled"].(int64); ok && te == 1 {
			totpEnabled = true
		}
	}

	if totpEnabled && h.cache != nil && h.emailSender != nil && h.emailSender.IsConfigured() {
		return h.handle2FALogin(c, userID, username, userEmail)
	}

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

func (h *AuthHandler) handle2FALogin(c *fiber.Ctx, userID, username, userEmail string) error {
	otpCode, err := security.GenerateOTP(6)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate verification code"})
	}

	h.cache.Set(fmt.Sprintf("otp:%s", userID), &otpEntry{Code: otpCode, Attempts: 0}, 5*time.Minute)

	htmlBody, err := email.RenderOTPEmail(otpCode, 5)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to render email"})
	}

	go func() {
		if sendErr := h.emailSender.Send(userEmail, "Your Verification Code", htmlBody); sendErr != nil {
			log.Printf("[2FA] failed to send OTP email to %s: %v", userEmail, sendErr)
		}
	}()

	tempToken, err := security.GenerateToken(
		h.jwtCfg,
		userID,
		username,
		nil,
		nil,
		security.WithPurpose("2fa"),
		security.WithExpiration(10*time.Minute),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate temp token"})
	}

	h.writeAuditLog(c, userID, "2fa_challenge", "user", userID)

	return c.JSON(fiber.Map{
		"requires_2fa": true,
		"temp_token":   tempToken,
		"message":      "verification code sent to your email",
	})
}

func (h *AuthHandler) Validate2FA(c *fiber.Ctx) error {
	var body struct {
		TempToken string `json:"temp_token"`
		Code      string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if body.TempToken == "" || body.Code == "" {
		return c.Status(400).JSON(fiber.Map{"error": "temp_token and code required"})
	}

	claims, err := security.ValidateToken(h.jwtCfg, body.TempToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid or expired temp token"})
	}

	if claims.Purpose != "2fa" {
		return c.Status(400).JSON(fiber.Map{"error": "invalid token purpose"})
	}

	if h.cache == nil {
		return c.Status(500).JSON(fiber.Map{"error": "2FA not available"})
	}

	cacheKey := fmt.Sprintf("otp:%s", claims.UserID)
	cached, ok := h.cache.Get(cacheKey)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "verification code expired, please login again"})
	}

	entry, ok := cached.(*otpEntry)
	if !ok {
		h.cache.Delete(cacheKey)
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	if entry.Attempts >= 3 {
		h.cache.Delete(cacheKey)
		return c.Status(400).JSON(fiber.Map{"error": "too many attempts, please login again"})
	}

	if entry.Code != body.Code {
		entry.Attempts++
		h.cache.Set(cacheKey, entry, 5*time.Minute)
		remaining := 3 - entry.Attempts
		return c.Status(400).JSON(fiber.Map{
			"error":              "invalid verification code",
			"attempts_remaining": remaining,
		})
	}

	h.cache.Delete(cacheKey)

	token, err := security.GenerateToken(h.jwtCfg, claims.UserID, claims.Username, nil, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
	}

	repo := persistence.NewGenericRepository(h.db, "users")
	repo.Update(c.Context(), claims.UserID, map[string]any{
		"last_login":       time.Now(),
		"totp_verified_at": time.Now(),
	})

	h.writeAuditLog(c, claims.UserID, "2fa_verified", "user", claims.UserID)
	h.writeAuditLog(c, claims.UserID, "login", "user", claims.UserID)

	return c.JSON(fiber.Map{
		"token":    token,
		"user_id":  claims.UserID,
		"username": claims.Username,
	})
}

func (h *AuthHandler) Enable2FA(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "authentication required"})
	}

	repo := persistence.NewGenericRepository(h.db, "users")
	if err := repo.Update(c.Context(), userID, map[string]any{"totp_enabled": true}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to enable 2FA"})
	}

	h.writeAuditLog(c, userID, "2fa_enabled", "user", userID)

	return c.JSON(fiber.Map{
		"ok":      true,
		"message": "two-factor authentication enabled",
	})
}

func (h *AuthHandler) Disable2FA(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "authentication required"})
	}

	repo := persistence.NewGenericRepository(h.db, "users")
	if err := repo.Update(c.Context(), userID, map[string]any{"totp_enabled": false, "totp_verified_at": nil}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to disable 2FA"})
	}

	h.writeAuditLog(c, userID, "2fa_disabled", "user", userID)

	return c.JSON(fiber.Map{
		"ok":      true,
		"message": "two-factor authentication disabled",
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

	existing, _, _ := repo.FindAll(c.Context(), persistence.QueryFromDomain([][]any{{"username", "=", body.Username}}), 1, 1)
	if len(existing) > 0 {
		return c.Status(409).JSON(fiber.Map{"error": "username already exists"})
	}

	existingEmail, _, _ := repo.FindAll(c.Context(), persistence.QueryFromDomain([][]any{{"email", "=", body.Email}}), 1, 1)
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
