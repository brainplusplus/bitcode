package storage

type StorageConfig struct {
	Driver            string
	MaxSize           int64
	AllowedExtensions []string
	PathFormat        string
	NameFormat        string
	Local             LocalStorageConfig
	S3                S3StorageConfig
	Thumbnail         ThumbnailConfig
}

type LocalStorageConfig struct {
	Path    string
	BaseURL string
}

type S3StorageConfig struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKey       string
	SecretKey       string
	UsePathStyle    bool
	SignedURLExpiry int
}

type ThumbnailConfig struct {
	Enabled bool
	Width   int
	Height  int
	Quality int
}

func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		Driver:            "local",
		MaxSize:           10 * 1024 * 1024,
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".csv", ".txt", ".zip"},
		PathFormat:        "{model}/{year}/{month}",
		NameFormat:        "{uuid}_{original}{ext}",
		Local: LocalStorageConfig{
			Path:    "uploads",
			BaseURL: "/uploads",
		},
		S3: S3StorageConfig{
			SignedURLExpiry: 3600,
		},
		Thumbnail: ThumbnailConfig{
			Enabled: true,
			Width:   300,
			Height:  300,
			Quality: 85,
		},
	}
}
