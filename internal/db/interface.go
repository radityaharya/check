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
	GetLastStatusByRegion(checkID int64) (map[string]*models.CheckHistory, error)

	// Stats operations
	GetStats(since *time.Time) (*models.Stats, error)

	// Settings operations
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
	GetCheckSnapshot(checkID int64) (*models.CheckSnapshot, error)
	UpsertCheckSnapshot(snapshot *models.CheckSnapshot) error
	GetAllCheckSnapshots() ([]models.CheckSnapshot, error)

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

	// User operations
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int64) (*models.User, error)
	CreateUser(u *models.User) error
	HasUsers() (bool, error)

	// API Key operations
	CreateAPIKey(key *models.APIKey) error
	GetAPIKeyByHash(keyHash string) (*models.APIKey, error)
	GetAPIKeysByUserID(userID int64) ([]models.APIKey, error)
	UpdateAPIKeyLastUsed(id int64) error
	DeleteAPIKey(id int64) error

	// Session operations
	CreateSession(session *models.Session) error
	GetSessionByToken(token string) (*models.Session, error)
	DeleteSession(token string) error
	DeleteExpiredSessions() error
	DeleteUserSessions(userID int64) error

	// WebAuthn Credential operations
	CreateWebAuthnCredential(cred *models.WebAuthnCredential) error
	GetWebAuthnCredentialsByUserID(userID int64) ([]models.WebAuthnCredential, error)
	GetWebAuthnCredentialByID(credID []byte) (*models.WebAuthnCredential, error)
	UpdateWebAuthnCredentialSignCount(credID []byte, signCount uint32) error
	DeleteWebAuthnCredential(id int64) error

	// Probe operations
	CreateProbe(regionCode, ipAddress string) (int64, string, error)
	ValidateProbeToken(token string) (int64, error)
	UpdateProbeStatus(probeID int64, status string) error
	UpdateProbeLastSeen(probeID int64) error
	GetAllProbes() ([]models.Probe, error)
	GetProbeByID(id int64) (*models.Probe, error)
	DeleteProbe(id int64) error
	RegenerateProbeToken(id int64) (string, error)
}
