import React from 'react';
import { Badge, Card } from '@grafana/ui';
import { onCallApi } from '../api';
import { ApiState } from '../components/ApiState';

const labels: Record<string, string> = {
  grafanaUsers: 'Grafana users',
  jiraCloud: 'Jira Cloud',
  telegram: 'Telegram',
  lark: 'Lark',
};

export function SettingsPage() {
  return (
    <ApiState load={onCallApi.meta}>
      {(meta) => (
        <div style={{ display: 'grid', gap: 12 }}>
          {Object.entries(meta.features).map(([key, enabled]) => (
            <Card key={key}>
              <Card.Heading>{labels[key] ?? key}</Card.Heading>
              <Card.Description>{enabled ? 'Configured' : 'Not configured'}</Card.Description>
              <Card.Meta>
                <Badge text={enabled ? 'enabled' : 'disabled'} color={enabled ? 'green' : 'red'} />
              </Card.Meta>
            </Card>
          ))}
        </div>
      )}
    </ApiState>
  );
}
