# OnCall Platform for Grafana

Self-hosted on-call platform built as a Grafana App Plugin plus a separate backend service.

## Scope

- Grafana App Plugin with an `OnCall` navigation section.
- Go backend API for schedules, alerts, Jira, notifications, and reports.
- PostgreSQL storage.
- Alertmanager webhook endpoint.
- Alertmanager alert storage and assignment to the active duty shift.
- Jira Cloud integration with auto/manual issue creation and deduplication by alert fingerprint.
- Telegram and Lark notification routing from UI settings.
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

This starts:

- Grafana: `http://localhost:3000`
- OnCall API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Jira mock (WireMock): `http://localhost:8081`

Default local credentials:

```text
admin / admin
```

Backend health endpoint:

```text
http://localhost:8080/health
```

Jira mock admin:

```text
http://localhost:8081/__admin/mappings
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
```

Stored alerts are available in the Grafana plugin `Alerts` page and through:

```text
GET http://localhost:8080/api/v1/alerts
```

The alerts endpoint supports period filters and limit:

```text
GET /api/v1/alerts?from=2026-05-01&to=2026-05-28&limit=200
```

Alert intake accepts all severities by default, while routing rules decide who gets notified:

```text
ONCALL_ALERTMANAGER_ALLOWED_SEVERITIES=*
ONCALL_ALERTMANAGER_MAX_ALERTS_PER_GROUP=20
ONCALL_ALERTMANAGER_RATE_WINDOW=10m
ONCALL_ALERTMANAGER_MAX_ALERTS_PER_WEBHOOK=100
```

All Alertmanager severities are accepted by default, and notification routing decides which severities are sent to each person or group. The per-group window and per-webhook cap protect Jira/task creation from incident cascades where one root cause fans out into many secondary alerts. Set `ONCALL_ALERTMANAGER_ALLOWED_SEVERITIES=critical,fatal` if the intake itself should drop lower-severity alerts before storage.

## Notifications

Notification delivery is configured in `Settings` under the `Receivers` and `Notifications` collapses:

1. Add a receiver with delivery credentials.
2. Add notification routing rules that point to a receiver.

Supported receiver types:

- `Telegram`: bot token.
- `Lark`: group bot webhook URL.

Supported target types:

- `Person`: Telegram only. Select a Grafana user, set chat ID, and choose severities. Alerts are sent only when that user is the active on-call assignee.
- `Group`: Telegram or Lark. Set a group name and severities. Telegram rules also need a chat ID; Lark uses the receiver webhook.

Telegram environment configuration is still supported as a local fallback for critical alerts when no database routing rule matches:

```text
ONCALL_TELEGRAM_BOT_TOKEN=...
ONCALL_TELEGRAM_CHAT_ID=...
```

For direct messages Telegram requires the user to open the bot and send `/start`; bots cannot resolve or message a private user by username alone. Use the numeric chat ID in the plugin form.

Current Telegram events:

- new firing Alertmanager alert matching a configured severity route;
- manually created on-call shift.

## Jira Integration

Jira integration is configured in `Settings` under `Jira integration` (visible to Grafana admins only).

Capabilities:

- Enable/disable integration in UI.
- Configure Jira URL, project key, issue type, summary/description templates, labels.
- Auto-create Jira issue on firing alerts.
- Manual issue creation from `Alerts` page when needed.
- Deduplication: repeated alert with same fingerprint does not create duplicate Jira issue.

Credentials are provided from environment:

```text
ONCALL_JIRA_EMAIL=...
ONCALL_JIRA_API_TOKEN=...
```

Base URL can be set in UI. For local development default `.env.example` points to the bundled Jira mock:

```text
ONCALL_JIRA_BASE_URL=http://jira-mock:8081
```

## UI Notes

- `Settings` has three collapses:
  - `Jira integration` (admin only)
  - `Receivers`
  - `Notifications`
- `Alerts` supports filtering by period and max rows, and shows assigned on-call user plus Jira link/status.

## Architecture

Grafana owns the UI. The backend owns all durable behavior: schedules, alert intake, deduplication, Jira issue creation, notifications, and reports. This keeps the system less sensitive to Grafana upgrades.
