import React from 'react';
import { Alert, Badge } from '@grafana/ui';
import { onCallApi } from '../api';
import { ApiState } from '../components/ApiState';

export function AlertsPage() {
  return (
    <ApiState load={onCallApi.alerts}>
      {(data) => (
        <div className="oncall-reports-page">
          <Alert title="Alertmanager webhook" severity="info">
            POST /api/v1/integrations/alertmanager/webhook with Authorization: Bearer dev-alertmanager-token.
          </Alert>

          <section className="oncall-table-panel">
            <div className="oncall-table-title">
              <h3>Incoming alerts</h3>
              <span>{data.alerts.length} stored alerts</span>
            </div>

            <table className="oncall-table">
              <thead>
                <tr>
                  <th>Status</th>
                  <th>Severity</th>
                  <th>Alert</th>
                  <th>Service</th>
                  <th>Assigned</th>
                  <th>Started</th>
                </tr>
              </thead>
              <tbody>
                {data.alerts.map((alert) => (
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
                    <td>{alert.startsAt ? new Date(alert.startsAt).toLocaleString() : '-'}</td>
                  </tr>
                ))}
                {data.alerts.length === 0 && (
                  <tr>
                    <td colSpan={6}>No Alertmanager alerts received yet.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </section>
        </div>
      )}
    </ApiState>
  );
}
