package bridge

import (
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

type auditBridge struct {
	repo    *persistence.AuditLogRepository
	session Session
	module  string
}

func newAuditBridge(repo *persistence.AuditLogRepository, session Session, module string) *auditBridge {
	return &auditBridge{repo: repo, session: session, module: module}
}

func (a *auditBridge) Log(opts AuditOptions) error {
	if opts.Action == "" {
		return NewError(ErrValidation, "audit action is required")
	}
	entry := persistence.AuditLogEntry{
		UserID:    a.session.UserID,
		Action:    opts.Action,
		ModelName: opts.Model,
		RecordID:  opts.RecordID,
		Changes:   opts.Detail,
	}
	return a.repo.Write(entry)
}
