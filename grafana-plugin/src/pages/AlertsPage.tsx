import React, { useEffect, useState } from 'react';
import { Alert, Badge, Button } from '@grafana/ui';
import { AlertmanagerAlert, onCallApi } from '../api';

export function AlertsPage() {
  const today = new Date().toISOString().slice(0, 10);
  const weekAgo = new Date(Date.now() - 6 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
  const [from, setFrom] = useState(weekAgo);
  const [to, setTo] = useState(today);
  const [limit, setLimit] = useState(200);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [alerts, setAlerts] = useState<AlertmanagerAlert[]>([]);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await onCallApi.alerts(from, to, limit);
      setAlerts(data.alerts);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const createIssue = async (id: string) => {
    await onCallApi.createAlertJiraIssue(id);
    await load();
  };

  return (
    <div className="oncall-reports-page">
      <Alert title="Alertmanager webhook" severity="info">
        POST /api/v1/integrations/alertmanager/webhook.
      </Alert>
      {error && (
        <Alert title="Alerts loading failed" severity="error">
          {error}
        </Alert>
      )}
      <form
        className="oncall-form-panel oncall-report-config"
        onSubmit={(event) => {
          event.preventDefault();
          load();
        }}
      >
        <div>
          <div className="oncall-kicker">Filter</div>
          <h3>Alert period</h3>
        </div>
        <label>
          From
          <input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        </label>
        <label>
          To
          <input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
        </label>
        <label>
          Max alerts
          <input type="number" min={50} max={1000} step={50} value={limit} onChange={(event) => setLimit(Number(event.target.value) || 200)} />
        </label>
        <Button type="submit" disabled={loading}>
          {loading ? 'Loading...' : 'Apply'}
        </Button>
      </form>

      <section className="oncall-table-panel">
        <div className="oncall-table-title">
          <h3>Incoming alerts</h3>
          <span>{alerts.length} alerts</span>
        </div>

        <table className="oncall-table">
          <thead>
            <tr>
              <th>Status</th>
              <th>Severity</th>
              <th>Alert</th>
              <th>Service</th>
              <th>Assigned</th>
              <th>Jira</th>
              <th>Started</th>
            </tr>
          </thead>
          <tbody>
            {alerts.map((alert) => (
              <tr key={alert.id}>
                <td>
                  <Badge text={alert.status || 'unknown'} color={alert.status === 'resolved' ? 'green' : 'orange'} />
                </td>
                <td>{alert.severity || '-'}</td>
                <td>
                  <strong>{alert.alertName || alert.fingerprint}</strong>
                  {alert.summary && <div className="oncall-muted">{alert.summary}</div>}
                </td>
                <td>{alert.service || '-'}</td>
                <td>{alert.assignedUserName || alert.assignedUserLogin || '-'}</td>
                <td>
                  {alert.jiraIssueUrl ? (
                    <a href={alert.jiraIssueUrl} target="_blank" rel="noreferrer">
                      {alert.jiraIssueKey}
                    </a>
                  ) : (
                    <Button size="sm" onClick={() => createIssue(alert.id)}>
                      Create
                    </Button>
                  )}
                </td>
                <td>{alert.startsAt ? new Date(alert.startsAt).toLocaleString() : '-'}</td>
              </tr>
            ))}
            {alerts.length === 0 && (
              <tr>
                <td colSpan={7}>No alerts in selected period.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </div>
  );
}
