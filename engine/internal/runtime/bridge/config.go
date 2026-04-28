package bridge

import "github.com/spf13/viper"

type configBridge struct {
	viper  *viper.Viper
	module string
}

func newConfigBridge(v *viper.Viper, module string) *configBridge {
	return &configBridge{viper: v, module: module}
}

func (c *configBridge) Get(key string) any {
	fullKey := "modules." + c.module + ".settings." + key
	if c.viper.IsSet(fullKey) {
		return c.viper.Get(fullKey)
	}
	if c.viper.IsSet(key) {
		return c.viper.Get(key)
	}
	return nil
}
