package internal

import (
	"log"
	"strings"

	"github.com/bitcode-engine/engine/internal/infrastructure/cache"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"github.com/bitcode-engine/engine/internal/presentation/middleware"
	"github.com/spf13/viper"
)

func LoadConfig(explicitPath string) (AppConfig, error) {
	v := viper.New()

	v.SetDefault("port", 8080)
	v.SetDefault("module_dir", "modules")
	v.SetDefault("jwt_secret", "change-me-in-production-32chars!")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "bitcode")
	v.SetDefault("database.password", "bitcode")
	v.SetDefault("database.name", "bitcode")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.sqlite_path", "bitcode.db")
	v.SetDefault("cache.driver", "memory")
	v.SetDefault("cache.redis_url", "")
	v.SetDefault("tenant.enabled", false)
	v.SetDefault("tenant.strategy", "header")
	v.SetDefault("tenant.header", "X-Tenant-ID")
	v.SetDefault("global_module_dir", "")

	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.BindEnv("port", "PORT")
	v.BindEnv("module_dir", "MODULE_DIR")
	v.BindEnv("jwt_secret", "JWT_SECRET")
	v.BindEnv("database.driver", "DB_DRIVER")
	v.BindEnv("database.host", "DB_HOST")
	v.BindEnv("database.port", "DB_PORT")
	v.BindEnv("database.user", "DB_USER")
	v.BindEnv("database.password", "DB_PASSWORD")
	v.BindEnv("database.name", "DB_NAME")
	v.BindEnv("database.sslmode", "DB_SSLMODE")
	v.BindEnv("database.sqlite_path", "DB_SQLITE_PATH")
	v.BindEnv("cache.driver", "CACHE_DRIVER")
	v.BindEnv("cache.redis_url", "REDIS_URL")
	v.BindEnv("tenant.enabled", "TENANT_ENABLED")
	v.BindEnv("tenant.strategy", "TENANT_STRATEGY")
	v.BindEnv("tenant.header", "TENANT_HEADER")
	v.BindEnv("global_module_dir", "GLOBAL_MODULE_DIR")

	if explicitPath != "" {
		v.SetConfigFile(explicitPath)
	} else {
		v.SetConfigName("bitcode")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("[CONFIG] no config file found, using defaults + env vars")
		} else {
			log.Printf("[CONFIG] warning: %v", err)
		}
	} else {
		log.Printf("[CONFIG] loaded from %s", v.ConfigFileUsed())
	}

	cfg := AppConfig{
		Port:            v.GetString("port"),
		ModuleDir:       v.GetString("module_dir"),
		GlobalModuleDir: v.GetString("global_module_dir"),
		JWTSecret:       v.GetString("jwt_secret"),
		DB: persistence.DatabaseConfig{
			Driver:     v.GetString("database.driver"),
			Host:       v.GetString("database.host"),
			Port:       v.GetInt("database.port"),
			User:       v.GetString("database.user"),
			Password:   v.GetString("database.password"),
			DBName:     v.GetString("database.name"),
			SSLMode:    v.GetString("database.sslmode"),
			SQLitePath: v.GetString("database.sqlite_path"),
		},
		Cache: cache.CacheConfig{
			Driver:   v.GetString("cache.driver"),
			RedisURL: v.GetString("cache.redis_url"),
		},
		Tenant: middleware.TenantConfig{
			Enabled:  v.GetBool("tenant.enabled"),
			Strategy: v.GetString("tenant.strategy"),
			Header:   v.GetString("tenant.header"),
		},
	}

	return cfg, nil
}
