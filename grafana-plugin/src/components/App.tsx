import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { PluginPage } from '@grafana/runtime';
import { CalendarPage } from '../pages/CalendarPage';
import { PeoplePage } from '../pages/PeoplePage';
import { AlertsPage } from '../pages/AlertsPage';
import { ReportsPage } from '../pages/ReportsPage';
import { SettingsPage } from '../pages/SettingsPage';

const nav = [
  { text: 'Calendar', path: '/a/devops-oncall-app/calendar' },
  { text: 'People', path: '/a/devops-oncall-app/people' },
  { text: 'Alerts', path: '/a/devops-oncall-app/alerts' },
  { text: 'Reports', path: '/a/devops-oncall-app/reports' },
  { text: 'Settings', path: '/a/devops-oncall-app/settings' },
];

export function App() {
  return (
    <PluginPage pageNav={{ text: 'OnCall' }} actions={null} subTitle="Schedules, alerts, Jira and duty reports">
      <div className="oncall-shell">
        <nav className="oncall-nav">
          {nav.map((item) => (
            <a key={item.path} className="oncall-nav-link" href={item.path}>
              {item.text}
            </a>
          ))}
        </nav>

        <Routes>
          <Route path="/" element={<Navigate to="/calendar" replace />} />
          <Route path="/calendar" element={<CalendarPage />} />
          <Route path="/people" element={<PeoplePage />} />
          <Route path="/alerts" element={<AlertsPage />} />
          <Route path="/reports" element={<ReportsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </div>
    </PluginPage>
  );
}
