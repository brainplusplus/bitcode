package internal

import (
	"log"
	"strings"
	"time"

	"github.com/bitcode-framework/bitcode/internal/infrastructure/cache"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	infrastorage "github.com/bitcode-framework/bitcode/internal/infrastructure/storage"
	"github.com/bitcode-framework/bitcode/internal/presentation/middleware"
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
	v.SetDefault("database.schema", "public")
	v.SetDefault("cache.driver", "memory")
	v.SetDefault("cache.redis_url", "")
	v.SetDefault("tenant.enabled", false)
	v.SetDefault("tenant.strategy", "header")
	v.SetDefault("tenant.header", "X-Tenant-ID")
	v.SetDefault("tenant.isolation", "shared_table")
	v.SetDefault("tenant.column", "tenant_id")
	v.SetDefault("global_module_dir", "")

	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.max", 100)
	v.SetDefault("rate_limit.window", "1m")
	v.SetDefault("rate_limit.auth_max", 5)
	v.SetDefault("rate_limit.auth_window", "1m")

	v.SetDefault("smtp.host", "")
	v.SetDefault("smtp.port", 587)
	v.SetDefault("smtp.user", "")
	v.SetDefault("smtp.password", "")
	v.SetDefault("smtp.from", "")
	v.SetDefault("smtp.tls", true)

	v.SetDefault("encryption_key", "")

	v.SetDefault("security.ip_whitelist_enabled", false)
	v.SetDefault("security.ip_whitelist", []string{})
	v.SetDefault("security.ip_whitelist_admin_only", true)
	v.SetDefault("security.session_duration", "24h")
	v.SetDefault("security.cookie_secure", false)
	v.SetDefault("security.cookie_samesite", "Lax")

	v.SetDefault("auth.register_enabled", false)

	v.SetDefault("app.mode", "online")

	v.SetDefault("execution_log.enabled", true)
	v.SetDefault("execution_log.save_input", true)
	v.SetDefault("execution_log.save_output", true)
	v.SetDefault("execution_log.save_steps", true)
	v.SetDefault("execution_log.save_on_success", true)
	v.SetDefault("execution_log.max_age", "30d")
	v.SetDefault("execution_log.max_records", 100000)
	v.SetDefault("execution_log.cleanup_interval", "1h")
	v.SetDefault("execution_log.max_input_size", 10240)
	v.SetDefault("execution_log.max_output_size", 10240)

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
	v.BindEnv("database.schema", "DB_SCHEMA")
	v.BindEnv("cache.driver", "CACHE_DRIVER")
	v.BindEnv("cache.redis_url", "REDIS_URL")
	v.BindEnv("tenant.enabled", "TENANT_ENABLED")
	v.BindEnv("tenant.strategy", "TENANT_STRATEGY")
	v.BindEnv("tenant.header", "TENANT_HEADER")
	v.BindEnv("tenant.isolation", "TENANT_ISOLATION")
	v.BindEnv("tenant.column", "TENANT_COLUMN")
	v.BindEnv("global_module_dir", "GLOBAL_MODULE_DIR")

	v.BindEnv("rate_limit.enabled", "RATE_LIMIT_ENABLED")
	v.BindEnv("rate_limit.max", "RATE_LIMIT_MAX")
	v.BindEnv("rate_limit.window", "RATE_LIMIT_WINDOW")
	v.BindEnv("rate_limit.auth_max", "RATE_LIMIT_AUTH_MAX")
	v.BindEnv("rate_limit.auth_window", "RATE_LIMIT_AUTH_WINDOW")

	v.BindEnv("smtp.host", "SMTP_HOST")
	v.BindEnv("smtp.port", "SMTP_PORT")
	v.BindEnv("smtp.user", "SMTP_USER")
	v.BindEnv("smtp.password", "SMTP_PASSWORD")
	v.BindEnv("smtp.from", "SMTP_FROM")
	v.BindEnv("smtp.tls", "SMTP_TLS")

	v.BindEnv("encryption_key", "ENCRYPTION_KEY")

	v.BindEnv("security.ip_whitelist_enabled", "SECURITY_IP_WHITELIST_ENABLED")
	v.BindEnv("security.ip_whitelist", "SECURITY_IP_WHITELIST")
	v.BindEnv("security.ip_whitelist_admin_only", "SECURITY_IP_WHITELIST_ADMIN_ONLY")
	v.BindEnv("security.session_duration", "SECURITY_SESSION_DURATION")
	v.BindEnv("security.cookie_secure", "SECURITY_COOKIE_SECURE")
	v.BindEnv("security.cookie_samesite", "SECURITY_COOKIE_SAMESITE")

	v.BindEnv("auth.register_enabled", "AUTH_REGISTER_ENABLED")

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

	sessionDuration, _ := time.ParseDuration(v.GetString("security.session_duration"))
	if sessionDuration == 0 {
		sessionDuration = 24 * time.Hour
	}

	rateLimitWindow, _ := time.ParseDuration(v.GetString("rate_limit.window"))
	if rateLimitWindow == 0 {
		rateLimitWindow = 1 * time.Minute
	}
	rateLimitAuthWindow, _ := time.ParseDuration(v.GetString("rate_limit.auth_window"))
	if rateLimitAuthWindow == 0 {
		rateLimitAuthWindow = 1 * time.Minute
	}

	cfg := AppConfig{
		Port:            v.GetString("port"),
		ModuleDir:       v.GetString("module_dir"),
		GlobalModuleDir: v.GetString("global_module_dir"),
		JWTSecret:       v.GetString("jwt_secret"),
		EncryptionKey:   v.GetString("encryption_key"),
		DB: persistence.DatabaseConfig{
			Driver:     v.GetString("database.driver"),
			Host:       v.GetString("database.host"),
			Port:       v.GetInt("database.port"),
			User:       v.GetString("database.user"),
			Password:   v.GetString("database.password"),
			DBName:     v.GetString("database.name"),
			SSLMode:    v.GetString("database.sslmode"),
			SQLitePath: v.GetString("database.sqlite_path"),
		Schema:     v.GetString("database.schema"),
		},
		Cache: cache.CacheConfig{
			Driver:   v.GetString("cache.driver"),
			RedisURL: v.GetString("cache.redis_url"),
		},
		Tenant: middleware.TenantConfig{
			Enabled:   v.GetBool("tenant.enabled"),
			Strategy:  v.GetString("tenant.strategy"),
			Header:    v.GetString("tenant.header"),
			Isolation: v.GetString("tenant.isolation"),
			Column:    v.GetString("tenant.column"),
		},
		RateLimit: middleware.RateLimitConfig{
			Enabled:    v.GetBool("rate_limit.enabled"),
			Max:        v.GetInt("rate_limit.max"),
			Window:     rateLimitWindow,
			AuthMax:    v.GetInt("rate_limit.auth_max"),
			AuthWindow: rateLimitAuthWindow,
		},
		IPWhitelist: middleware.IPWhitelistConfig{
			Enabled:    v.GetBool("security.ip_whitelist_enabled"),
			AllowedIPs: v.GetStringSlice("security.ip_whitelist"),
			AdminOnly:  v.GetBool("security.ip_whitelist_admin_only"),
		},
		Security: SecurityConfig{
			SessionDuration: sessionDuration,
			CookieSecure:    v.GetBool("security.cookie_secure"),
			CookieSameSite:  v.GetString("security.cookie_samesite"),
		},
		SMTP: SMTPConfig{
			Host:     v.GetString("smtp.host"),
			Port:     v.GetInt("smtp.port"),
			User:     v.GetString("smtp.user"),
			Password: v.GetString("smtp.password"),
			From:     v.GetString("smtp.from"),
			TLS:      v.GetBool("smtp.tls"),
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
		AppMode: v.GetString("app.mode"),
	}

	return cfg, nil
}
