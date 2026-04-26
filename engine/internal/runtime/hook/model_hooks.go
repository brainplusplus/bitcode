package hook

import (
	"context"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type ModelHookDispatcher struct {
	dispatcher *Dispatcher
}

func NewModelHookDispatcher(d *Dispatcher) *ModelHookDispatcher {
	return &ModelHookDispatcher{dispatcher: d}
}

func (m *ModelHookDispatcher) DispatchCreate(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, session map[string]any) error {
	if modelDef.Events == nil {
		return nil
	}

	eventCtx := &EventContext{
		Model:      modelDef.Name,
		Module:     modelDef.Module,
		ModulePath: modelDef.ModulePath,
		Operation:  "create",
		Data:       data,
		Session:    session,
	}
	if uid, ok := session["user_id"].(string); ok {
		eventCtx.UserID = uid
	}
	if tid, ok := session["tenant_id"].(string); ok {
		eventCtx.TenantID = tid
	}

	eventCtx.Event = "before_validate"
	if err := m.dispatcher.Dispatch(ctx, "before_validate", modelDef.Events.BeforeValidate, eventCtx); err != nil {
		return err
	}

	eventCtx.Event = "after_validate"
	if err := m.dispatcher.Dispatch(ctx, "after_validate", modelDef.Events.AfterValidate, eventCtx); err != nil {
		return err
	}

	eventCtx.Event = "before_save"
	if err := m.dispatcher.Dispatch(ctx, "before_save", modelDef.Events.BeforeSave, eventCtx); err != nil {
		return err
	}

	eventCtx.Event = "before_create"
	if err := m.dispatcher.Dispatch(ctx, "before_create", modelDef.Events.BeforeCreate, eventCtx); err != nil {
		return err
	}

	return nil
}

func (m *ModelHookDispatcher) DispatchAfterCreate(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, session map[string]any) {
	if modelDef.Events == nil {
		return
	}

	eventCtx := &EventContext{
		Model:      modelDef.Name,
		Module:     modelDef.Module,
		ModulePath: modelDef.ModulePath,
		Operation:  "create",
		Data:       data,
		Session:    session,
	}
	if uid, ok := session["user_id"].(string); ok {
		eventCtx.UserID = uid
	}

	eventCtx.Event = "after_create"
	m.dispatcher.DispatchSplit(ctx, "after_create", modelDef.Events.AfterCreate, eventCtx)

	eventCtx.Event = "after_save"
	m.dispatcher.DispatchSplit(ctx, "after_save", modelDef.Events.AfterSave, eventCtx)
}

func (m *ModelHookDispatcher) DispatchBeforeUpdate(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, oldData map[string]any, changes map[string]any, session map[string]any) error {
	if modelDef.Events == nil {
		return nil
	}

	eventCtx := &EventContext{
		Model:      modelDef.Name,
		Module:     modelDef.Module,
		ModulePath: modelDef.ModulePath,
		Operation:  "update",
		Data:       data,
		OldData:    oldData,
		Changes:    changes,
		Session:    session,
	}
	if uid, ok := session["user_id"].(string); ok {
		eventCtx.UserID = uid
	}
	if tid, ok := session["tenant_id"].(string); ok {
		eventCtx.TenantID = tid
	}

	if err := m.dispatcher.DispatchOnChange(ctx, eventCtx, modelDef.Events, 0); err != nil {
		return err
	}

	eventCtx.Event = "before_validate"
	if err := m.dispatcher.Dispatch(ctx, "before_validate", modelDef.Events.BeforeValidate, eventCtx); err != nil {
		return err
	}

	eventCtx.Event = "after_validate"
	if err := m.dispatcher.Dispatch(ctx, "after_validate", modelDef.Events.AfterValidate, eventCtx); err != nil {
		return err
	}

	eventCtx.Event = "before_save"
	if err := m.dispatcher.Dispatch(ctx, "before_save", modelDef.Events.BeforeSave, eventCtx); err != nil {
		return err
	}

	eventCtx.Event = "before_update"
	if err := m.dispatcher.Dispatch(ctx, "before_update", modelDef.Events.BeforeUpdate, eventCtx); err != nil {
		return err
	}

	return nil
}

func (m *ModelHookDispatcher) DispatchAfterUpdate(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, oldData map[string]any, changes map[string]any, session map[string]any) {
	if modelDef.Events == nil {
		return
	}

	eventCtx := &EventContext{
		Model:      modelDef.Name,
		Module:     modelDef.Module,
		ModulePath: modelDef.ModulePath,
		Operation:  "update",
		Data:       data,
		OldData:    oldData,
		Changes:    changes,
		Session:    session,
	}
	if uid, ok := session["user_id"].(string); ok {
		eventCtx.UserID = uid
	}

	eventCtx.Event = "after_update"
	m.dispatcher.DispatchSplit(ctx, "after_update", modelDef.Events.AfterUpdate, eventCtx)

	eventCtx.Event = "after_save"
	m.dispatcher.DispatchSplit(ctx, "after_save", modelDef.Events.AfterSave, eventCtx)
}

func (m *ModelHookDispatcher) DispatchBeforeDelete(ctx context.Context, modelDef *parser.ModelDefinition, record map[string]any, isSoft bool, session map[string]any) error {
	if modelDef.Events == nil {
		return nil
	}

	eventCtx := &EventContext{
		Model:      modelDef.Name,
		Module:     modelDef.Module,
		ModulePath: modelDef.ModulePath,
		Operation:  "delete",
		Data:       record,
		OldData:    record,
		Session:    session,
	}
	if uid, ok := session["user_id"].(string); ok {
		eventCtx.UserID = uid
	}

	eventCtx.Event = "before_delete"
	if err := m.dispatcher.Dispatch(ctx, "before_delete", modelDef.Events.BeforeDelete, eventCtx); err != nil {
		return err
	}

	if isSoft {
		eventCtx.Event = "before_soft_delete"
		if err := m.dispatcher.Dispatch(ctx, "before_soft_delete", modelDef.Events.BeforeSoftDelete, eventCtx); err != nil {
			return err
		}
	} else {
		eventCtx.Event = "before_hard_delete"
		if err := m.dispatcher.Dispatch(ctx, "before_hard_delete", modelDef.Events.BeforeHardDelete, eventCtx); err != nil {
			return err
		}
	}

	return nil
}

func (m *ModelHookDispatcher) DispatchAfterDelete(ctx context.Context, modelDef *parser.ModelDefinition, record map[string]any, isSoft bool, session map[string]any) {
	if modelDef.Events == nil {
		return
	}

	eventCtx := &EventContext{
		Model:      modelDef.Name,
		Module:     modelDef.Module,
		ModulePath: modelDef.ModulePath,
		Operation:  "delete",
		Data:       record,
		OldData:    record,
		Session:    session,
	}
	if uid, ok := session["user_id"].(string); ok {
		eventCtx.UserID = uid
	}

	eventCtx.Event = "after_delete"
	m.dispatcher.DispatchSplit(ctx, "after_delete", modelDef.Events.AfterDelete, eventCtx)

	if isSoft {
		eventCtx.Event = "after_soft_delete"
		m.dispatcher.DispatchSplit(ctx, "after_soft_delete", modelDef.Events.AfterSoftDelete, eventCtx)
	} else {
		eventCtx.Event = "after_hard_delete"
		m.dispatcher.DispatchSplit(ctx, "after_hard_delete", modelDef.Events.AfterHardDelete, eventCtx)
	}
}

func (m *ModelHookDispatcher) DispatchOnChangeOnly(ctx context.Context, modelDef *parser.ModelDefinition, data map[string]any, changes map[string]any, session map[string]any) error {
	if modelDef.Events == nil || len(modelDef.Events.OnChange) == 0 {
		return nil
	}

	eventCtx := &EventContext{
		Model:      modelDef.Name,
		Module:     modelDef.Module,
		ModulePath: modelDef.ModulePath,
		Operation:  "update",
		Data:       data,
		Changes:    changes,
		Session:    session,
	}
	if uid, ok := session["user_id"].(string); ok {
		eventCtx.UserID = uid
	}

	return m.dispatcher.DispatchOnChange(ctx, eventCtx, modelDef.Events, 0)
}
