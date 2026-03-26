package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/asachs01/school_menu_connector/internal/menu"
)

const (
	defaultTTL      = 6 * time.Hour
	defaultCacheDir = "/tmp/menu-cache"
)

// entry wraps a cached menu with its expiration time.
type entry struct {
	Menu      *menu.Menu `json:"menu"`
	ExpiresAt time.Time  `json:"expires_at"`
}

// Cache provides a file-backed menu cache with in-memory reads.
type Cache struct {
	dir string
	ttl time.Duration
	mu  sync.RWMutex
}

// New creates a Cache that stores JSON files in dir with the given TTL.
// If dir is empty, it defaults to /tmp/menu-cache.
// If ttl is zero, it defaults to 6 hours.
func New(dir string, ttl time.Duration) (*Cache, error) {
	if dir == "" {
		dir = defaultCacheDir
	}
	if ttl == 0 {
		ttl = defaultTTL
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache dir: %w", err)
	}

	return &Cache{dir: dir, ttl: ttl}, nil
}

// cacheKey returns a deterministic filename for the given parameters.
func cacheKey(buildingID, districtID, startDate, endDate string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s:%s", buildingID, districtID, startDate, endDate)))
	return fmt.Sprintf("%x.json", h[:16])
}

// Get returns a cached menu if one exists and has not expired.
func (c *Cache) Get(buildingID, districtID, startDate, endDate string) (*menu.Menu, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	path := filepath.Join(c.dir, cacheKey(buildingID, districtID, startDate, endDate))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false
	}

	if time.Now().After(e.ExpiresAt) {
		return nil, false
	}

	return e.Menu, true
}

// Set stores a menu in the cache.
func (c *Cache) Set(buildingID, districtID, startDate, endDate string, m *menu.Menu) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := entry{
		Menu:      m,
		ExpiresAt: time.Now().Add(c.ttl),
	}

	data, err := json.Marshal(e)
	if err != nil {
		return
	}

	path := filepath.Join(c.dir, cacheKey(buildingID, districtID, startDate, endDate))
	_ = os.WriteFile(path, data, 0644)
}
