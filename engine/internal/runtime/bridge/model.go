package bridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/domain/model"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"gorm.io/gorm"
)

const defaultSearchLimit = 100
const maxSearchLimit = 10000

type modelFactory struct {
	db            *gorm.DB
	registry      *model.Registry
	permService   *persistence.PermissionService
	repoFactory   func(modelName string, session Session, db *gorm.DB) (*persistence.GenericRepository, *parser.ModelDefinition, error)
}

func newModelFactory(db *gorm.DB, registry *model.Registry, permService *persistence.PermissionService) *modelFactory {
	return &modelFactory{
		db:          db,
		registry:    registry,
		permService: permService,
		repoFactory: func(modelName string, session Session, db *gorm.DB) (*persistence.GenericRepository, *parser.ModelDefinition, error) {
			modelDef, err := registry.Get(modelName)
			if err != nil || modelDef == nil {
				return nil, nil, ErrModelNotFoundFor(modelName)
			}
			tableName := registry.TableName(modelName)
			if tableName == "" {
				tableName = modelName
			}

			var repo *persistence.GenericRepository
			if session.TenantID != "" {
				repo = persistence.NewGenericRepositoryWithModelAndTenant(db, tableName, modelDef, session.TenantID)
			} else {
				repo = persistence.NewGenericRepositoryWithModel(db, tableName, modelDef)
			}
			repo.SetCurrentUser(session.UserID)
			repo.SetLocale(session.Locale)
			return repo, modelDef, nil
		},
	}
}

func (f *modelFactory) Model(name string, session Session, sudo bool) ModelHandle {
	handle := &modelBridge{
		factory:     f,
		modelName:   name,
		session:     session,
		permService: f.permService,
		db:          f.db,
	}
	if sudo {
		return &sudoModelBridge{inner: handle}
	}
	return handle
}

type modelBridge struct {
	factory     *modelFactory
	modelName   string
	session     Session
	permService *persistence.PermissionService
	db          *gorm.DB
}

func (m *modelBridge) getRepo() (*persistence.GenericRepository, *parser.ModelDefinition, error) {
	return m.factory.repoFactory(m.modelName, m.session, m.db)
}

func (m *modelBridge) checkPermission(operation string) error {
	perms, err := m.permService.GetModelPermissions(m.session.UserID, m.modelName)
	if err != nil {
		return nil
	}
	if !perms.Has(operation) {
		return ErrPermissionDeniedFor(m.modelName, operation)
	}
	return nil
}

func (m *modelBridge) Search(opts SearchOptions) ([]map[string]any, error) {
	if err := m.checkPermission("read"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}

	query := buildQuery(opts)
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	results, _, err := repo.FindAll(context.Background(), query, 1, limit)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func (m *modelBridge) Get(id string, opts ...GetOptions) (map[string]any, error) {
	if err := m.checkPermission("read"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}

	record, err := repo.FindByID(context.Background(), id)
	if err != nil {
		return nil, nil
	}
	return record, nil
}

func (m *modelBridge) Create(data map[string]any) (map[string]any, error) {
	if err := m.checkPermission("create"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}

	result, err := repo.Create(context.Background(), data)
	if err != nil {
		if strings.Contains(err.Error(), "validation") || strings.Contains(err.Error(), "required") {
			return nil, NewError(ErrValidation, err.Error())
		}
		return nil, NewError(ErrInternalError, err.Error())
	}
	return result, nil
}

func (m *modelBridge) Write(id string, data map[string]any) error {
	if err := m.checkPermission("write"); err != nil {
		return err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return err
	}

	if err := repo.Update(context.Background(), id, data); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrRecordNotFoundFor(m.modelName, id)
		}
		return NewError(ErrInternalError, err.Error())
	}
	return nil
}

func (m *modelBridge) Delete(id string) error {
	if err := m.checkPermission("delete"); err != nil {
		return err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return err
	}

	if err := repo.Delete(context.Background(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrRecordNotFoundFor(m.modelName, id)
		}
		return NewError(ErrInternalError, err.Error())
	}
	return nil
}

func (m *modelBridge) Count(opts SearchOptions) (int64, error) {
	if err := m.checkPermission("read"); err != nil {
		return 0, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return 0, err
	}

	query := buildQuery(opts)
	count, err := repo.Count(context.Background(), query)
	if err != nil {
		return 0, NewError(ErrInternalError, err.Error())
	}
	return count, nil
}

func (m *modelBridge) Sum(field string, opts SearchOptions) (float64, error) {
	if err := m.checkPermission("read"); err != nil {
		return 0, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return 0, err
	}

	query := buildQuery(opts)
	sum, err := repo.Sum(context.Background(), field, query)
	if err != nil {
		return 0, NewError(ErrInternalError, err.Error())
	}
	return sum, nil
}

func (m *modelBridge) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	if err := m.checkPermission("create"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}

	result, err := repo.Upsert(context.Background(), data, uniqueFields)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	return result, nil
}

func (m *modelBridge) CreateMany(records []map[string]any) ([]map[string]any, error) {
	if err := m.checkPermission("create"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}

	results, err := repo.BulkCreate(context.Background(), records)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	return results, nil
}

func (m *modelBridge) WriteMany(ids []string, data map[string]any) (*BulkResult, error) {
	if err := m.checkPermission("write"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}
	affected, bulkErr := repo.BulkUpdate(context.Background(), ids, data)
	if bulkErr != nil {
		return nil, NewError(ErrInternalError, bulkErr.Error())
	}
	return &BulkResult{Affected: affected}, nil
}

func (m *modelBridge) DeleteMany(ids []string) (*BulkResult, error) {
	if err := m.checkPermission("delete"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}
	affected, bulkErr := repo.BulkDelete(context.Background(), ids)
	if bulkErr != nil {
		return nil, NewError(ErrInternalError, bulkErr.Error())
	}
	return &BulkResult{Affected: affected}, nil
}

func (m *modelBridge) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	if err := m.checkPermission("create"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}
	results, bulkErr := repo.BulkUpsert(context.Background(), records, uniqueFields)
	if bulkErr != nil {
		return nil, NewError(ErrInternalError, bulkErr.Error())
	}
	return results, nil
}

func (m *modelBridge) AddRelation(id string, field string, relatedIDs []string) error {
	if err := m.checkPermission("write"); err != nil {
		return err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return err
	}
	return repo.AddMany2Many(context.Background(), id, field, relatedIDs)
}

func (m *modelBridge) RemoveRelation(id string, field string, relatedIDs []string) error {
	if err := m.checkPermission("write"); err != nil {
		return err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return err
	}
	return repo.RemoveMany2Many(context.Background(), id, field, relatedIDs)
}

func (m *modelBridge) SetRelation(id string, field string, relatedIDs []string) error {
	if err := m.checkPermission("write"); err != nil {
		return err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return err
	}

	existing, err := repo.LoadMany2Many(context.Background(), id, field)
	if err != nil {
		return NewError(ErrInternalError, err.Error())
	}

	var existingIDs []string
	for _, rec := range existing {
		if recID, ok := rec["id"].(string); ok {
			existingIDs = append(existingIDs, recID)
		}
	}

	if len(existingIDs) > 0 {
		if rmErr := repo.RemoveMany2Many(context.Background(), id, field, existingIDs); rmErr != nil {
			return NewError(ErrInternalError, rmErr.Error())
		}
	}

	if len(relatedIDs) > 0 {
		return repo.AddMany2Many(context.Background(), id, field, relatedIDs)
	}
	return nil
}

func (m *modelBridge) LoadRelation(id string, field string) ([]map[string]any, error) {
	if err := m.checkPermission("read"); err != nil {
		return nil, err
	}
	repo, _, err := m.getRepo()
	if err != nil {
		return nil, err
	}

	results, err := repo.LoadMany2Many(context.Background(), id, field)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	return results, nil
}

func (m *modelBridge) Sudo() SudoModelHandle {
	return &sudoModelBridge{inner: m}
}

type sudoModelBridge struct {
	inner          *modelBridge
	skipValidation bool
	tenantOverride string
}

func (s *sudoModelBridge) Search(opts SearchOptions) ([]map[string]any, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}

	query := buildQuery(opts)
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	results, _, err := repo.FindAll(context.Background(), query, 1, limit)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func (s *sudoModelBridge) Get(id string, opts ...GetOptions) (map[string]any, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	record, err := repo.FindByID(context.Background(), id)
	if err != nil {
		return nil, nil
	}
	return record, nil
}

func (s *sudoModelBridge) Create(data map[string]any) (map[string]any, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	result, err := repo.Create(context.Background(), data)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	return result, nil
}

func (s *sudoModelBridge) Write(id string, data map[string]any) error {
	repo, _, err := s.getRepo()
	if err != nil {
		return err
	}
	if err := repo.Update(context.Background(), id, data); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrRecordNotFoundFor(s.inner.modelName, id)
		}
		return NewError(ErrInternalError, err.Error())
	}
	return nil
}

func (s *sudoModelBridge) Delete(id string) error {
	repo, _, err := s.getRepo()
	if err != nil {
		return err
	}
	return repo.Delete(context.Background(), id)
}

func (s *sudoModelBridge) Count(opts SearchOptions) (int64, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return 0, err
	}
	return repo.Count(context.Background(), buildQuery(opts))
}

func (s *sudoModelBridge) Sum(field string, opts SearchOptions) (float64, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return 0, err
	}
	return repo.Sum(context.Background(), field, buildQuery(opts))
}

func (s *sudoModelBridge) Upsert(data map[string]any, uniqueFields []string) (map[string]any, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	return repo.Upsert(context.Background(), data, uniqueFields)
}

func (s *sudoModelBridge) CreateMany(records []map[string]any) ([]map[string]any, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	return repo.BulkCreate(context.Background(), records)
}

func (s *sudoModelBridge) WriteMany(ids []string, data map[string]any) (*BulkResult, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	affected, bulkErr := repo.BulkUpdate(context.Background(), ids, data)
	if bulkErr != nil {
		return nil, NewError(ErrInternalError, bulkErr.Error())
	}
	return &BulkResult{Affected: affected}, nil
}

func (s *sudoModelBridge) DeleteMany(ids []string) (*BulkResult, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	affected, bulkErr := repo.BulkDelete(context.Background(), ids)
	if bulkErr != nil {
		return nil, NewError(ErrInternalError, bulkErr.Error())
	}
	return &BulkResult{Affected: affected}, nil
}

func (s *sudoModelBridge) UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	results, bulkErr := repo.BulkUpsert(context.Background(), records, uniqueFields)
	if bulkErr != nil {
		return nil, NewError(ErrInternalError, bulkErr.Error())
	}
	return results, nil
}

func (s *sudoModelBridge) AddRelation(id string, field string, relatedIDs []string) error {
	repo, _, err := s.getRepo()
	if err != nil {
		return err
	}
	return repo.AddMany2Many(context.Background(), id, field, relatedIDs)
}

func (s *sudoModelBridge) RemoveRelation(id string, field string, relatedIDs []string) error {
	repo, _, err := s.getRepo()
	if err != nil {
		return err
	}
	return repo.RemoveMany2Many(context.Background(), id, field, relatedIDs)
}

func (s *sudoModelBridge) SetRelation(id string, field string, relatedIDs []string) error {
	repo, _, err := s.getRepo()
	if err != nil {
		return err
	}

	existing, err := repo.LoadMany2Many(context.Background(), id, field)
	if err != nil {
		return NewError(ErrInternalError, err.Error())
	}

	var existingIDs []string
	for _, rec := range existing {
		if recID, ok := rec["id"].(string); ok {
			existingIDs = append(existingIDs, recID)
		}
	}

	if len(existingIDs) > 0 {
		if rmErr := repo.RemoveMany2Many(context.Background(), id, field, existingIDs); rmErr != nil {
			return NewError(ErrInternalError, rmErr.Error())
		}
	}

	if len(relatedIDs) > 0 {
		return repo.AddMany2Many(context.Background(), id, field, relatedIDs)
	}
	return nil
}

func (s *sudoModelBridge) LoadRelation(id string, field string) ([]map[string]any, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	return repo.LoadMany2Many(context.Background(), id, field)
}

func (s *sudoModelBridge) Sudo() SudoModelHandle {
	return s
}

func (s *sudoModelBridge) HardDelete(id string) error {
	repo, _, err := s.getRepo()
	if err != nil {
		return err
	}
	return repo.HardDelete(context.Background(), id)
}

func (s *sudoModelBridge) HardDeleteMany(ids []string) (*BulkResult, error) {
	repo, _, err := s.getRepo()
	if err != nil {
		return nil, err
	}
	affected, bulkErr := repo.BulkHardDelete(context.Background(), ids)
	if bulkErr != nil {
		return nil, NewError(ErrInternalError, bulkErr.Error())
	}
	return &BulkResult{Affected: affected}, nil
}

func (s *sudoModelBridge) WithTenant(tenantID string) SudoModelHandle {
	clone := *s
	clone.tenantOverride = tenantID
	return &clone
}

func (s *sudoModelBridge) SkipValidation() SudoModelHandle {
	clone := *s
	clone.skipValidation = true
	return &clone
}

func (s *sudoModelBridge) getRepo() (*persistence.GenericRepository, *parser.ModelDefinition, error) {
	session := s.inner.session
	if s.tenantOverride != "" {
		session.TenantID = s.tenantOverride
	}
	return s.inner.factory.repoFactory(s.inner.modelName, session, s.inner.db)
}

func buildQuery(opts SearchOptions) *persistence.Query {
	q := persistence.NewQuery()

	if len(opts.Domain) > 0 {
		for _, cond := range opts.Domain {
			if len(cond) >= 3 {
				field := fmt.Sprintf("%v", cond[0])
				op := fmt.Sprintf("%v", cond[1])
				q.Where(field, op, cond[2])
			}
		}
	}

	if len(opts.Fields) > 0 {
		q.SetSelect(opts.Fields...)
	}

	if opts.Order != "" {
		parts := strings.Fields(opts.Order)
		if len(parts) >= 1 {
			dir := "asc"
			if len(parts) >= 2 {
				dir = strings.ToLower(parts[1])
			}
			q.Order(parts[0], dir)
		}
	}

	if opts.Offset > 0 {
		q.Offset = opts.Offset
	}

	if len(opts.Include) > 0 {
		for _, rel := range opts.Include {
			q.WithRelation(rel)
		}
	}

	return q
}
