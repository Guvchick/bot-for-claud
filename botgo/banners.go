package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// BannerStore resolves per-menu banner images from a folder on disk and caches the
// Telegram file_id of each one after its first upload, so later menus reuse the
// file_id instead of re-uploading the file. The mapping menu->filename comes from
// the environment (see loadBannerFiles); the images live in BANNER_DIR.
type BannerStore struct {
	dir       string
	files     map[string]string
	cachePath string
	mu        sync.Mutex
	cache     map[string]bannerCacheEntry
}

type bannerCacheEntry struct {
	FileID  string `json:"file_id"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
}

func NewBannerStore(cfg Config) *BannerStore {
	return &BannerStore{
		dir:       cfg.BannerDir,
		files:     cfg.BannerFiles,
		cachePath: cfg.BannerCacheFile,
		cache:     map[string]bannerCacheEntry{},
	}
}

func (b *BannerStore) Load() error {
	raw, err := os.ReadFile(b.cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var cache map[string]bannerCacheEntry
	if err := json.Unmarshal(raw, &cache); err != nil {
		return err
	}
	if cache != nil {
		b.mu.Lock()
		b.cache = cache
		b.mu.Unlock()
	}
	return nil
}

func (b *BannerStore) save() error {
	b.mu.Lock()
	raw, err := json.MarshalIndent(b.cache, "", "  ")
	b.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(b.cachePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(b.cachePath, raw, 0o600)
}

// Keys returns the menu keys that have a configured, existing banner file.
func (b *BannerStore) Keys() []string {
	keys := make([]string, 0, len(b.files))
	for key := range b.files {
		if b.Path(key) != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// Path returns the on-disk path of a banner, or "" when it is unset or missing.
func (b *BannerStore) Path(key string) string {
	name := b.files[key]
	if name == "" {
		return ""
	}
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(b.dir, name)
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}
	return path
}

func (b *BannerStore) Has(key string) bool {
	return b.Path(key) != ""
}

// FileID returns the cached Telegram file_id for a banner, but only if the file on
// disk still matches what was cached (same path, size and mod time).
func (b *BannerStore) FileID(key string) string {
	path := b.Path(key)
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := b.cache[key]
	if ok && entry.FileID != "" && entry.Path == path && entry.Size == info.Size() && entry.ModTime == info.ModTime().Unix() {
		return entry.FileID
	}
	return ""
}

// Remember stores the file_id Telegram returned for the current banner file.
func (b *BannerStore) Remember(key, fileID string) {
	path := b.Path(key)
	if path == "" || fileID == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	b.mu.Lock()
	b.cache[key] = bannerCacheEntry{FileID: fileID, Path: path, Size: info.Size(), ModTime: info.ModTime().Unix()}
	b.mu.Unlock()
	_ = b.save()
}

// bannerKeyFor maps a content/message key to its banner key. Most menus share the
// same name; the personal home screen maps to the generic "menu" banner.
func bannerKeyFor(contentKey string) string {
	switch contentKey {
	case "account_home", "menu", "main":
		return "menu"
	default:
		return contentKey
	}
}
