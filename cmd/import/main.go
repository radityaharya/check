package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gocheck/internal/db"
	"gocheck/internal/models"
)

type UptimeKumaMonitor struct {
	ID                      int      `json:"id"`
	Name                    string   `json:"name"`
	Type                    string   `json:"type"`
	URL                     string   `json:"url"`
	Hostname                string   `json:"hostname"`
	Interval                int      `json:"interval"`
	Timeout                 int      `json:"timeout"`
	Active                  bool     `json:"active"`
	AcceptedStatusCodes     []string `json:"accepted_statuscodes"`
	DatabaseConnectionString string  `json:"databaseConnectionString"`
	JSONPath                string   `json:"jsonPath"`
	ExpectedValue           string   `json:"expectedValue"`
	DNSResolveType          string   `json:"dns_resolve_type"`
	Method                  string   `json:"method"`
}

func parseStatusCodes(codes []string) []int {
	if len(codes) == 0 {
		return []int{200}
	}

	var result []int
	seen := make(map[int]bool)

	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}

		if strings.Contains(code, "-") {
			parts := strings.Split(code, "-")
			if len(parts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil && start <= end {
					for i := start; i <= end && i <= 599; i++ {
						if !seen[i] {
							result = append(result, i)
							seen[i] = true
						}
					}
				}
			}
		} else {
			if num, err := strconv.Atoi(code); err == nil && num >= 100 && num <= 599 {
				if !seen[num] {
					result = append(result, num)
					seen[num] = true
				}
			}
		}
	}

	if len(result) == 0 {
		return []int{200}
	}
	return result
}

func mapUptimeKumaType(kumaType string) models.CheckType {
	switch kumaType {
	case "http":
		return models.CheckTypeHTTP
	case "ping":
		return models.CheckTypePing
	case "postgres":
		return models.CheckTypePostgres
	case "json-query":
		return models.CheckTypeJSONHTTP
	case "dns":
		return models.CheckTypeDNS
	default:
		return models.CheckTypeHTTP
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/import/main.go <import.json> [database_path]")
	}

	jsonPath := os.Args[1]
	dbPath := "gocheck.db"
	if len(os.Args) > 2 {
		dbPath = os.Args[2]
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %v", err)
	}

	var kumaData map[string]UptimeKumaMonitor
	if err := json.Unmarshal(data, &kumaData); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	database, err := db.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	imported := 0
	skipped := 0

	for key, monitor := range kumaData {
		if monitor.Type == "group" {
			skipped++
			continue
		}

		if !monitor.Active {
			skipped++
			continue
		}

		checkType := mapUptimeKumaType(monitor.Type)
		check := models.Check{
			Name:            monitor.Name,
			Type:            checkType,
			URL:             monitor.URL,
			IntervalSeconds: monitor.Interval,
			TimeoutSeconds:  monitor.Timeout,
			Enabled:         monitor.Active,
			Method:          monitor.Method,
		}

		if check.Method == "" {
			check.Method = "GET"
		}

		if check.IntervalSeconds <= 0 {
			check.IntervalSeconds = 60
		}
		if check.TimeoutSeconds <= 0 {
			check.TimeoutSeconds = 10
		}

		switch checkType {
		case models.CheckTypeHTTP:
			check.ExpectedStatusCodes = parseStatusCodes(monitor.AcceptedStatusCodes)
			if check.URL == "" || check.URL == "https://" || check.URL == "http://" {
				fmt.Printf("Skipping %s: invalid URL\n", monitor.Name)
				skipped++
				continue
			}

		case models.CheckTypeJSONHTTP:
			check.ExpectedStatusCodes = parseStatusCodes(monitor.AcceptedStatusCodes)
			check.JSONPath = monitor.JSONPath
			check.ExpectedJSONValue = monitor.ExpectedValue
			if check.URL == "" || check.URL == "https://" || check.URL == "http://" {
				fmt.Printf("Skipping %s: invalid URL\n", monitor.Name)
				skipped++
				continue
			}

		case models.CheckTypePing:
			check.Host = monitor.Hostname
			if check.Host == "" {
				fmt.Printf("Skipping %s: no hostname\n", monitor.Name)
				skipped++
				continue
			}

		case models.CheckTypePostgres:
			check.PostgresConnString = monitor.DatabaseConnectionString
			if check.PostgresConnString == "" {
				fmt.Printf("Skipping %s: no connection string\n", monitor.Name)
				skipped++
				continue
			}

		case models.CheckTypeDNS:
			check.DNSHostname = monitor.Hostname
			check.DNSRecordType = monitor.DNSResolveType
			if check.DNSHostname == "" {
				fmt.Printf("Skipping %s: no hostname\n", monitor.Name)
				skipped++
				continue
			}
			if check.DNSRecordType == "" {
				check.DNSRecordType = "A"
			}
		}

		if err := database.CreateCheck(&check); err != nil {
			log.Printf("Failed to import %s (key: %s): %v", monitor.Name, key, err)
			skipped++
			continue
		}

		imported++
		fmt.Printf("Imported: %s (type: %s, id: %d)\n", check.Name, check.Type, check.ID)
	}

	fmt.Printf("\nImport complete: %d imported, %d skipped\n", imported, skipped)
}

