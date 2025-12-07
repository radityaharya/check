package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"gocheck/internal/db"
)

type Config struct {
	Database struct {
		Path string `yaml:"path"`
		URL  string `yaml:"url"`
	} `yaml:"database"`
}

func loadConfig() (*Config, error) {
	configPath := "config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func main() {
	sqlitePath := flag.String("sqlite", "data/gocheck.db", "Path to SQLite database file")
	postgresURL := flag.String("postgres", "", "PostgreSQL connection URL (e.g., postgres://user:pass@localhost:5432/dbname)")
	flag.Parse()

	if *postgresURL == "" {
		// Try environment variable
		*postgresURL = os.Getenv("DATABASE_URL")
		if *postgresURL == "" {
			// Try config.yaml
			config, err := loadConfig()
			if err == nil && config.Database.URL != "" {
				*postgresURL = config.Database.URL
				log.Println("Using PostgreSQL URL from config.yaml")
			}
		}
		
		if *postgresURL == "" {
			log.Fatal("PostgreSQL URL required. Use -postgres flag, DATABASE_URL environment variable, or set database.url in config.yaml")
		}
	}

	log.Printf("Starting migration from SQLite (%s) to PostgreSQL", *sqlitePath)
	log.Printf("PostgreSQL URL: %s", *postgresURL)

	// Open SQLite database
	log.Println("Opening SQLite database...")
	sqliteDB, err := db.NewSQLiteDB(*sqlitePath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer sqliteDB.Close()

	// Open PostgreSQL database
	log.Println("Connecting to PostgreSQL...")
	postgresDB, err := db.NewPostgresDB(*postgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	// Migrate groups first (they're referenced by checks)
	if err := migrateGroups(sqliteDB, postgresDB); err != nil {
		log.Fatalf("Failed to migrate groups: %v", err)
	}

	// Migrate tags
	if err := migrateTags(sqliteDB, postgresDB); err != nil {
		log.Fatalf("Failed to migrate tags: %v", err)
	}

	// Migrate checks
	if err := migrateChecks(sqliteDB, postgresDB); err != nil {
		log.Fatalf("Failed to migrate checks: %v", err)
	}

	// Migrate check history
	if err := migrateCheckHistory(sqliteDB, postgresDB); err != nil {
		log.Fatalf("Failed to migrate check history: %v", err)
	}

	// Migrate settings
	if err := migrateSettings(sqliteDB, postgresDB); err != nil {
		log.Fatalf("Failed to migrate settings: %v", err)
	}

	log.Println("✓ Migration completed successfully!")
}

func migrateGroups(source, dest db.DB) error {
	log.Println("Migrating groups...")
	groups, err := source.GetAllGroups()
	if err != nil {
		return fmt.Errorf("failed to get groups from SQLite: %w", err)
	}

	if len(groups) == 0 {
		log.Println("  No groups to migrate")
		return nil
	}

	// Create a mapping of old IDs to new IDs
	idMap := make(map[int64]int64)

	for _, group := range groups {
		oldID := group.ID
		// Reset ID so PostgreSQL generates a new one
		group.ID = 0
		if err := dest.CreateGroup(&group); err != nil {
			return fmt.Errorf("failed to create group '%s': %w", group.Name, err)
		}
		idMap[oldID] = group.ID
		groupMappings[oldID] = group.ID // Store in global mapping
		log.Printf("  ✓ Migrated group: %s (old ID: %d -> new ID: %d)", group.Name, oldID, group.ID)
	}

	log.Printf("✓ Migrated %d groups", len(groups))
	return nil
}

func migrateTags(source, dest db.DB) error {
	log.Println("Migrating tags...")
	tags, err := source.GetAllTags()
	if err != nil {
		return fmt.Errorf("failed to get tags from SQLite: %w", err)
	}

	if len(tags) == 0 {
		log.Println("  No tags to migrate")
		return nil
	}

	// Create a mapping of old IDs to new IDs
	tagIDMap := make(map[int64]int64)

	for _, tag := range tags {
		oldID := tag.ID
		// Reset ID so PostgreSQL generates a new one
		tag.ID = 0
		if err := dest.CreateTag(&tag); err != nil {
			return fmt.Errorf("failed to create tag '%s': %w", tag.Name, err)
		}
		tagIDMap[oldID] = tag.ID
		log.Printf("  ✓ Migrated tag: %s (old ID: %d -> new ID: %d)", tag.Name, oldID, tag.ID)
	}

	log.Printf("✓ Migrated %d tags", len(tags))
	
	// Store mapping for use in check migration
	tagMappings = tagIDMap
	return nil
}

// Global variable to store tag ID mappings
var tagMappings = make(map[int64]int64)
var checkMappings = make(map[int64]int64)
var groupMappings = make(map[int64]int64)

func migrateChecks(source, dest db.DB) error {
	log.Println("Migrating checks...")
	checks, err := source.GetAllChecks()
	if err != nil {
		return fmt.Errorf("failed to get checks from SQLite: %w", err)
	}

	if len(checks) == 0 {
		log.Println("  No checks to migrate")
		return nil
	}

	for _, check := range checks {
		oldID := check.ID
		oldTags := check.Tags
		
		// Reset ID so PostgreSQL generates a new one
		check.ID = 0
		check.Tags = nil // We'll set tags after creation

		// Update group_id if it exists and map it to new ID
		if check.GroupID != nil {
			if newGroupID, ok := groupMappings[*check.GroupID]; ok {
				check.GroupID = &newGroupID
			} else {
				// Group doesn't exist in mapping, set to nil
				log.Printf("  ⚠ Warning: Check '%s' references non-existent group ID %d, setting to nil", check.Name, *check.GroupID)
				check.GroupID = nil
			}
		}

		if err := dest.CreateCheck(&check); err != nil {
			return fmt.Errorf("failed to create check '%s': %w", check.Name, err)
		}

		// Map old check ID to new check ID
		checkMappings[oldID] = check.ID

		// Migrate tags for this check
		if len(oldTags) > 0 {
			var newTagIDs []int64
			for _, tag := range oldTags {
				if newID, ok := tagMappings[tag.ID]; ok {
					newTagIDs = append(newTagIDs, newID)
				}
			}
			if len(newTagIDs) > 0 {
				if err := dest.SetCheckTags(check.ID, newTagIDs); err != nil {
					log.Printf("  ⚠ Warning: Failed to set tags for check '%s': %v", check.Name, err)
				}
			}
		}

		log.Printf("  ✓ Migrated check: %s (old ID: %d -> new ID: %d)", check.Name, oldID, check.ID)
	}

	log.Printf("✓ Migrated %d checks", len(checks))
	return nil
}

func migrateCheckHistory(source, dest db.DB) error {
	log.Println("Migrating check history...")
	
	// Get all checks to iterate through their history
	checks, err := source.GetAllChecks()
	if err != nil {
		return fmt.Errorf("failed to get checks: %w", err)
	}

	if len(checks) == 0 {
		log.Println("  No checks found, skipping history migration")
		return nil
	}

	totalHistory := 0
	for _, check := range checks {
		oldCheckID := check.ID
		newCheckID, ok := checkMappings[oldCheckID]
		if !ok {
			log.Printf("  ⚠ Warning: No mapping found for check ID %d, skipping history", oldCheckID)
			continue
		}

		// Get all history for this check
		history, err := source.GetCheckHistory(oldCheckID, nil, 0)
		if err != nil {
			log.Printf("  ⚠ Warning: Failed to get history for check ID %d: %v", oldCheckID, err)
			continue
		}

		if len(history) == 0 {
			continue
		}

		// Insert each history record with the new check ID
		for _, h := range history {
			h.ID = 0 // Let PostgreSQL generate new ID
			h.CheckID = newCheckID
			if err := dest.AddHistory(&h); err != nil {
				log.Printf("  ⚠ Warning: Failed to add history record for check ID %d: %v", newCheckID, err)
				continue
			}
		}

		totalHistory += len(history)
		log.Printf("  ✓ Migrated %d history records for check: %s", len(history), check.Name)
	}

	log.Printf("✓ Migrated %d total history records", totalHistory)
	return nil
}

func migrateSettings(source, dest db.DB) error {
	log.Println("Migrating settings...")
	
	// Settings keys we want to migrate
	settingsKeys := []string{
		"discord_webhook_url",
		"gotify_server_url",
		"gotify_token",
		"tailscale_api_key",
		"tailscale_tailnet",
	}

	migratedCount := 0
	for _, key := range settingsKeys {
		value, err := source.GetSetting(key)
		if err != nil {
			log.Printf("  ⚠ Warning: Failed to get setting '%s': %v", key, err)
			continue
		}

		if value == "" {
			continue
		}

		if err := dest.SetSetting(key, value); err != nil {
			log.Printf("  ⚠ Warning: Failed to set setting '%s': %v", key, err)
			continue
		}

		migratedCount++
		log.Printf("  ✓ Migrated setting: %s", key)
	}

	log.Printf("✓ Migrated %d settings", migratedCount)
	return nil
}
