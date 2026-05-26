import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Badge, Button, LoadingPlaceholder } from '@grafana/ui';
import { CreateShiftInput, GrafanaUser, Schedule, Shift, onCallApi } from '../api';

const DAY_MS = 24 * 60 * 60 * 1000;
const HOUR_MARKS = ['00:00', '03:00', '06:00', '09:00', '12:00', '15:00', '18:00', '21:00', '24:00'];
const COLORS = ['#5794F2', '#B877D9', '#73BF69', '#FF9830', '#F2495C', '#FADE2A'];

export function CalendarPage() {
  const [weekStart, setWeekStart] = useState(() => startOfWeek(new Date()));
  const [users, setUsers] = useState<GrafanaUser[]>([]);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [shifts, setShifts] = useState<Shift[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const weekEnd = useMemo(() => addDays(weekStart, 6), [weekStart]);
  const days = useMemo(() => Array.from({ length: 7 }, (_, index) => addDays(weekStart, index)), [weekStart]);

  const [form, setForm] = useState({
    userId: '',
    date: toDateInput(new Date()),
    startsAt: '09:00',
    endsAt: '09:00',
    layer: '0',
  });

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const [calendar, people, scheduleData] = await Promise.all([
        onCallApi.calendar(toDateInput(weekStart), toDateInput(weekEnd)),
        onCallApi.users(),
        onCallApi.schedules(),
      ]);
      setShifts(calendar.shifts);
      setUsers(people.users);
      setSchedules(scheduleData.schedules);
      if (!form.userId && people.users[0]) {
        setForm((value) => ({ ...value, userId: String(people.users[0].id) }));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [weekStart]);

  const createShift = async (event: React.FormEvent) => {
    event.preventDefault();
    const user = users.find((candidate) => String(candidate.id) === form.userId);
    if (!user) {
      setError('Select Grafana user');
      return;
    }

    const startsAt = localDateTime(form.date, form.startsAt);
    let endsAt = localDateTime(form.date, form.endsAt);
    if (endsAt <= startsAt) {
      endsAt = new Date(endsAt.getTime() + DAY_MS);
    }

    const input: CreateShiftInput = {
      scheduleId: schedules[0]?.id ?? 'primary-devops',
      userId: user.id,
      userLogin: user.login,
      userName: user.name || user.login,
      startsAt: startsAt.toISOString(),
      endsAt: endsAt.toISOString(),
      layer: Number(form.layer),
    };

    setSaving(true);
    setError(null);
    try {
      await onCallApi.createShift(input);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const currentLine = currentTimeLine(weekStart);

  return (
    <div className="oncall-calendar-page">
      <section className="oncall-toolbar">
        <div>
          <div className="oncall-kicker">Final schedule</div>
          <h2>{formatRange(weekStart, weekEnd)}</h2>
        </div>
        <div className="oncall-toolbar-actions">
          <span className="oncall-timezone">Current timezone: browser local</span>
          <Button variant="secondary" onClick={() => setWeekStart(addDays(weekStart, -7))}>
            Previous
          </Button>
          <Button variant="secondary" onClick={() => setWeekStart(startOfWeek(new Date()))}>
            Today
          </Button>
          <Button variant="secondary" onClick={() => setWeekStart(addDays(weekStart, 7))}>
            Next
          </Button>
        </div>
      </section>

      {error && (
        <Alert title="OnCall action failed" severity="error">
          {error}
        </Alert>
      )}

      <section className="oncall-schedule-band">
        <div className="oncall-band-title">Schedule team and timezones</div>
        <div className="oncall-avatar-row">
          {users.slice(0, 10).map((user, index) => (
            <span key={user.id} className="oncall-avatar" style={{ background: COLORS[index % COLORS.length] }}>
              {initials(user.name || user.login)}
            </span>
          ))}
        </div>
        <div className="oncall-hour-row">
          {HOUR_MARKS.map((mark) => (
            <span key={mark}>{mark}</span>
          ))}
        </div>
      </section>

      <section className="oncall-board">
        <div className="oncall-board-header">
          {days.map((day) => (
            <div key={day.toISOString()} className="oncall-day-header">
              <span>{weekday(day)}</span>
              <strong>{day.getDate()} {monthShort(day)}</strong>
            </div>
          ))}
        </div>

        <div className="oncall-timeline">
          {currentLine && <div className="oncall-now-line" style={{ left: `${currentLine}%` }} />}
          {loading ? (
            <LoadingPlaceholder text="Loading on-call schedule" />
          ) : (
            days.map((day) => (
              <div key={day.toISOString()} className="oncall-day-column">
                {segmentsForDay(shifts, day).map((segment) => (
                  <div
                    key={segment.key}
                    className={`oncall-shift oncall-shift-${segment.status}`}
                    style={{
                      background: COLORS[segment.colorIndex % COLORS.length],
                      left: `${segment.left}%`,
                      top: 18 + segment.layer * 36,
                      width: `${segment.width}%`,
                    }}
                    title={`${segment.userName}: ${formatTime(segment.startsAt)} - ${formatTime(segment.endsAt)}`}
                  >
                    <span>{segment.userName}</span>
                  </div>
                ))}
              </div>
            ))
          )}
        </div>
      </section>

      <form className="oncall-form-panel" onSubmit={createShift}>
        <div>
          <div className="oncall-kicker">Manual assignment</div>
          <h3>Add Grafana user to duty</h3>
        </div>

        <label>
          Person
          <select value={form.userId} onChange={(event) => setForm({ ...form, userId: event.target.value })}>
            {users.map((user) => (
              <option key={user.id} value={user.id}>
                {user.name || user.login} ({user.login})
              </option>
            ))}
          </select>
        </label>

        <label>
          Date
          <input type="date" value={form.date} onChange={(event) => setForm({ ...form, date: event.target.value })} />
        </label>

        <label>
          Start
          <input
            type="time"
            value={form.startsAt}
            onChange={(event) => setForm({ ...form, startsAt: event.target.value })}
          />
        </label>

        <label>
          End
          <input type="time" value={form.endsAt} onChange={(event) => setForm({ ...form, endsAt: event.target.value })} />
        </label>

        <label>
          Layer
          <select value={form.layer} onChange={(event) => setForm({ ...form, layer: event.target.value })}>
            <option value="0">Primary</option>
            <option value="1">Secondary</option>
            <option value="2">Escalation</option>
          </select>
        </label>

        <Button type="submit" disabled={saving || users.length === 0}>
          {saving ? 'Adding...' : '+ Add duty'}
        </Button>
      </form>

      <section className="oncall-legend">
        <Badge text="active" color="green" />
        <Badge text="planned" color="blue" />
        <Badge text="finished" color="purple" />
      </section>
    </div>
  );
}

interface Segment {
  key: string;
  userName: string;
  startsAt: Date;
  endsAt: Date;
  left: number;
  width: number;
  layer: number;
  status: string;
  colorIndex: number;
}

function segmentsForDay(shifts: Shift[], day: Date): Segment[] {
  const start = startOfDay(day);
  const end = addDays(start, 1);

  return shifts.flatMap((shift, index) => {
    const startsAt = new Date(shift.startsAt);
    const endsAt = new Date(shift.endsAt);
    if (endsAt <= start || startsAt >= end) {
      return [];
    }

    const segmentStart = startsAt > start ? startsAt : start;
    const segmentEnd = endsAt < end ? endsAt : end;
    const left = ((segmentStart.getTime() - start.getTime()) / DAY_MS) * 100;
    const width = Math.max(((segmentEnd.getTime() - segmentStart.getTime()) / DAY_MS) * 100, 6);

    return [
      {
        key: `${shift.id}-${day.toISOString()}`,
        userName: shift.userName,
        startsAt,
        endsAt,
        left,
        width: Math.min(width, 100 - left),
        layer: shift.layer ?? 0,
        status: shift.status,
        colorIndex: index,
      },
    ];
  });
}

function currentTimeLine(weekStart: Date) {
  const now = new Date();
  const start = startOfDay(weekStart);
  const end = addDays(start, 7);
  if (now < start || now > end) {
    return null;
  }
  return ((now.getTime() - start.getTime()) / (7 * DAY_MS)) * 100;
}

function startOfWeek(date: Date) {
  const value = startOfDay(date);
  const day = value.getDay();
  const diff = day === 0 ? -6 : 1 - day;
  return addDays(value, diff);
}

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function addDays(date: Date, days: number) {
  return new Date(date.getTime() + days * DAY_MS);
}

function localDateTime(date: string, time: string) {
  return new Date(`${date}T${time}:00`);
}

function toDateInput(date: Date) {
  return date.toISOString().slice(0, 10);
}

function formatRange(from: Date, to: Date) {
  return `${from.getDate()} ${monthShort(from)} - ${to.getDate()} ${monthShort(to)}`;
}

function weekday(date: Date) {
  return new Intl.DateTimeFormat(undefined, { weekday: 'short' }).format(date);
}

function monthShort(date: Date) {
  return new Intl.DateTimeFormat(undefined, { month: 'short' }).format(date);
}

function formatTime(date: Date) {
  return new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' }).format(date);
}

function initials(value: string) {
  return value
    .split(/[\s._-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}
