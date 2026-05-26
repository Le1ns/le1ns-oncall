import React from 'react';
import { Badge, Card } from '@grafana/ui';
import { onCallApi } from '../api';
import { ApiState } from '../components/ApiState';

export function PeoplePage() {
  return (
    <ApiState load={onCallApi.users}>
      {(data) => (
        <div style={{ display: 'grid', gap: 12 }}>
          {data.note && <p>{data.note}</p>}
          {data.users.map((user) => (
            <Card key={user.id}>
              <Card.Heading>{user.name || user.login}</Card.Heading>
              <Card.Description>{user.email || user.login}</Card.Description>
              <Card.Meta>
                <Badge text={user.role || 'User'} color="blue" />
              </Card.Meta>
            </Card>
          ))}
        </div>
      )}
    </ApiState>
  );
}
