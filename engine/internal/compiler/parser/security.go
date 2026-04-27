package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

var allPermissions = []string{
	"select", "read", "write", "create", "delete", "export",
	"import", "print", "email", "share", "report", "submit",
}

type SecurityACL []string

func (s *SecurityACL) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "all" {
			*s = make([]string, len(allPermissions))
			copy(*s, allPermissions)
			return nil
		}
		return fmt.Errorf("invalid access shorthand %q, only \"all\" is supported", str)
	}

	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("access must be \"all\" or an array of permission names: %w", err)
	}
	*s = arr
	return nil
}

type SecurityRuleDefinition struct {
	Name       string  `json:"name,omitempty"`
	Model      string  `json:"model"`
	Domain     [][]any `json:"domain,omitempty"`
	PermRead   *bool   `json:"perm_read,omitempty"`
	PermWrite  *bool   `json:"perm_write,omitempty"`
	PermCreate *bool   `json:"perm_create,omitempty"`
	PermDelete *bool   `json:"perm_delete,omitempty"`
	Global     bool    `json:"global,omitempty"`
}

func (r *SecurityRuleDefinition) IsPermRead() bool {
	if r.PermRead == nil {
		return true
	}
	return *r.PermRead
}

func (r *SecurityRuleDefinition) IsPermWrite() bool {
	if r.PermWrite == nil {
		return true
	}
	return *r.PermWrite
}

func (r *SecurityRuleDefinition) IsPermCreate() bool {
	if r.PermCreate == nil {
		return true
	}
	return *r.PermCreate
}

func (r *SecurityRuleDefinition) IsPermDelete() bool {
	if r.PermDelete == nil {
		return true
	}
	return *r.PermDelete
}

type SecurityDefinition struct {
	Name     string                        `json:"name"`
	Label    string                        `json:"label,omitempty"`
	Category string                        `json:"category,omitempty"`
	Implies  []string                      `json:"implies,omitempty"`
	Share    bool                          `json:"share,omitempty"`
	Access   map[string]SecurityACL        `json:"access,omitempty"`
	Rules    []SecurityRuleDefinition      `json:"rules,omitempty"`
	Menus    []string                      `json:"menus,omitempty"`
	Pages    []string                      `json:"pages,omitempty"`
	Comment  string                        `json:"comment,omitempty"`
}

func ParseSecurity(data []byte) (*SecurityDefinition, error) {
	var sec SecurityDefinition
	if err := json.Unmarshal(data, &sec); err != nil {
		return nil, fmt.Errorf("invalid security JSON: %w", err)
	}
	if sec.Name == "" {
		return nil, fmt.Errorf("security group name is required")
	}

	permSet := make(map[string]bool, len(allPermissions))
	for _, p := range allPermissions {
		permSet[p] = true
	}

	for model, perms := range sec.Access {
		for _, p := range perms {
			if !permSet[p] {
				return nil, fmt.Errorf("invalid permission %q for model %q", p, model)
			}
		}
	}

	for i, rule := range sec.Rules {
		if rule.Model == "" {
			return nil, fmt.Errorf("rule %d must specify a model", i)
		}
	}

	return &sec, nil
}

func ParseSecurityFile(path string) (*SecurityDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read security file %s: %w", path, err)
	}
	return ParseSecurity(data)
}
