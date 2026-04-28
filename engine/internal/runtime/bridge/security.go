package bridge

import (
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

type securityBridge struct {
	permService *persistence.PermissionService
	session     Session
}

func newSecurityBridge(permService *persistence.PermissionService, session Session) *securityBridge {
	return &securityBridge{permService: permService, session: session}
}

func (s *securityBridge) Permissions(modelName string) (*ModelPermissions, error) {
	perms, err := s.permService.GetModelPermissions(s.session.UserID, modelName)
	if err != nil {
		return nil, NewErrorf(ErrInternalError, "failed to get permissions for model '%s': %s", modelName, err)
	}
	return &ModelPermissions{
		CanRead:   perms.CanRead,
		CanWrite:  perms.CanWrite,
		CanCreate: perms.CanCreate,
		CanDelete: perms.CanDelete,
		CanPrint:  perms.CanPrint,
		CanEmail:  perms.CanEmail,
		CanExport: perms.CanExport,
		CanImport: perms.CanImport,
		CanClone:  perms.CanClone,
	}, nil
}

func (s *securityBridge) HasGroup(groupName string) (bool, error) {
	groupIDs, err := s.permService.ResolveUserGroupIDs(s.session.UserID)
	if err != nil {
		return false, NewError(ErrInternalError, err.Error())
	}
	for _, g := range groupIDs {
		if g == groupName {
			return true, nil
		}
	}
	return false, nil
}

func (s *securityBridge) Groups() ([]string, error) {
	groupIDs, err := s.permService.ResolveUserGroupIDs(s.session.UserID)
	if err != nil {
		return nil, NewError(ErrInternalError, err.Error())
	}
	return groupIDs, nil
}
