import React, { useEffect, useState } from 'react';
import { Alert, Button, LoadingPlaceholder } from '@grafana/ui';
import { PeriodReport, onCallApi } from '../api';

export function ReportsPage() {
  const [from, setFrom] = useState(() => toDateInput(addDays(new Date(), -7)));
  const [to, setTo] = useState(() => toDateInput(new Date()));
  const [report, setReport] = useState<PeriodReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadReport = async () => {
    setLoading(true);
    setError(null);
    try {
      setReport(await onCallApi.periodReport(from, to));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadReport();
  }, []);

  return (
    <div className="oncall-reports-page">
      <section className="oncall-form-panel oncall-report-config">
        <div>
          <div className="oncall-kicker">Report configurator</div>
          <h3>Duty results by period</h3>
        </div>

        <label>
          From
          <input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        </label>

        <label>
          To
          <input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
        </label>

        <Button type="button" onClick={loadReport} disabled={loading}>
          {loading ? 'Building...' : 'Build report'}
        </Button>
      </section>

      {error && (
        <Alert title="Report failed" severity="error">
          {error}
        </Alert>
      )}

      {loading && <LoadingPlaceholder text="Building duty report" />}

      {report && !loading && (
        <>
          <section className="oncall-report-summary">
            <Metric label="People" value={report.people.length} />
            <Metric label="Shifts" value={report.totals.shiftCount} />
            <Metric label="On-call hours" value={report.totals.onCallHours} />
            <Metric label="Alerts" value={report.totals.alerts} />
            <Metric label="Jira issues" value={report.totals.jiraIssuesCreated} />
          </section>

          <section className="oncall-table-panel">
            <div className="oncall-table-title">
              <h3>
                {report.from} - {report.to}
              </h3>
              <span>Generated {new Date(report.generatedAt).toLocaleString()}</span>
            </div>

            <table className="oncall-table">
              <thead>
                <tr>
                  <th>Person</th>
                  <th>Login</th>
                  <th>Shifts</th>
                  <th>Hours</th>
                  <th>Alerts</th>
                  <th>Jira</th>
                </tr>
              </thead>
              <tbody>
                {report.people.map((person) => (
                  <tr key={person.userId}>
                    <td>{person.userName}</td>
                    <td>{person.userLogin}</td>
                    <td>{person.shiftCount}</td>
                    <td>{person.onCallHours}</td>
                    <td>{person.alerts}</td>
                    <td>{person.jiraIssuesCreated}</td>
                  </tr>
                ))}
                {report.people.length === 0 && (
                  <tr>
                    <td colSpan={6}>No duties found for this period.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </section>
        </>
      )}
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="oncall-metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

const DAY_MS = 24 * 60 * 60 * 1000;

function addDays(date: Date, days: number) {
  return new Date(date.getTime() + days * DAY_MS);
}

function toDateInput(date: Date) {
  return date.toISOString().slice(0, 10);
}
