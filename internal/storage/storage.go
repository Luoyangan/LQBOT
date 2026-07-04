// Package storage implements the Storage interface using GORM + SQLite.
package storage

import (
	"fmt"
	"sync"

	"github.com/Luoyangan/LQBOT/internal/types"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Storage implements contract.Storage using GORM with SQLite.
type Storage struct {
	mu   sync.RWMutex
	db   *gorm.DB
	data map[string]string // Simple KV cache
}

// New creates a new Storage instance.
func New(cfg types.StorageConfig) (*Storage, error) {
	db, err := gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Auto-migrate the KV table
	if err := db.AutoMigrate(&KVEntry{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return &Storage{
		db:   db,
		data: make(map[string]string),
	}, nil
}

// KVEntry represents a key-value pair in the database.
type KVEntry struct {
	Key   string `gorm:"primaryKey;size:512"`
	Value string `gorm:"type:text;not null"`
}

// Get retrieves a value by key and unmarshals it into dest.
func (s *Storage) Get(key string, dest interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entry KVEntry
	if err := s.db.First(&entry, "key = ?", key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("read key %s: %w", key, err)
	}

	// For now, if dest is a *string, assign directly
	if s, ok := dest.(*string); ok {
		*s = entry.Value
		return nil
	}

	return fmt.Errorf("unsupported destination type %T", dest)
}

// Set stores a key-value pair.
func (s *Storage) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valStr := fmt.Sprintf("%v", value)
	entry := KVEntry{Key: key, Value: valStr}

	if err := s.db.Save(&entry).Error; err != nil {
		return fmt.Errorf("write key %s: %w", key, err)
	}
	return nil
}

// Delete removes a key-value pair.
func (s *Storage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Delete(&KVEntry{}, "key = ?", key).Error; err != nil {
		return fmt.Errorf("delete key %s: %w", key, err)
	}
	return nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("get underlying db: %w", err)
	}
	return sqlDB.Close()
}
