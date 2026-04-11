package sync

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func BuildCacheKey(context, space, stream, app, workdir string) string {
	input := fmt.Sprintf("%s|%s|%s|%s|%s", context, space, stream, app, workdir)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash[:8])
}

func CacheWrite(dir, key string, data []byte) error {
	return os.WriteFile(cachePath(dir, key), data, 0644)
}

func CacheRead(dir, key string, ttl time.Duration) ([]byte, error) {
	path := cachePath(dir, key)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) { return nil, nil }
		return nil, err
	}
	if time.Since(info.ModTime()) > ttl { return nil, nil }
	return os.ReadFile(path)
}

func cachePath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("qs-cache-%s.json", key))
}
