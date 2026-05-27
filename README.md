# OnCall Platform for Grafana

Self-hosted on-call platform built as a Grafana App Plugin plus a separate backend service.

## Scope

- Grafana App Plugin with an `OnCall` navigation section.
- Go backend API for schedules, alerts, Jira, notifications, and reports.
- PostgreSQL storage.
- Alertmanager webhook endpoint.
- Alertmanager alert storage and assignment to the active duty shift.
- Jira Cloud integration placeholder.
- Telegram and Lark notification placeholders.
- Docker Compose for local development, Helm directory reserved for Kubernetes packaging.

## Local Development

Copy the sample environment file:

```bash
cp .env.example .env
```

Run the backend locally:

```bash
cd backend
go test ./...
go run ./cmd/oncall-api
```

Start the local stack:

```bash
docker compose up --build
```

Grafana 12.4.3 will be available at:

```text
http://localhost:3000
```

Default local credentials:

```text
admin / admin
```

Backend health endpoint:

```text
http://localhost:8080/health
```

## Plugin Build

The Grafana plugin source is in `grafana-plugin/`.

```bash
cd grafana-plugin
npm install
npm run build
```

The Docker Compose stack also builds the plugin in a `node:22-alpine` container and mounts the built `dist/` output into Grafana. Grafana is configured in development mode to allow the unsigned plugin.

## Alertmanager Webhook

Local endpoint:

```text
POST http://localhost:8080/api/v1/integrations/alertmanager/webhook
Authorization: Bearer dev-alertmanager-token
```

Stored alerts are available in the Grafana plugin `Alerts` page and through:

```text
GET http://localhost:8080/api/v1/alerts
```

Alert intake is intentionally conservative by default:

```text
ONCALL_ALERTMANAGER_ALLOWED_SEVERITIES=critical,fatal
ONCALL_ALERTMANAGER_MAX_ALERTS_PER_GROUP=20
ONCALL_ALERTMANAGER_RATE_WINDOW=10m
ONCALL_ALERTMANAGER_MAX_ALERTS_PER_WEBHOOK=100
```

All Alertmanager severities are accepted by default, and notification routing decides which severities are sent to each person or group. The per-group window and per-webhook cap protect Jira/task creation from incident cascades where one root cause fans out into many secondary alerts. Set `ONCALL_ALERTMANAGER_ALLOWED_SEVERITIES=critical,fatal` if the intake itself should drop lower-severity alerts before storage.

## Notifications

Notification routing is configured from the Grafana plugin `Settings` page. Click `Add notification`, choose a provider and target type, then select severities and a chat target.

Supported target types:

- `Person`: select a Grafana user, set chat ID, and choose severities. Alerts are sent only when that user is the active on-call assignee.
- `Group`: set a group name, chat ID, and severities. Matching alerts are sent regardless of the assigned person.

Telegram still needs a bot token in backend configuration:

```text
ONCALL_TELEGRAM_BOT_TOKEN=...
```

For direct messages Telegram requires the user to open the bot and send `/start`; bots cannot resolve or message a private user by username alone. Use the numeric chat ID in the plugin form. `ONCALL_TELEGRAM_CHAT_ID` and `ONCALL_TELEGRAM_CHAT_MAPPING` are still supported as a local fallback for critical alerts when no database routing rule matches.

Current Telegram events:

- new firing critical/fatal Alertmanager alert;
- manually created on-call shift.

## Architecture

Grafana owns the UI. The backend owns all durable behavior: schedules, alert intake, deduplication, Jira issue creation, notifications, and reports. This keeps the system less sensitive to Grafana upgrades.
