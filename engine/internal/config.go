package internal

import (
	"log"
	"strings"

	"github.com/bitcode-engine/engine/internal/infrastructure/cache"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	infrastorage "github.com/bitcode-engine/engine/internal/infrastructure/storage"
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

	v.SetDefault("storage.driver", "local")
	v.SetDefault("storage.max_size", 10*1024*1024)
	v.SetDefault("storage.allowed_extensions", []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".csv", ".txt", ".zip"})
	v.SetDefault("storage.path_format", "{model}/{year}/{month}")
	v.SetDefault("storage.name_format", "{uuid}_{original}{ext}")
	v.SetDefault("storage.local.path", "uploads")
	v.SetDefault("storage.local.base_url", "/uploads")
	v.SetDefault("storage.s3.bucket", "")
	v.SetDefault("storage.s3.region", "")
	v.SetDefault("storage.s3.endpoint", "")
	v.SetDefault("storage.s3.access_key", "")
	v.SetDefault("storage.s3.secret_key", "")
	v.SetDefault("storage.s3.use_path_style", false)
	v.SetDefault("storage.s3.signed_url_expiry", 3600)
	v.SetDefault("storage.thumbnail.enabled", true)
	v.SetDefault("storage.thumbnail.width", 300)
	v.SetDefault("storage.thumbnail.height", 300)
	v.SetDefault("storage.thumbnail.quality", 85)

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

	v.BindEnv("storage.driver", "STORAGE_DRIVER")
	v.BindEnv("storage.max_size", "STORAGE_MAX_SIZE")
	v.BindEnv("storage.path_format", "STORAGE_PATH_FORMAT")
	v.BindEnv("storage.name_format", "STORAGE_NAME_FORMAT")
	v.BindEnv("storage.local.path", "STORAGE_LOCAL_PATH")
	v.BindEnv("storage.local.base_url", "STORAGE_LOCAL_BASE_URL")
	v.BindEnv("storage.s3.bucket", "STORAGE_S3_BUCKET")
	v.BindEnv("storage.s3.region", "STORAGE_S3_REGION")
	v.BindEnv("storage.s3.endpoint", "STORAGE_S3_ENDPOINT")
	v.BindEnv("storage.s3.access_key", "STORAGE_S3_ACCESS_KEY")
	v.BindEnv("storage.s3.secret_key", "STORAGE_S3_SECRET_KEY")
	v.BindEnv("storage.s3.use_path_style", "STORAGE_S3_USE_PATH_STYLE")
	v.BindEnv("storage.s3.signed_url_expiry", "STORAGE_S3_SIGNED_URL_EXPIRY")
	v.BindEnv("storage.thumbnail.enabled", "STORAGE_THUMBNAIL_ENABLED")
	v.BindEnv("storage.thumbnail.width", "STORAGE_THUMBNAIL_WIDTH")
	v.BindEnv("storage.thumbnail.height", "STORAGE_THUMBNAIL_HEIGHT")
	v.BindEnv("storage.thumbnail.quality", "STORAGE_THUMBNAIL_QUALITY")

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
		Storage: infrastorage.StorageConfig{
			Driver:            v.GetString("storage.driver"),
			MaxSize:           v.GetInt64("storage.max_size"),
			AllowedExtensions: v.GetStringSlice("storage.allowed_extensions"),
			PathFormat:        v.GetString("storage.path_format"),
			NameFormat:        v.GetString("storage.name_format"),
			Local: infrastorage.LocalStorageConfig{
				Path:    v.GetString("storage.local.path"),
				BaseURL: v.GetString("storage.local.base_url"),
			},
			S3: infrastorage.S3StorageConfig{
				Bucket:          v.GetString("storage.s3.bucket"),
				Region:          v.GetString("storage.s3.region"),
				Endpoint:        v.GetString("storage.s3.endpoint"),
				AccessKey:       v.GetString("storage.s3.access_key"),
				SecretKey:       v.GetString("storage.s3.secret_key"),
				UsePathStyle:    v.GetBool("storage.s3.use_path_style"),
				SignedURLExpiry: v.GetInt("storage.s3.signed_url_expiry"),
			},
			Thumbnail: infrastorage.ThumbnailConfig{
				Enabled: v.GetBool("storage.thumbnail.enabled"),
				Width:   v.GetInt("storage.thumbnail.width"),
				Height:  v.GetInt("storage.thumbnail.height"),
				Quality: v.GetInt("storage.thumbnail.quality"),
			},
		},
	}

	return cfg, nil
}
