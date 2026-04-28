package api

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
)

type SyncHandler struct {
	db       *gorm.DB
	modelReg *domainModel.Registry
}

func NewSyncHandler(db *gorm.DB, modelReg *domainModel.Registry) *SyncHandler {
	return &SyncHandler{db: db, modelReg: modelReg}
}

func (h *SyncHandler) RegisterDevice(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "Device registration endpoint — coming in Phase 3",
	})
}

func (h *SyncHandler) PushEnvelope(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "Sync push endpoint — coming in Phase 3",
	})
}

func (h *SyncHandler) PullChanges(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "Sync pull endpoint — coming in Phase 3",
	})
}

func (h *SyncHandler) CacheAuth(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "Auth cache endpoint — coming in Phase 5",
	})
}

func (h *SyncHandler) DeviceStatus(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "Device status endpoint — coming in Phase 3",
	})
}

type schemaField struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"`
	Model    string   `json:"model,omitempty"`
}

type schemaModel struct {
	Name       string        `json:"name"`
	Module     string        `json:"module"`
	TableName  string        `json:"table_name"`
	Fields     []schemaField `json:"fields"`
	PrimaryKey interface{}   `json:"primary_key,omitempty"`
}

func (h *SyncHandler) GetSchema(c *fiber.Ctx) error {
	allModels := h.modelReg.List()
	var offlineModels []schemaModel

	for _, m := range allModels {
		if !m.OfflineModule {
			continue
		}

		fields := make([]schemaField, 0, len(m.Fields))
		for name, f := range m.Fields {
			sf := schemaField{
				Name:     name,
				Type:     string(f.Type),
				Required: f.Required,
			}
			if len(f.Options) > 0 {
				sf.Options = f.Options
			}
			if f.Model != "" {
				sf.Model = f.Model
			}
			fields = append(fields, sf)
		}

		tableName := h.modelReg.TableName(m.Name)
		if tableName == "" {
			tableName = m.Name
		}

		sm := schemaModel{
			Name:      m.Name,
			Module:    m.Module,
			TableName: tableName,
			Fields:    fields,
		}
		if m.PrimaryKey != nil {
			sm.PrimaryKey = m.PrimaryKey
		}

		offlineModels = append(offlineModels, sm)
	}

	return c.JSON(fiber.Map{
		"models": offlineModels,
	})
}
