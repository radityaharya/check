# GoCheck - HTTP Uptime Monitor

A lightweight HTTP uptime monitoring service built with Go, featuring a web dashboard with Alpine.js and Tailwind CSS, dual database support (SQLite/PostgreSQL), and Discord notifications.

## Features

- HTTP endpoint monitoring with configurable intervals
- Multiple check types: HTTP, Ping, DNS, PostgreSQL, Tailscale
- Real-time status dashboard
- Check history and statistics
- Discord and Gotify notifications on status changes
- **Dual database support**: SQLite (default) or PostgreSQL for production
- Modern web UI with Alpine.js and Tailwind CSS
- Check grouping and tagging
- Retry logic with configurable delays

## Database Support

gocheck supports both SQLite and PostgreSQL:

- **SQLite** (default): Zero-configuration, file-based database
- **PostgreSQL**: Production-ready with optimized indexing and performance

See [DATABASE.md](DATABASE.md) for detailed configuration options.

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

3. Configure the application:
   - Edit `config.yaml` for basic settings
   - Set `DATABASE_URL` environment variable for PostgreSQL
   - Set notification webhook URLs as needed

4. Run the application:
```bash
go run main.go
```

The web interface will be available at `http://localhost:8080`

## Quick Start with Docker

### SQLite (Default)
```bash
docker-compose up -d
```

### PostgreSQL
```bash
docker-compose -f docker-compose.postgres.yml up -d
```

## Configuration

### config.yaml

```yaml
server:
  port: "8080"

database:
  path: "gocheck.db"  # SQLite database file
  # url: "postgres://user:password@localhost:5432/gocheck?sslmode=disable"  # Optional: PostgreSQL URL
```

### Environment Variables

- `CONFIG_PATH` - Override config file path (default: `config.yaml`)
- `DATABASE_URL` - PostgreSQL connection string (e.g., `postgres://user:password@localhost:5432/gocheck`)
- `DISCORD_WEBHOOK_URL` - Discord webhook URL (overrides config file)

For PostgreSQL configuration details, see [DATABASE.md](DATABASE.md).

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


