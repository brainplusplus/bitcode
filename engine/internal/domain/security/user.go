package security

import (
	"time"

	"github.com/bitcode-framework/bitcode/pkg/ddd"
	pkgsec "github.com/bitcode-framework/bitcode/pkg/security"
)

type User struct {
	ddd.BaseAggregate
	Username     string    `json:"username" gorm:"uniqueIndex;size:100"`
	Email        string    `json:"email" gorm:"uniqueIndex;size:255"`
	PasswordHash string    `json:"-" gorm:"size:255"`
	Active       bool      `json:"active" gorm:"default:true"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	Roles        []Role    `json:"roles" gorm:"many2many:user_roles;"`
	Groups       []Group   `json:"groups" gorm:"many2many:user_groups;"`
}

func NewUser(id string, username string, email string, password string) (*User, error) {
	hash, err := pkgsec.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &User{
		BaseAggregate: ddd.BaseAggregate{
			BaseEntity: ddd.BaseEntity{ID: id, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Active:       true,
	}
	u.RaiseEvent(ddd.NewDomainEvent("user.created", id))
	return u, nil
}

func (u *User) CheckPassword(password string) bool {
	return pkgsec.CheckPassword(password, u.PasswordHash)
}

func (u *User) SetPassword(password string) error {
	hash, err := pkgsec.HashPassword(password)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) Activate() {
	u.Active = true
	u.UpdatedAt = time.Now()
	u.RaiseEvent(ddd.NewDomainEvent("user.activated", u.ID))
}

func (u *User) Deactivate() {
	u.Active = false
	u.UpdatedAt = time.Now()
	u.RaiseEvent(ddd.NewDomainEvent("user.deactivated", u.ID))
}

func (u *User) HasPermission(permission string) bool {
	for _, role := range u.Roles {
		if role.HasPermission(permission) {
			return true
		}
	}
	return false
}

func (u *User) InGroup(groupName string) bool {
	for _, g := range u.Groups {
		if g.Name == groupName {
			return true
		}
	}
	return false
}

func (u *User) RoleNames() []string {
	names := make([]string, len(u.Roles))
	for i, r := range u.Roles {
		names[i] = r.Name
	}
	return names
}

func (u *User) GroupNames() []string {
	names := make([]string, len(u.Groups))
	for i, g := range u.Groups {
		names[i] = g.Name
	}
	return names
}
