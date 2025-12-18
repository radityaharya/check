# GoCheck - HTTP Uptime Monitor

A lightweight HTTP uptime monitoring service built with Go, featuring a web dashboard with Alpine.js and Tailwind CSS, TimescaleDB for time-series data, and Discord notifications.

## Features

- HTTP endpoint monitoring with configurable intervals
- Multiple check types: HTTP, Ping, DNS, PostgreSQL, Tailscale
- Real-time status dashboard
- Check history and statistics
- Discord and Gotify notifications on status changes
- **TimescaleDB**: Production-ready time-series database with optimized performance
- Modern web UI with Alpine.js and Tailwind CSS
- Check grouping and tagging
- Retry logic with configurable delays
- Distributed monitoring with probe support

## Database

gocheck uses TimescaleDB for storing monitoring data. TimescaleDB is a PostgreSQL extension optimized for time-series data, providing:

- Automatic time-based partitioning (hypertables)
- Data compression for older data
- Optimized queries for time-series workloads

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd gocheck
```

2. Install dependencies:
```bash
go mod download
```

3. Set up TimescaleDB:
   - Use the provided docker-compose file, or
   - Set `DATABASE_URL` environment variable to your TimescaleDB instance

4. Configure the application:
   - Edit `config.yaml` for basic settings
   - Set notification webhook URLs as needed

5. Run the application:
```bash
go run main.go
```

The web interface will be available at `http://localhost:8080`

## Quick Start with Docker

```bash
docker-compose -f docker-compose.postgres.yml up -d
```

## Configuration

### config.yaml

```yaml
server:
  port: "8080"

database:
  url: "postgres://user:password@localhost:5432/gocheck?sslmode=disable"
```

### Environment Variables

- `CONFIG_PATH` - Override config file path (default: `config.yaml`)
- `DATABASE_URL` - TimescaleDB/PostgreSQL connection string (required)
- `DISCORD_WEBHOOK_URL` - Discord webhook URL (overrides config file)

## Usage

1. Open the web dashboard at `http://localhost:8080`
2. Click "Add Check" to create a new uptime check
3. Configure:
   - Name: Display name for the check
   - URL: HTTP/HTTPS endpoint to monitor
   - Interval: How often to check (in seconds)
   - Timeout: Request timeout (in seconds)
   - Enabled: Whether the check is active

4. The dashboard will automatically refresh every 5 seconds
5. Discord notifications will be sent when a check status changes (up/down)

## API Endpoints

- `GET /api/checks` - List all checks with status
- `POST /api/checks` - Create a new check
- `PUT /api/checks/:id` - Update a check
- `DELETE /api/checks/:id` - Delete a check
- `GET /api/checks/:id/history` - Get check history
- `GET /api/stats` - Get overall statistics

## Building

Build the binary:
```bash
go build -o gocheck main.go
```

Run the binary:
```bash
./gocheck
```

## Discord Setup

1. Go to your Discord server settings
2. Navigate to Integrations > Webhooks
3. Create a new webhook
4. Copy the webhook URL
5. Set it in `config.yaml` or as `DISCORD_WEBHOOK_URL` environment variable

## License

MIT


