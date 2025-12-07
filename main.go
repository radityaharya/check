package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
	"gocheck/internal/api"
	"gocheck/internal/checker"
	"gocheck/internal/db"
	"gocheck/internal/notifier"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
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

	if config.Server.Port == "" {
		config.Server.Port = "8080"
	}
	if config.Database.Path == "" {
		config.Database.Path = "gocheck.db"
	}

	// Override with environment variables if set
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.Database.URL = dbURL
	}

	return &config, nil
}

func main() {
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database (PostgreSQL or SQLite based on config)
	var database *db.Database
	if config.Database.URL != "" {
		database, err = db.NewDatabaseWithURL(config.Database.URL, config.Database.Path)
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		log.Printf("Using PostgreSQL database")
	} else {
		database, err = db.NewDatabase(config.Database.Path)
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		log.Printf("Using SQLite database at %s", config.Database.Path)
	}
	defer database.Close()

	// Load notification settings from database or environment variables
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if dbWebhook, err := database.GetSetting("discord_webhook_url"); err == nil && dbWebhook != "" {
		webhookURL = dbWebhook
	}

	gotifyServerURL, _ := database.GetSetting("gotify_server_url")
	gotifyToken, _ := database.GetSetting("gotify_token")

	var notifiers []notifier.Notifier
	if webhookURL != "" {
		notifiers = append(notifiers, notifier.NewDiscordNotifier(webhookURL))
	}
	if gotifyServerURL != "" && gotifyToken != "" {
		notifiers = append(notifiers, notifier.NewGotifyNotifier(gotifyServerURL, gotifyToken))
	}

	engine := checker.NewEngine(database, notifiers)

	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start check engine: %v", err)
	}
	defer engine.Stop()

	handlers := api.NewHandlers(database, engine, notifiers)

	router := mux.NewRouter()

	router.HandleFunc("/api/checks", handlers.GetChecks).Methods("GET")
	router.HandleFunc("/api/checks", handlers.CreateCheck).Methods("POST")
	router.HandleFunc("/api/checks/{id}", handlers.UpdateCheck).Methods("PUT")
	router.HandleFunc("/api/checks/{id}", handlers.DeleteCheck).Methods("DELETE")
	router.HandleFunc("/api/checks/{id}/history", handlers.GetCheckHistory).Methods("GET")
	router.HandleFunc("/api/checks/grouped", handlers.GetGroupedChecks).Methods("GET")
	router.HandleFunc("/api/stream/updates", handlers.StreamCheckUpdates).Methods("GET")
	router.HandleFunc("/api/stats", handlers.GetStats).Methods("GET")
	router.HandleFunc("/api/settings", handlers.GetSettings).Methods("GET")
	router.HandleFunc("/api/settings", handlers.UpdateSettings).Methods("PUT")
	router.HandleFunc("/api/settings/test-webhook", handlers.TestWebhook).Methods("POST")
	router.HandleFunc("/api/settings/test-gotify", handlers.TestGotify).Methods("POST")
	router.HandleFunc("/api/settings/test-tailscale", handlers.TestTailscale).Methods("POST")
	router.HandleFunc("/api/tailscale/devices", handlers.GetTailscaleDevices).Methods("GET")
	router.HandleFunc("/api/groups", handlers.GetGroups).Methods("GET")
	router.HandleFunc("/api/groups", handlers.CreateGroup).Methods("POST")
	router.HandleFunc("/api/groups/{id}", handlers.UpdateGroup).Methods("PUT")
	router.HandleFunc("/api/groups/{id}", handlers.DeleteGroup).Methods("DELETE")
	router.HandleFunc("/api/tags", handlers.GetTags).Methods("GET")
	router.HandleFunc("/api/tags", handlers.CreateTag).Methods("POST")
	router.HandleFunc("/api/tags/{id}", handlers.UpdateTag).Methods("PUT")
	router.HandleFunc("/api/tags/{id}", handlers.DeleteTag).Methods("DELETE")

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))

	addr := ":" + config.Server.Port
	log.Printf("Server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
