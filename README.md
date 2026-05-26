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

## Architecture

Grafana owns the UI. The backend owns all durable behavior: schedules, alert intake, deduplication, Jira issue creation, notifications, and reports. This keeps the system less sensitive to Grafana upgrades.
