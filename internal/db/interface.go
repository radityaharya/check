package db

import (
	"time"

	"gocheck/internal/models"
)

// DB defines the interface that all database implementations must satisfy
type DB interface {
	Close() error

	// Check operations
	GetAllChecks() ([]models.Check, error)
	GetCheck(id int64) (*models.Check, error)
	CreateCheck(c *models.Check) error
	UpdateCheck(c *models.Check) error
	DeleteCheck(id int64) error
	GetEnabledChecks() ([]models.Check, error)

	// History operations
	AddHistory(h *models.CheckHistory) error
	GetCheckHistory(checkID int64, since *time.Time, limit int) ([]models.CheckHistory, error)
	GetCheckHistoryAggregated(checkID int64, since *time.Time, bucketMinutes int, limit int) ([]models.CheckHistory, error)
	GetLastStatus(checkID int64) (*models.CheckHistory, error)

	// Stats operations
	GetStats(since *time.Time) (*models.Stats, error)

	// Settings operations
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error

	// Group operations
	GetAllGroups() ([]models.Group, error)
	GetGroup(id int64) (*models.Group, error)
	CreateGroup(g *models.Group) error
	UpdateGroup(g *models.Group) error
	DeleteGroup(id int64) error

	// Tag operations
	GetAllTags() ([]models.Tag, error)
	GetTag(id int64) (*models.Tag, error)
	CreateTag(t *models.Tag) error
	UpdateTag(t *models.Tag) error
	DeleteTag(id int64) error
	GetCheckTags(checkID int64) ([]models.Tag, error)
	SetCheckTags(checkID int64, tagIDs []int64) error
}
