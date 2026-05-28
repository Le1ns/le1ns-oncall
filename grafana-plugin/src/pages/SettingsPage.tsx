import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Badge, Button, Card, LoadingPlaceholder } from '@grafana/ui';
import {
  CreateNotificationChannelInput,
  CreateNotificationReceiverInput,
  FeatureMeta,
  GrafanaUser,
  JiraSettings,
  NotificationChannel,
  NotificationReceiver,
  onCallApi,
} from '../api';

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
  const [receivers, setReceivers] = useState<NotificationReceiver[]>([]);
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isAdmin, setIsAdmin] = useState(false);
  const [jiraSettings, setJiraSettings] = useState<JiraSettings | null>(null);
  const [receiverForm, setReceiverForm] = useState<CreateNotificationReceiverInput>({
    kind: 'telegram',
    name: '',
    botToken: '',
    webhookUrl: '',
    enabled: true,
  });
  const [routeForm, setRouteForm] = useState<CreateNotificationChannelInput>({
    receiverId: '',
    targetType: 'person',
    name: '',
    grafanaUserId: 0,
    userLogin: '',
    userName: '',
    chatId: '',
    severities: ['critical'],
    enabled: true,
  });

  const selectedReceiver = useMemo(
    () => receivers.find((receiver) => receiver.id === routeForm.receiverId),
    [receivers, routeForm.receiverId]
  );
  const selectedUser = useMemo(
    () => users.find((user) => user.id === Number(routeForm.grafanaUserId)),
    [routeForm.grafanaUserId, users]
  );

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const [metaData, peopleData, receiverData, channelData] = await Promise.all([
        onCallApi.meta(),
        onCallApi.users(),
        onCallApi.notificationReceivers(),
        onCallApi.notificationChannels(),
      ]);
      const [currentUser, jiraData] = await Promise.all([onCallApi.currentGrafanaUser(), onCallApi.jiraSettings()]);
      setMeta(metaData);
      setUsers(peopleData.users);
      setReceivers(receiverData.receivers);
      setChannels(channelData.channels);
      setIsAdmin(currentUser.isGrafanaAdmin);
      setJiraSettings(jiraData);
      setRouteForm((value) => ({
        ...value,
        receiverId: value.receiverId || receiverData.receivers[0]?.id || '',
        grafanaUserId: value.grafanaUserId || peopleData.users[0]?.id || 0,
        userLogin: value.userLogin || peopleData.users[0]?.login || '',
        userName: value.userName || peopleData.users[0]?.name || peopleData.users[0]?.login || '',
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const createReceiver = async (event: React.FormEvent) => {
    event.preventDefault();
    const input = {
      ...receiverForm,
      name: receiverForm.name.trim(),
      botToken: receiverForm.kind === 'telegram' ? receiverForm.botToken.trim() : '',
      webhookUrl: receiverForm.kind === 'lark' ? receiverForm.webhookUrl.trim() : '',
    };
    if (!input.name || (input.kind === 'telegram' && !input.botToken) || (input.kind === 'lark' && !input.webhookUrl)) {
      setError('Receiver name and credentials are required');
      return;
    }

    setSaving(true);
    setError(null);
    try {
      const receiver = await onCallApi.createNotificationReceiver(input);
      setReceiverForm({ kind: 'telegram', name: '', botToken: '', webhookUrl: '', enabled: true });
      setRouteForm((value) => ({ ...value, receiverId: receiver.id }));
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const createChannel = async (event: React.FormEvent) => {
    event.preventDefault();
    const input = normalizeRouteForm(routeForm, selectedReceiver, selectedUser);
    if (!input.receiverId || input.severities.length === 0) {
      setError('Receiver and at least one severity are required');
      return;
    }
    if (selectedReceiver?.kind === 'telegram' && !input.chatId) {
      setError('Telegram chat ID is required');
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
      setRouteForm((value) => ({
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

  const deleteReceiver = async (id: string) => {
    setSaving(true);
    setError(null);
    try {
      await onCallApi.deleteNotificationReceiver(id);
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

  const saveJiraSettings = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!jiraSettings) {
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const payload: JiraSettings = {
        ...jiraSettings,
        baseUrl: jiraSettings.baseUrl.trim(),
        projectKey: jiraSettings.projectKey.trim(),
        issueType: jiraSettings.issueType.trim(),
        summaryTemplate: jiraSettings.summaryTemplate.trim(),
        descriptionTemplate: jiraSettings.descriptionTemplate.trim(),
        labels: jiraSettings.labels.map((item) => item.trim()).filter(Boolean),
      };
      const updated = await onCallApi.updateJiraSettings(payload);
      setJiraSettings(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const setSelectedReceiver = (receiverId: string) => {
    const receiver = receivers.find((candidate) => candidate.id === receiverId);
    setRouteForm({
      ...routeForm,
      receiverId,
      targetType: receiver?.kind === 'lark' ? 'group' : routeForm.targetType,
      chatId: receiver?.kind === 'lark' ? '' : routeForm.chatId,
    });
  };

  const toggleSeverity = (severity: string) => {
    const severities = routeForm.severities.includes(severity)
      ? routeForm.severities.filter((item) => item !== severity)
      : [...routeForm.severities, severity];
    setRouteForm({ ...routeForm, severities });
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

      {jiraSettings && isAdmin && (
        <details className="oncall-collapse">
          <summary>
            <span>Jira integration</span>
          </summary>
          <form className="oncall-form-panel oncall-receiver-form" onSubmit={saveJiraSettings}>
            <div>
              <div className="oncall-kicker">Jira integration</div>
              <h3>Issue creation</h3>
            </div>
            <label className="oncall-toggle-row">
              <input
                type="checkbox"
                checked={jiraSettings.enabled}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, enabled: event.target.checked })}
              />
              <span>Enable Jira integration</span>
            </label>
            <label>
              Jira URL
              <input
                value={jiraSettings.baseUrl}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, baseUrl: event.target.value })}
              />
            </label>
            <label>
              Project key
              <input
                value={jiraSettings.projectKey}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, projectKey: event.target.value })}
              />
            </label>
            <label>
              Issue type
              <input
                value={jiraSettings.issueType}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, issueType: event.target.value })}
              />
            </label>
            <label className="oncall-toggle-row">
              <input
                type="checkbox"
                checked={jiraSettings.autoCreateOnFiring}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, autoCreateOnFiring: event.target.checked })}
              />
              <span>Auto-create issue on firing alert</span>
            </label>
            <label>
              Summary template
              <input
                value={jiraSettings.summaryTemplate}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, summaryTemplate: event.target.value })}
              />
            </label>
            <label>
              Description template
              <textarea
                value={jiraSettings.descriptionTemplate}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, descriptionTemplate: event.target.value })}
              />
            </label>
            <label>
              Labels (comma separated)
              <input
                value={jiraSettings.labels.join(',')}
                disabled={!isAdmin}
                onChange={(event) => setJiraSettings({ ...jiraSettings, labels: event.target.value.split(',') })}
              />
            </label>
            <Button type="submit" disabled={saving || !isAdmin}>
              {saving ? 'Saving...' : 'Save Jira settings'}
            </Button>
          </form>
        </details>
      )}

      <details className="oncall-collapse">
        <summary>
          <span>Receivers</span>
        </summary>
        <form className="oncall-form-panel oncall-receiver-form oncall-collapse-block" onSubmit={createReceiver}>
          <div>
            <div className="oncall-kicker">Add receiver</div>
            <h3>Delivery credentials</h3>
          </div>

        <label>
          Provider
          <select
            value={receiverForm.kind}
            onChange={(event) =>
              setReceiverForm({ ...receiverForm, kind: event.target.value as 'telegram' | 'lark' })
            }
          >
            <option value="telegram">Telegram</option>
            <option value="lark">Lark</option>
          </select>
        </label>

        <label>
          Name
          <input value={receiverForm.name} onChange={(event) => setReceiverForm({ ...receiverForm, name: event.target.value })} />
        </label>

        {receiverForm.kind === 'telegram' ? (
          <label>
            Bot token
            <input
              type="password"
              value={receiverForm.botToken}
              onChange={(event) => setReceiverForm({ ...receiverForm, botToken: event.target.value })}
            />
          </label>
        ) : (
          <label>
            Webhook URL
            <input
              type="password"
              value={receiverForm.webhookUrl}
              onChange={(event) => setReceiverForm({ ...receiverForm, webhookUrl: event.target.value })}
            />
          </label>
        )}

          <Button type="submit" disabled={saving}>
            {saving ? 'Adding...' : 'Add receiver'}
          </Button>
        </form>
        <section className="oncall-table-panel oncall-collapse-block">
          <div className="oncall-table-title">
            <h3>Receivers</h3>
            <span>{receivers.length} configured</span>
          </div>
          <table className="oncall-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Provider</th>
              <th>Credentials</th>
              <th>Status</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {receivers.map((receiver) => (
              <tr key={receiver.id}>
                <td>
                  <strong>{receiver.name}</strong>
                </td>
                <td>{receiver.kind}</td>
                <td>{receiver.hasSecret ? 'configured' : 'missing'}</td>
                <td>
                  <Badge text={receiver.enabled ? 'enabled' : 'disabled'} color={receiver.enabled ? 'green' : 'red'} />
                </td>
                <td>
                  <Button variant="destructive" size="sm" disabled={saving} onClick={() => deleteReceiver(receiver.id)}>
                    Delete
                  </Button>
                </td>
              </tr>
            ))}
            {receivers.length === 0 && (
              <tr>
                <td colSpan={5}>No receivers configured.</td>
              </tr>
            )}
          </tbody>
          </table>
        </section>
      </details>

      <details className="oncall-collapse">
        <summary>
          <span>Notifications</span>
        </summary>
        <form className="oncall-form-panel oncall-notification-form oncall-collapse-block" onSubmit={createChannel}>
          <div>
            <div className="oncall-kicker">Add notification</div>
            <h3>Alert routing</h3>
          </div>

        <label>
          Receiver
          <select value={routeForm.receiverId} onChange={(event) => setSelectedReceiver(event.target.value)}>
            <option value="">Select receiver</option>
            {receivers.map((receiver) => (
              <option key={receiver.id} value={receiver.id}>
                {receiver.name} ({receiver.kind})
              </option>
            ))}
          </select>
        </label>

        <label>
          Type
          <select
            value={routeForm.targetType}
            disabled={selectedReceiver?.kind === 'lark'}
            onChange={(event) =>
              setRouteForm({ ...routeForm, targetType: event.target.value as 'person' | 'group', name: '' })
            }
          >
            <option value="person">Person</option>
            <option value="group">Group</option>
          </select>
        </label>

        {routeForm.targetType === 'person' ? (
          <label>
            Person
            <select
              value={routeForm.grafanaUserId}
              onChange={(event) => {
                const user = users.find((candidate) => candidate.id === Number(event.target.value));
                setRouteForm({
                  ...routeForm,
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
            <input value={routeForm.name} onChange={(event) => setRouteForm({ ...routeForm, name: event.target.value })} />
          </label>
        )}

        {selectedReceiver?.kind === 'telegram' && (
          <label>
            Chat ID
            <input value={routeForm.chatId} onChange={(event) => setRouteForm({ ...routeForm, chatId: event.target.value })} />
          </label>
        )}

        <fieldset className="oncall-severity-field">
          <legend>Severity</legend>
          <div>
            {severityOptions.map((severity) => (
              <label key={severity}>
                <input
                  type="checkbox"
                  checked={routeForm.severities.includes(severity)}
                  onChange={() => toggleSeverity(severity)}
                />
                <span>{severity}</span>
              </label>
            ))}
          </div>
        </fieldset>

          <Button type="submit" disabled={saving || receivers.length === 0}>
            {saving ? 'Adding...' : 'Add notification'}
          </Button>
        </form>
        <section className="oncall-table-panel oncall-collapse-block">
          <div className="oncall-table-title">
            <h3>Notification routing</h3>
            <span>{channels.length} rules</span>
          </div>
          <table className="oncall-table">
          <thead>
            <tr>
              <th>Target</th>
              <th>Receiver</th>
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
                <td>
                  <strong>{channel.receiverName || channel.kind}</strong>
                  <div className="oncall-muted">{channel.kind}</div>
                </td>
                <td>
                  <div className="oncall-badge-row">
                    {channel.severities.map((severity) => (
                      <Badge key={severity} text={severity} color={severityColor(severity)} />
                    ))}
                  </div>
                </td>
                <td>{channel.kind === 'telegram' ? maskChat(channel.chatId) : '-'}</td>
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
                <td colSpan={6}>No notification rules configured.</td>
              </tr>
            )}
          </tbody>
          </table>
        </section>
      </details>
    </div>
  );
}

function normalizeRouteForm(
  input: CreateNotificationChannelInput,
  selectedReceiver?: NotificationReceiver,
  selectedUser?: GrafanaUser
): CreateNotificationChannelInput {
  if (input.targetType === 'group') {
    return {
      ...input,
      kind: selectedReceiver?.kind,
      name: input.name.trim(),
      grafanaUserId: 0,
      userLogin: '',
      userName: '',
      chatId: selectedReceiver?.kind === 'telegram' ? input.chatId.trim() : '',
    };
  }
  return {
    ...input,
    kind: selectedReceiver?.kind,
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
