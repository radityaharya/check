# GoCheck - HTTP Uptime Monitor

A lightweight HTTP uptime monitoring service built with Go, featuring a web dashboard with Alpine.js and Tailwind CSS, SQLite persistence, and Discord notifications.

## Features

- HTTP endpoint monitoring with configurable intervals
- Real-time status dashboard
- Check history and statistics
- Discord webhook notifications on status changes
- SQLite database for persistence
- Modern web UI with Alpine.js and Tailwind CSS

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd gocheck
```

2. Install dependencies:
```bash
go get github.com/gorilla/mux
go get github.com/mattn/go-sqlite3
go get gopkg.in/yaml.v3
```

3. Configure the application:
   - Edit `config.yaml` or set environment variables
   - Set `DISCORD_WEBHOOK_URL` environment variable for Discord notifications

4. Run the application:
```bash
go run main.go
```

The web interface will be available at `http://localhost:8080`

## Configuration

### config.yaml

```yaml
server:
  port: "8080"

database:
  path: "gocheck.db"

discord:
  webhook_url: ""
```

### Environment Variables

- `CONFIG_PATH` - Override config file path (default: `config.yaml`)
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


