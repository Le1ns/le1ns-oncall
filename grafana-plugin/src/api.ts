import { getBackendSrv } from '@grafana/runtime';

const API_BASE = 'http://localhost:8080';

export interface HealthResponse {
  status: string;
  time: string;
}

export interface FeatureMeta {
  service: string;
  features: Record<string, boolean>;
}

export interface Schedule {
  id: string;
  name: string;
  rotation: string;
  timezone: string;
  description: string;
}

export interface Shift {
  id: string;
  scheduleId: string;
  scheduleName: string;
  userId: number;
  userLogin: string;
  userName: string;
  startsAt: string;
  endsAt: string;
  status: string;
  layer: number;
}

export interface GrafanaUser {
  id: number;
  login: string;
  name: string;
  email: string;
  role: string;
}

export interface DailyReport {
  date: string;
  primaryOnCall: string;
  alertsTotal: number;
  jiraIssuesCreated: number;
  unresolvedAlerts: number;
}

export interface CreateShiftInput {
  scheduleId: string;
  userId: number;
  userLogin: string;
  userName: string;
  startsAt: string;
  endsAt: string;
  layer: number;
}

export interface PeriodReport {
  from: string;
  to: string;
  generatedAt: string;
  people: PersonReport[];
  totals: {
    shiftCount: number;
    onCallHours: number;
    alerts: number;
    jiraIssuesCreated: number;
  };
}

export interface AlertmanagerAlert {
  id: string;
  fingerprint: string;
  groupKey: string;
  status: string;
  severity: string;
  alertName: string;
  service: string;
  summary: string;
  startsAt?: string;
  endsAt?: string;
  assignedUserId: number;
  assignedUserLogin: string;
  assignedUserName: string;
  createdAt: string;
  updatedAt: string;
  jiraIssueKey?: string;
  jiraIssueUrl?: string;
}
export interface JiraSettings {
  enabled: boolean;
  baseUrl: string;
  projectKey: string;
  issueType: string;
  autoCreateOnFiring: boolean;
  summaryTemplate: string;
  descriptionTemplate: string;
  labels: string[];
  updatedAt: string;
}
export interface CurrentGrafanaUser {
  login: string;
  email: string;
  isGrafanaAdmin: boolean;
}

export interface PersonReport {
  userId: number;
  userLogin: string;
  userName: string;
  shiftCount: number;
  onCallHours: number;
  alerts: number;
  jiraIssuesCreated: number;
}

export interface NotificationChannel {
  id: string;
  kind: string;
  receiverId: string;
  receiverName: string;
  targetType: 'person' | 'group';
  name: string;
  grafanaUserId: number;
  userLogin: string;
  userName: string;
  chatId: string;
  severities: string[];
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateNotificationChannelInput {
  kind?: string;
  receiverId: string;
  targetType: 'person' | 'group';
  name: string;
  grafanaUserId: number;
  userLogin: string;
  userName: string;
  chatId: string;
  severities: string[];
  enabled: boolean;
}

export interface NotificationReceiver {
  id: string;
  kind: 'telegram' | 'lark';
  name: string;
  hasSecret: boolean;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateNotificationReceiverInput {
  kind: 'telegram' | 'lark';
  name: string;
  botToken: string;
  webhookUrl: string;
  enabled: boolean;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  });

  if (!response.ok) {
    throw new Error(`API request failed: ${response.status}`);
  }

  return response.json() as Promise<T>;
}

export const onCallApi = {
  health: () => request<HealthResponse>('/health'),
  meta: () => request<FeatureMeta>('/api/v1/meta'),
  schedules: () => request<{ schedules: Schedule[] }>('/api/v1/schedules'),
  calendar: (from?: string, to?: string) =>
    request<{ range: { from: string; to: string }; shifts: Shift[] }>(withPeriod('/api/v1/calendar', from, to)),
  createShift: (input: CreateShiftInput) =>
    request<Shift>('/api/v1/shifts', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  users: () => request<{ users: GrafanaUser[]; source: string; note?: string }>('/api/v1/grafana/users'),
  alerts: (from: string, to: string, limit = 200) =>
    request<{ range: { from: string; to: string }; alerts: AlertmanagerAlert[] }>(
      withPeriod(`/api/v1/alerts?limit=${encodeURIComponent(String(limit))}`, from, to)
    ),
  createAlertJiraIssue: (alertId: string) =>
    request<{ issue: { issueKey: string; issueUrl: string }; existing: boolean }>(
      `/api/v1/alerts/${encodeURIComponent(alertId)}/jira-issue`,
      { method: 'POST' }
    ),
  jiraSettings: () => request<JiraSettings>('/api/v1/integrations/jira/settings'),
  updateJiraSettings: (input: JiraSettings) =>
    request<JiraSettings>('/api/v1/integrations/jira/settings', {
      method: 'PUT',
      body: JSON.stringify(input),
    }),
  notificationReceivers: () => request<{ receivers: NotificationReceiver[] }>('/api/v1/notification-receivers'),
  createNotificationReceiver: (input: CreateNotificationReceiverInput) =>
    request<NotificationReceiver>('/api/v1/notification-receivers', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  deleteNotificationReceiver: (id: string) =>
    request<{ status: string }>(`/api/v1/notification-receivers/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  notificationChannels: () => request<{ channels: NotificationChannel[] }>('/api/v1/notification-channels'),
  createNotificationChannel: (input: CreateNotificationChannelInput) =>
    request<NotificationChannel>('/api/v1/notification-channels', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  deleteNotificationChannel: (id: string) =>
    request<{ status: string }>(`/api/v1/notification-channels/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  dailyReport: () => request<DailyReport>('/api/v1/reports/daily'),
  periodReport: (from: string, to: string) => request<PeriodReport>(withPeriod('/api/v1/reports/period', from, to)),
  currentGrafanaUser: async () => {
    const user = await getBackendSrv().get('/api/user');
    return {
      login: user.login ?? '',
      email: user.email ?? '',
      isGrafanaAdmin: Boolean(user.isGrafanaAdmin),
    } as CurrentGrafanaUser;
  },
};

function withPeriod(path: string, from?: string, to?: string) {
  const [base, existing] = path.split('?', 2);
  const query = new URLSearchParams(existing ?? '');
  if (from) {
    query.set('from', from);
  }
  if (to) {
    query.set('to', to);
  }
  const suffix = query.toString();
  return suffix ? `${base}?${suffix}` : base;
}
