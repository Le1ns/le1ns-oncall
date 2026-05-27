import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Badge, Button, Card, LoadingPlaceholder } from '@grafana/ui';
import { CreateNotificationChannelInput, FeatureMeta, GrafanaUser, NotificationChannel, onCallApi } from '../api';

const labels: Record<string, string> = {
  grafanaUsers: 'Grafana users',
  jiraCloud: 'Jira Cloud',
  telegram: 'Telegram',
  lark: 'Lark',
};

const severityOptions = ['critical', 'warning', 'fatal'];

export function SettingsPage() {
  const [meta, setMeta] = useState<FeatureMeta | null>(null);
  const [users, setUsers] = useState<GrafanaUser[]>([]);
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<CreateNotificationChannelInput>({
    kind: 'telegram',
    targetType: 'person',
    name: '',
    grafanaUserId: 0,
    userLogin: '',
    userName: '',
    chatId: '',
    severities: ['critical'],
    enabled: true,
  });

  const selectedUser = useMemo(
    () => users.find((user) => user.id === Number(form.grafanaUserId)),
    [form.grafanaUserId, users]
  );

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const [metaData, peopleData, channelData] = await Promise.all([
        onCallApi.meta(),
        onCallApi.users(),
        onCallApi.notificationChannels(),
      ]);
      setMeta(metaData);
      setUsers(peopleData.users);
      setChannels(channelData.channels);
      if (!form.grafanaUserId && peopleData.users[0]) {
        const user = peopleData.users[0];
        setForm((value) => ({
          ...value,
          grafanaUserId: user.id,
          userLogin: user.login,
          userName: user.name || user.login,
        }));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const createChannel = async (event: React.FormEvent) => {
    event.preventDefault();
    const input = normalizeForm(form, selectedUser);
    if (!input.chatId || input.severities.length === 0) {
      setError('Chat ID and at least one severity are required');
      return;
    }
    if (input.targetType === 'person' && !input.userLogin) {
      setError('Select a person for personal notifications');
      return;
    }
    if (input.targetType === 'group' && !input.name) {
      setError('Group name is required');
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await onCallApi.createNotificationChannel(input);
      setForm((value) => ({
        ...value,
        name: '',
        chatId: '',
        severities: ['critical'],
      }));
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const deleteChannel = async (id: string) => {
    setSaving(true);
    setError(null);
    try {
      await onCallApi.deleteNotificationChannel(id);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const toggleSeverity = (severity: string) => {
    const severities = form.severities.includes(severity)
      ? form.severities.filter((item) => item !== severity)
      : [...form.severities, severity];
    setForm({ ...form, severities });
  };

  if (loading) {
    return <LoadingPlaceholder text="Loading OnCall settings" />;
  }

  return (
    <div className="oncall-settings-page">
      {error && (
        <Alert title="Settings action failed" severity="error">
          {error}
        </Alert>
      )}

      {meta && (
        <section className="oncall-settings-grid">
          {Object.entries(meta.features).map(([key, enabled]) => (
            <Card key={key}>
              <Card.Heading>{labels[key] ?? key}</Card.Heading>
              <Card.Description>{enabled ? 'Configured' : 'Not configured'}</Card.Description>
              <Card.Meta>
                <Badge text={enabled ? 'enabled' : 'disabled'} color={enabled ? 'green' : 'red'} />
              </Card.Meta>
            </Card>
          ))}
        </section>
      )}

      <form className="oncall-form-panel oncall-notification-form" onSubmit={createChannel}>
        <div>
          <div className="oncall-kicker">Add notification</div>
          <h3>Telegram and Lark routing</h3>
        </div>

        <label>
          Provider
          <select value={form.kind} onChange={(event) => setForm({ ...form, kind: event.target.value })}>
            <option value="telegram">Telegram</option>
            <option value="lark">Lark</option>
          </select>
        </label>

        <label>
          Type
          <select
            value={form.targetType}
            onChange={(event) =>
              setForm({ ...form, targetType: event.target.value as 'person' | 'group', name: '' })
            }
          >
            <option value="person">Person</option>
            <option value="group">Group</option>
          </select>
        </label>

        {form.targetType === 'person' ? (
          <label>
            Person
            <select
              value={form.grafanaUserId}
              onChange={(event) => {
                const user = users.find((candidate) => candidate.id === Number(event.target.value));
                setForm({
                  ...form,
                  grafanaUserId: user?.id ?? 0,
                  userLogin: user?.login ?? '',
                  userName: user?.name || user?.login || '',
                });
              }}
            >
              {users.map((user) => (
                <option key={user.id} value={user.id}>
                  {user.name || user.login}
                </option>
              ))}
            </select>
          </label>
        ) : (
          <label>
            Group name
            <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
          </label>
        )}

        <label>
          Chat ID
          <input value={form.chatId} onChange={(event) => setForm({ ...form, chatId: event.target.value })} />
        </label>

        <fieldset className="oncall-severity-field">
          <legend>Severity</legend>
          <div>
            {severityOptions.map((severity) => (
              <label key={severity}>
                <input
                  type="checkbox"
                  checked={form.severities.includes(severity)}
                  onChange={() => toggleSeverity(severity)}
                />
                <span>{severity}</span>
              </label>
            ))}
          </div>
        </fieldset>

        <Button type="submit" disabled={saving}>
          {saving ? 'Adding...' : 'Add notification'}
        </Button>
      </form>

      <section className="oncall-table-panel">
        <div className="oncall-table-title">
          <h3>Notification routing</h3>
          <span>{channels.length} channels</span>
        </div>
        <table className="oncall-table">
          <thead>
            <tr>
              <th>Target</th>
              <th>Provider</th>
              <th>Severity</th>
              <th>Chat ID</th>
              <th>Status</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {channels.map((channel) => (
              <tr key={channel.id}>
                <td>
                  <strong>{channel.name || channel.userName || channel.userLogin}</strong>
                  <div className="oncall-muted">{channel.targetType}</div>
                </td>
                <td>{channel.kind}</td>
                <td>
                  <div className="oncall-badge-row">
                    {channel.severities.map((severity) => (
                      <Badge key={severity} text={severity} color={severityColor(severity)} />
                    ))}
                  </div>
                </td>
                <td>{maskChat(channel.chatId)}</td>
                <td>
                  <Badge text={channel.enabled ? 'enabled' : 'disabled'} color={channel.enabled ? 'green' : 'red'} />
                </td>
                <td>
                  <Button variant="destructive" size="sm" disabled={saving} onClick={() => deleteChannel(channel.id)}>
                    Delete
                  </Button>
                </td>
              </tr>
            ))}
            {channels.length === 0 && (
              <tr>
                <td colSpan={6}>No notification channels configured.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </div>
  );
}

function normalizeForm(input: CreateNotificationChannelInput, selectedUser?: GrafanaUser): CreateNotificationChannelInput {
  if (input.targetType === 'group') {
    return {
      ...input,
      name: input.name.trim(),
      grafanaUserId: 0,
      userLogin: '',
      userName: '',
      chatId: input.chatId.trim(),
    };
  }
  return {
    ...input,
    name: selectedUser?.name || selectedUser?.login || input.name,
    grafanaUserId: selectedUser?.id ?? input.grafanaUserId,
    userLogin: selectedUser?.login ?? input.userLogin,
    userName: selectedUser?.name || selectedUser?.login || input.userName,
    chatId: input.chatId.trim(),
  };
}

function severityColor(severity: string) {
  if (severity === 'critical' || severity === 'fatal') {
    return 'red';
  }
  if (severity === 'warning') {
    return 'orange';
  }
  return 'blue';
}

function maskChat(chatId: string) {
  if (chatId.length <= 4) {
    return chatId;
  }
  return `${chatId.slice(0, 2)}***${chatId.slice(-2)}`;
}
