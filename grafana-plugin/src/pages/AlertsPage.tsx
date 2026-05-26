import React from 'react';
import { Alert, Badge, Card } from '@grafana/ui';
import { onCallApi } from '../api';
import { ApiState } from '../components/ApiState';

export function AlertsPage() {
  return (
    <ApiState load={onCallApi.alerts}>
      {(data) => (
        <div style={{ display: 'grid', gap: 12 }}>
          <Alert title="Alertmanager webhook" severity="info">
            POST /api/v1/integrations/alertmanager/webhook with Authorization: Bearer token.
          </Alert>
          <Card>
            <Card.Heading>Incoming alerts</Card.Heading>
            <Card.Description>{data.alerts.length} alerts received in the current scaffold.</Card.Description>
            <Card.Meta>
              <Badge text="Alertmanager" color="orange" />
            </Card.Meta>
          </Card>
        </div>
      )}
    </ApiState>
  );
}
