package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"gocheck/internal/api"
	"gocheck/internal/auth"
	"gocheck/internal/checker"
	"gocheck/internal/db"
	grpc_server "gocheck/internal/grpc"
	"gocheck/internal/notifier"
	"gocheck/internal/snapshot"
	"gocheck/proto/pb"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		Path      string `yaml:"path"`
		URL       string `yaml:"url"`
		Timescale bool   `yaml:"timescale"`
	} `yaml:"database"`
}

func loadConfig() (*Config, error) {
	var config Config

	configPath := "config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Set defaults
	if config.Server.Port == "" {
		config.Server.Port = "8080"
	}
	if config.Database.Path == "" {
		config.Database.Path = "gocheck.db"
	}

	// Override with environment variables if set
	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.Database.URL = dbURL
	}
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	}

	return &config, nil
}

func main() {
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure data directory exists
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database (PostgreSQL, TimescaleDB, or SQLite based on config)
	var database *db.Database
	if config.Database.URL != "" {
		useTimescale := config.Database.Timescale || os.Getenv("USE_TIMESCALE") == "true" || strings.Contains(config.Database.URL, "timescale")
		database, err = db.NewDatabaseWithURLOptions(config.Database.URL, config.Database.Path, useTimescale)
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		if useTimescale {
			log.Printf("Using TimescaleDB database")
		} else {
			log.Printf("Using PostgreSQL database")
		}
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
	sentinelServer := grpc_server.NewSentinelServerWithEngine(database, engine)
	engine.SetSentinelServer(sentinelServer)

	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start check engine: %v", err)
	}
	defer engine.Stop()

	snapshotService := snapshot.NewService(database, engine, dataDir)
	snapshotService.Start()
	defer snapshotService.Stop()

	handlers := api.NewHandlers(database, engine, notifiers, snapshotService, dataDir, sentinelServer)
	authManager := auth.NewAuthManager(database)

	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}
	rpOrigin := os.Getenv("WEBAUTHN_RP_ORIGIN")
	if rpOrigin == "" {
		rpOrigin = "http://localhost:" + config.Server.Port
	}

	webAuthnManager, err := auth.NewWebAuthnManager(rpID, rpOrigin, database)
	if err != nil {
		log.Fatalf("Failed to initialize WebAuthn: %v", err)
	}

	auth.SetGlobalManagers(authManager, webAuthnManager)

	go func() {
		grpcPort := os.Getenv("GRPC_PORT")
		if grpcPort == "" {
			grpcPort = "50051"
		}
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port %s: %v", grpcPort, err)
		}
		s := grpc.NewServer()
		pb.RegisterSentinelServer(s, sentinelServer)
		log.Printf("gRPC server starting on :%s", grpcPort)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	router := mux.NewRouter()

	// Auth routes (no authentication required)
	router.HandleFunc("/api/auth/setup/check", authManager.CheckInitialSetup).Methods("GET")
	router.HandleFunc("/api/auth/setup", authManager.InitialSetup).Methods("POST")
	router.HandleFunc("/api/auth/login", authManager.Login).Methods("POST")
	router.HandleFunc("/api/auth/logout", authManager.Logout).Methods("POST")
	router.HandleFunc("/api/auth/check", authManager.CheckAuth).Methods("GET")

	// Passkey routes
	router.HandleFunc("/api/auth/passkey/begin-registration", webAuthnManager.BeginRegistration).Methods("POST")
	router.HandleFunc("/api/auth/passkey/finish-registration", webAuthnManager.FinishRegistration).Methods("POST")
	router.HandleFunc("/api/auth/passkey/begin-login", webAuthnManager.BeginLogin).Methods("POST")
	router.HandleFunc("/api/auth/passkey/finish-login", webAuthnManager.FinishLogin).Methods("POST")
	router.HandleFunc("/api/auth/passkeys", webAuthnManager.GetPasskeys).Methods("GET")
	router.HandleFunc("/api/auth/passkeys", webAuthnManager.DeletePasskey).Methods("DELETE")

	// API Key management (requires authentication)
	router.HandleFunc("/api/auth/apikeys", authManager.GetAPIKeys).Methods("GET")
	router.HandleFunc("/api/auth/apikeys", authManager.CreateAPIKey).Methods("POST")
	router.HandleFunc("/api/auth/apikeys", authManager.DeleteAPIKey).Methods("DELETE")

	// Protected routes
	router.HandleFunc("/api/checks", authManager.OptionalAuth(handlers.GetChecks)).Methods("GET")
	router.HandleFunc("/api/checks", authManager.OptionalAuth(handlers.CreateCheck)).Methods("POST")
	router.HandleFunc("/api/checks/{id}", authManager.OptionalAuth(handlers.UpdateCheck)).Methods("PUT")
	router.HandleFunc("/api/checks/{id}", authManager.OptionalAuth(handlers.DeleteCheck)).Methods("DELETE")
	router.HandleFunc("/api/checks/{id}/history", authManager.OptionalAuth(handlers.GetCheckHistory)).Methods("GET")
	router.HandleFunc("/api/checks/{id}/stats", authManager.OptionalAuth(handlers.GetCheckStats)).Methods("GET")
	router.HandleFunc("/api/checks/{id}/snapshot", authManager.OptionalAuth(handlers.GetCheckSnapshot)).Methods("GET")
	router.HandleFunc("/api/checks/{id}/snapshot/image", authManager.OptionalAuth(handlers.GetCheckSnapshotImage)).Methods("GET")
	router.HandleFunc("/api/checks/{id}/snapshot/trigger", authManager.OptionalAuth(handlers.TriggerCheckSnapshot)).Methods("POST")
	router.HandleFunc("/api/checks/{id}/trigger", authManager.OptionalAuth(handlers.TriggerCheck)).Methods("POST")
	router.HandleFunc("/api/checks/{id}/trigger/{region}", authManager.OptionalAuth(handlers.TriggerCheckForRegion)).Methods("POST")
	router.HandleFunc("/api/checks/grouped", authManager.OptionalAuth(handlers.GetGroupedChecks)).Methods("GET")
	router.HandleFunc("/api/stream/updates", authManager.OptionalAuth(handlers.StreamCheckUpdates)).Methods("GET")
	router.HandleFunc("/api/stats", authManager.OptionalAuth(handlers.GetStats)).Methods("GET")
	router.HandleFunc("/api/settings", authManager.OptionalAuth(handlers.GetSettings)).Methods("GET")
	router.HandleFunc("/api/settings", authManager.OptionalAuth(handlers.UpdateSettings)).Methods("PUT")
	router.HandleFunc("/api/settings/test-webhook", authManager.OptionalAuth(handlers.TestWebhook)).Methods("POST")
	router.HandleFunc("/api/settings/test-gotify", authManager.OptionalAuth(handlers.TestGotify)).Methods("POST")
	router.HandleFunc("/api/settings/test-tailscale", authManager.OptionalAuth(handlers.TestTailscale)).Methods("POST")
	router.HandleFunc("/api/tailscale/devices", authManager.OptionalAuth(handlers.GetTailscaleDevices)).Methods("GET")
	router.HandleFunc("/api/groups", authManager.OptionalAuth(handlers.GetGroups)).Methods("GET")
	router.HandleFunc("/api/groups", authManager.OptionalAuth(handlers.CreateGroup)).Methods("POST")
	router.HandleFunc("/api/groups/{id}", authManager.OptionalAuth(handlers.UpdateGroup)).Methods("PUT")
	router.HandleFunc("/api/groups/{id}", authManager.OptionalAuth(handlers.DeleteGroup)).Methods("DELETE")
	router.HandleFunc("/api/tags", authManager.OptionalAuth(handlers.GetTags)).Methods("GET")
	router.HandleFunc("/api/tags", authManager.OptionalAuth(handlers.CreateTag)).Methods("POST")
	router.HandleFunc("/api/tags/{id}", authManager.OptionalAuth(handlers.UpdateTag)).Methods("PUT")
	router.HandleFunc("/api/tags/{id}", authManager.OptionalAuth(handlers.DeleteTag)).Methods("DELETE")
	router.HandleFunc("/api/probes", authManager.OptionalAuth(handlers.GetProbes)).Methods("GET")
	router.HandleFunc("/api/probes", authManager.OptionalAuth(handlers.CreateProbe)).Methods("POST")
	router.HandleFunc("/api/probes/{id}", authManager.OptionalAuth(handlers.DeleteProbe)).Methods("DELETE")
	router.HandleFunc("/api/probes/{id}/regenerate-token", authManager.OptionalAuth(handlers.RegenerateProbeToken)).Methods("POST")

	// Serve static files from web/dist (built frontend)
	// In development, run the Vite dev server separately
	webDir := "./web/dist"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		webDir = "./web" // Fallback for development
	}

	// SPA fallback - serve index.html for any non-API, non-file routes
	fs := http.FileServer(http.Dir(webDir))
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := webDir + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// File doesn't exist, serve index.html for SPA routing
			http.ServeFile(w, r, webDir+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	addr := ":" + config.Server.Port
	log.Printf("Server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
