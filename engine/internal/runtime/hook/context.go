package hook

type EventContext struct {
	Model      string
	Module     string
	ModulePath string
	Event      string
	Operation  string
	Data       map[string]any
	OldData    map[string]any
	Changes    map[string]any
	UserID     string
	TenantID   string
	Session    map[string]any
	IsBulk     bool
	BulkIndex  int
	BulkTotal  int
}

func (ec *EventContext) Clone() *EventContext {
	clone := &EventContext{
		Model:      ec.Model,
		Module:     ec.Module,
		ModulePath: ec.ModulePath,
		Event:      ec.Event,
		Operation:  ec.Operation,
		UserID:     ec.UserID,
		TenantID:   ec.TenantID,
		IsBulk:     ec.IsBulk,
		BulkIndex:  ec.BulkIndex,
		BulkTotal:  ec.BulkTotal,
	}
	if ec.Data != nil {
		clone.Data = copyMap(ec.Data)
	}
	if ec.OldData != nil {
		clone.OldData = copyMap(ec.OldData)
	}
	if ec.Changes != nil {
		clone.Changes = copyMap(ec.Changes)
	}
	if ec.Session != nil {
		clone.Session = copyMap(ec.Session)
	}
	return clone
}

func (ec *EventContext) ImmutableCopy() *EventContext {
	return ec.Clone()
}

func (ec *EventContext) ToProcessInput() map[string]any {
	input := map[string]any{
		"_model":     ec.Model,
		"_module":    ec.Module,
		"_event":     ec.Event,
		"_operation": ec.Operation,
		"_user_id":   ec.UserID,
		"_tenant_id": ec.TenantID,
		"_is_bulk":   ec.IsBulk,
	}
	for k, v := range ec.Data {
		input[k] = v
	}
	if ec.OldData != nil {
		input["_old"] = ec.OldData
	}
	if ec.Changes != nil {
		input["_changes"] = ec.Changes
	}
	if ec.Session != nil {
		input["_session"] = ec.Session
	}
	return input
}

func copyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
