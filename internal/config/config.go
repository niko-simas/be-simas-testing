package config

import (
	"os"
	"strconv"
)

type Config struct {
	DB        DBConfig
	Redis     RedisConfig
	OAuth     OAuthConfig
	Mail      MailConfig
	S3        S3Config
	JWTSecret string
}

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	Username string
	Password string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

type OAuthConfig struct {
	GoogleClientID       string
	MobileGoogleClientID string
	AndroidPkg           string
}

type MailConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	FromName   string
	Encryption string
}

type S3Config struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

func Load() *Config {
	return &Config{
		DB: DBConfig{
			Host:     get("DB_HOST", "localhost"),
			Port:     getInt("DB_PORT", 5432),
			Name:     get("DB_NAME", "simas-db"),
			Username: get("DB_USERNAME", "postgres"),
			Password: get("DB_PASSWORD", ""),
		},
		Redis: RedisConfig{
			Host:     get("REDIS_HOST", "localhost"),
			Port:     getInt("REDIS_PORT", 6379),
			Password: get("REDIS_PASSWORD", ""),
		},
		OAuth: OAuthConfig{
			GoogleClientID:       get("WEB_GOOGLE_CLIENT_ID", ""),
			MobileGoogleClientID: get("MOBILE_GOOGLE_CLIENT_ID", ""),
			AndroidPkg:           get("ANDROID_PACKAGE_NAME", ""),
		},
		Mail: MailConfig{
			Host:       get("MAIL_HOST", ""),
			Port:       getInt("MAIL_PORT", 587),
			Username:   get("MAIL_USERNAME", ""),
			Password:   get("MAIL_PASSWORD", ""),
			From:       get("MAIL_FROM", ""),
			FromName:   get("MAIL_FROM_NAME", "SIMAS"),
			Encryption: get("MAIL_ENCRYPTION", "tls"),
		},
		S3: S3Config{
			Bucket:    get("S3_BUCKET", ""),
			Region:    get("S3_REGION", "ap-southeast-1"),
			AccessKey: get("S3_ACCESS_KEY", ""),
			SecretKey: get("S3_SECRET_KEY", ""),
			Endpoint:  get("S3_ENDPOINT", ""), // kosong = AWS S3 real, isi = MinIO/GCS S3-compatible
		},
		JWTSecret: get("JWT_SECRET", ""),
	}
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
