package storage

import (
	"context"
	"fmt"
	"time"
)

// Storage abstracts file operations for S3, GCS, or local filesystem.
type Storage interface {
	// Upload stores data at the given key and returns the stored URL.
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)

	// GeneratePresignedPutURL returns a short-lived URL for the FE to PUT an object directly.
	GeneratePresignedPutURL(ctx context.Context, key string, expiry time.Duration, contentType string) (string, error)

	// GeneratePresignedGetURL returns a short-lived URL for downloading/viewing an object.
	GeneratePresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// BuildKey returns a consistent S3/GCS key for a given entity.
func BuildSekolahLegalitasKey(sekolahID string, filename string) string {
	return fmt.Sprintf("sekolah/%s/legalitas/%s", sekolahID, filename)
}

func BuildPendidikFotoKey(pendidikID string, filename string) string {
	return fmt.Sprintf("pendidik/%s/foto/%s", pendidikID, filename)
}
