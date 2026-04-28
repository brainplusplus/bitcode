package bridge

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type envBridge struct {
	viper  *viper.Viper
	rules  SecurityRules
	module string
}

func newEnvBridge(v *viper.Viper, rules SecurityRules, module string) *envBridge {
	return &envBridge{viper: v, rules: rules, module: module}
}

func (e *envBridge) Get(key string) (string, error) {
	upperKey := strings.ToUpper(key)

	for _, secret := range EngineSecrets {
		if upperKey == secret {
			return "", NewErrorf(ErrEnvAccessDenied, "access denied for env key '%s'", key)
		}
	}

	if matchesAny(upperKey, e.rules.EnvDeny) {
		return "", NewErrorf(ErrEnvAccessDenied, "access denied for env key '%s'", key)
	}

	if len(e.rules.EnvAllow) > 0 {
		if !matchesAny(upperKey, e.rules.EnvAllow) {
			return "", NewErrorf(ErrEnvAccessDenied, "access denied for env key '%s'", key)
		}
	} else {
		prefix := strings.ToUpper(e.module) + "_"
		if !strings.HasPrefix(upperKey, prefix) {
			return "", NewErrorf(ErrEnvAccessDenied, "access denied for env key '%s'", key)
		}
	}

	return e.viper.GetString(key), nil
}

func matchesAny(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(strings.ToUpper(pattern), value); matched {
			return true
		}
	}
	return false
}
