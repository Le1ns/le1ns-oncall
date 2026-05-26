package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GrafanaUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Schedule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Rotation    string `json:"rotation"`
	Timezone    string `json:"timezone"`
	Description string `json:"description"`
}

type Shift struct {
	ID           string    `json:"id"`
	ScheduleID   string    `json:"scheduleId"`
	ScheduleName string    `json:"scheduleName"`
	UserID       int64     `json:"userId"`
	UserLogin    string    `json:"userLogin"`
	UserName     string    `json:"userName"`
	StartsAt     time.Time `json:"startsAt"`
	EndsAt       time.Time `json:"endsAt"`
	Status       string    `json:"status"`
	Layer        int       `json:"layer"`
}

type Alert struct {
	ID                string          `json:"id"`
	Fingerprint       string          `json:"fingerprint"`
	GroupKey          string          `json:"groupKey"`
	Status            string          `json:"status"`
	Severity          string          `json:"severity"`
	AlertName         string          `json:"alertName"`
	Service           string          `json:"service"`
	Summary           string          `json:"summary"`
	StartsAt          *time.Time      `json:"startsAt,omitempty"`
	EndsAt            *time.Time      `json:"endsAt,omitempty"`
	AssignedUserID    int64           `json:"assignedUserId"`
	AssignedUserLogin string          `json:"assignedUserLogin"`
	AssignedUserName  string          `json:"assignedUserName"`
	Payload           json.RawMessage `json:"payload,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

type CreateShift struct {
	ScheduleID string    `json:"scheduleId"`
	UserID     int64     `json:"userId"`
	UserLogin  string    `json:"userLogin"`
	UserName   string    `json:"userName"`
	StartsAt   time.Time `json:"startsAt"`
	EndsAt     time.Time `json:"endsAt"`
	Layer      int       `json:"layer"`
}

type PeriodReport struct {
	From      string         `json:"from"`
	To        string         `json:"to"`
	People    []PersonReport `json:"people"`
	Totals    ReportTotals   `json:"totals"`
	Generated string         `json:"generatedAt"`
}

type PersonReport struct {
	UserID            int64   `json:"userId"`
	UserLogin         string  `json:"userLogin"`
	UserName          string  `json:"userName"`
	ShiftCount        int     `json:"shiftCount"`
	OnCallHours       float64 `json:"onCallHours"`
	Alerts            int     `json:"alerts"`
	JiraIssuesCreated int     `json:"jiraIssuesCreated"`
}

type ReportTotals struct {
	ShiftCount        int     `json:"shiftCount"`
	OnCallHours       float64 `json:"onCallHours"`
	Alerts            int     `json:"alerts"`
	JiraIssuesCreated int     `json:"jiraIssuesCreated"`
}

type Store struct {
	db        *pgxpool.Pool
	mu        sync.Mutex
	schedules []Schedule
	shifts    []Shift
	alerts    []Alert
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	s := newMemoryStore()
	if databaseURL == "" {
		return s, nil
	}

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}

	s.db = db
	if err := s.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.seed(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func newMemoryStore() *Store {
	now := startOfDay(time.Now())
	schedule := Schedule{
		ID:          "primary-devops",
		Name:        "Primary DevOps",
		Rotation:    "manual",
		Timezone:    "Europe/Moscow",
		Description: "Main production on-call schedule",
	}

	return &Store{
		schedules: []Schedule{schedule},
		shifts: []Shift{
			{
				ID:           "shift-current",
				ScheduleID:   schedule.ID,
				ScheduleName: schedule.Name,
				UserID:       1,
				UserLogin:    "admin",
				UserName:     "Grafana Admin",
				StartsAt:     now,
				EndsAt:       now.Add(24 * time.Hour),
				Status:       "active",
				Layer:        0,
			},
		},
	}
}

func (s *Store) Schedules() []Schedule {
	if s.db != nil {
		return s.pgSchedules()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Schedule, len(s.schedules))
	copy(result, s.schedules)
	return result
}

func (s *Store) Shifts(from, to time.Time) []Shift {
	if s.db != nil {
		return s.pgShifts(from, to)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Shift, 0, len(s.shifts))
	for _, shift := range s.shifts {
		shift.Status = statusFor(shift.StartsAt, shift.EndsAt)
		if shift.EndsAt.After(from) && shift.StartsAt.Before(to) {
			result = append(result, shift)
		}
	}
	sortShifts(result)
	return result
}

func (s *Store) CreateShift(input CreateShift) (Shift, error) {
	if !input.EndsAt.After(input.StartsAt) {
		return Shift{}, fmt.Errorf("endsAt must be after startsAt")
	}
	if input.UserID == 0 || input.UserLogin == "" {
		return Shift{}, fmt.Errorf("grafana user is required")
	}
	if s.db != nil {
		return s.pgCreateShift(input)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	schedule := s.schedules[0]
	for _, candidate := range s.schedules {
		if candidate.ID == input.ScheduleID {
			schedule = candidate
			break
		}
	}

	shift := buildShift(schedule, input)
	s.shifts = append(s.shifts, shift)
	return shift, nil
}

func (s *Store) Alerts(limit int) []Alert {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if s.db != nil {
		return s.pgAlerts(limit)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Alert, len(s.alerts))
	copy(result, s.alerts)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if len(result) > limit {
		return result[:limit]
	}
	return result
}

func (s *Store) SaveAlertmanagerPayload(payload map[string]any) (int, error) {
	alertItems, ok := payload["alerts"].([]any)
	if !ok {
		return 0, nil
	}

	groupKey, _ := payload["groupKey"].(string)
	saved := 0
	for _, item := range alertItems {
		alertMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		alert := parseAlert(alertMap, groupKey)
		if alert.Fingerprint == "" {
			alert.Fingerprint = fmt.Sprintf("alert-%d", time.Now().UnixNano())
		}
		if alert.ID == "" {
			alert.ID = alert.Fingerprint
		}
		if alert.Payload == nil {
			encoded, _ := json.Marshal(alertMap)
			alert.Payload = encoded
		}

		if s.db != nil {
			if err := s.pgUpsertAlert(alert); err != nil {
				return saved, err
			}
		} else {
			s.memoryUpsertAlert(alert)
		}
		saved++
	}
	return saved, nil
}

func (s *Store) Report(from, to time.Time) PeriodReport {
	shifts := s.Shifts(from, to)
	alerts := s.Alerts(200)
	people := map[int64]*PersonReport{}
	totals := ReportTotals{}

	for _, shift := range shifts {
		overlapStart := maxTime(shift.StartsAt, from)
		overlapEnd := minTime(shift.EndsAt, to)
		hours := overlapEnd.Sub(overlapStart).Hours()
		if hours < 0 {
			hours = 0
		}

		person, ok := people[shift.UserID]
		if !ok {
			person = &PersonReport{
				UserID:    shift.UserID,
				UserLogin: shift.UserLogin,
				UserName:  shift.UserName,
			}
			people[shift.UserID] = person
		}

		person.ShiftCount++
		person.OnCallHours += hours
		totals.ShiftCount++
		totals.OnCallHours += hours
	}

	for _, alert := range alerts {
		alertTime := alert.CreatedAt
		if alert.StartsAt != nil {
			alertTime = *alert.StartsAt
		}
		if alertTime.Before(from) || !alertTime.Before(to) {
			continue
		}
		totals.Alerts++
		if alert.AssignedUserID == 0 {
			continue
		}
		person, ok := people[alert.AssignedUserID]
		if !ok {
			person = &PersonReport{
				UserID:    alert.AssignedUserID,
				UserLogin: alert.AssignedUserLogin,
				UserName:  alert.AssignedUserName,
			}
			people[alert.AssignedUserID] = person
		}
		person.Alerts++
	}

	result := make([]PersonReport, 0, len(people))
	for _, person := range people {
		person.OnCallHours = round1(person.OnCallHours)
		result = append(result, *person)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].OnCallHours == result[j].OnCallHours {
			return result[i].UserLogin < result[j].UserLogin
		}
		return result[i].OnCallHours > result[j].OnCallHours
	})

	totals.OnCallHours = round1(totals.OnCallHours)
	return PeriodReport{
		From:      from.Format(time.DateOnly),
		To:        to.Add(-time.Nanosecond).Format(time.DateOnly),
		People:    result,
		Totals:    totals,
		Generated: time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *Store) ensureSchema(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS oncall_schedules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  rotation TEXT NOT NULL DEFAULT 'manual',
  timezone TEXT NOT NULL DEFAULT 'UTC',
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oncall_shifts (
  id TEXT PRIMARY KEY,
  schedule_id TEXT NOT NULL REFERENCES oncall_schedules(id) ON DELETE CASCADE,
  grafana_user_id BIGINT NOT NULL,
  user_login TEXT NOT NULL,
  user_name TEXT NOT NULL,
  starts_at TIMESTAMPTZ NOT NULL,
  ends_at TIMESTAMPTZ NOT NULL,
  layer INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS oncall_shifts_time_idx ON oncall_shifts(starts_at, ends_at);
CREATE INDEX IF NOT EXISTS oncall_shifts_schedule_time_idx ON oncall_shifts(schedule_id, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS oncall_alerts (
  id TEXT PRIMARY KEY,
  fingerprint TEXT NOT NULL UNIQUE,
  group_key TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT '',
  alert_name TEXT NOT NULL DEFAULT '',
  service TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  assigned_grafana_user_id BIGINT NOT NULL DEFAULT 0,
  assigned_user_login TEXT NOT NULL DEFAULT '',
  assigned_user_name TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS oncall_alerts_created_at_idx ON oncall_alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS oncall_alerts_status_idx ON oncall_alerts(status);
`)
	return err
}

func (s *Store) seed(ctx context.Context) error {
	schedule := s.schedules[0]
	_, err := s.db.Exec(ctx, `
INSERT INTO oncall_schedules (id, name, rotation, timezone, description)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO NOTHING
`, schedule.ID, schedule.Name, schedule.Rotation, schedule.Timezone, schedule.Description)
	if err != nil {
		return err
	}

	var exists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM oncall_shifts)`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	shift := s.shifts[0]
	_, err = s.db.Exec(ctx, `
INSERT INTO oncall_shifts (id, schedule_id, grafana_user_id, user_login, user_name, starts_at, ends_at, layer)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, shift.ID, shift.ScheduleID, shift.UserID, shift.UserLogin, shift.UserName, shift.StartsAt, shift.EndsAt, shift.Layer)
	return err
}

func (s *Store) pgSchedules() []Schedule {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
SELECT id, name, rotation, timezone, description
FROM oncall_schedules
ORDER BY name
`)
	if err != nil {
		return s.memorySchedules()
	}
	defer rows.Close()

	schedules := []Schedule{}
	for rows.Next() {
		var schedule Schedule
		if err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&schedule.Rotation,
			&schedule.Timezone,
			&schedule.Description,
		); err != nil {
			return s.memorySchedules()
		}
		schedules = append(schedules, schedule)
	}
	return schedules
}

func (s *Store) pgShifts(from, to time.Time) []Shift {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
SELECT
  sh.id,
  sh.schedule_id,
  sc.name AS schedule_name,
  sh.grafana_user_id AS user_id,
  sh.user_login,
  sh.user_name,
  sh.starts_at,
  sh.ends_at,
  sh.layer
FROM oncall_shifts sh
JOIN oncall_schedules sc ON sc.id = sh.schedule_id
WHERE sh.ends_at > $1 AND sh.starts_at < $2
ORDER BY sh.starts_at
`, from, to)
	if err != nil {
		return nil
	}
	defer rows.Close()

	shifts := []Shift{}
	for rows.Next() {
		var shift Shift
		if err := rows.Scan(
			&shift.ID,
			&shift.ScheduleID,
			&shift.ScheduleName,
			&shift.UserID,
			&shift.UserLogin,
			&shift.UserName,
			&shift.StartsAt,
			&shift.EndsAt,
			&shift.Layer,
		); err != nil {
			return nil
		}
		shift.Status = statusFor(shift.StartsAt, shift.EndsAt)
		shifts = append(shifts, shift)
	}
	return shifts
}

func (s *Store) pgCreateShift(input CreateShift) (Shift, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	schedule := Schedule{}
	err := s.db.QueryRow(ctx, `
SELECT id, name, rotation, timezone, description
FROM oncall_schedules
WHERE id = $1
`, input.ScheduleID).Scan(&schedule.ID, &schedule.Name, &schedule.Rotation, &schedule.Timezone, &schedule.Description)
	if err != nil {
		return Shift{}, err
	}

	shift := buildShift(schedule, input)
	_, err = s.db.Exec(ctx, `
INSERT INTO oncall_shifts (id, schedule_id, grafana_user_id, user_login, user_name, starts_at, ends_at, layer)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, shift.ID, shift.ScheduleID, shift.UserID, shift.UserLogin, shift.UserName, shift.StartsAt, shift.EndsAt, shift.Layer)
	if err != nil {
		return Shift{}, err
	}
	return shift, nil
}

func (s *Store) pgAlerts(limit int) []Alert {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
SELECT
  id,
  fingerprint,
  group_key,
  status,
  severity,
  alert_name,
  service,
  summary,
  starts_at,
  ends_at,
  assigned_grafana_user_id,
  assigned_user_login,
  assigned_user_name,
  payload,
  created_at,
  updated_at
FROM oncall_alerts
ORDER BY created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	alerts := []Alert{}
	for rows.Next() {
		var alert Alert
		if err := rows.Scan(
			&alert.ID,
			&alert.Fingerprint,
			&alert.GroupKey,
			&alert.Status,
			&alert.Severity,
			&alert.AlertName,
			&alert.Service,
			&alert.Summary,
			&alert.StartsAt,
			&alert.EndsAt,
			&alert.AssignedUserID,
			&alert.AssignedUserLogin,
			&alert.AssignedUserName,
			&alert.Payload,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		); err != nil {
			return nil
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func (s *Store) pgUpsertAlert(alert Alert) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assignAlertToShift(ctx, s.db, &alert)
	_, err := s.db.Exec(ctx, `
INSERT INTO oncall_alerts (
  id,
  fingerprint,
  group_key,
  status,
  severity,
  alert_name,
  service,
  summary,
  starts_at,
  ends_at,
  assigned_grafana_user_id,
  assigned_user_login,
  assigned_user_name,
  payload
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (fingerprint) DO UPDATE SET
  group_key = EXCLUDED.group_key,
  status = EXCLUDED.status,
  severity = EXCLUDED.severity,
  alert_name = EXCLUDED.alert_name,
  service = EXCLUDED.service,
  summary = EXCLUDED.summary,
  starts_at = EXCLUDED.starts_at,
  ends_at = EXCLUDED.ends_at,
  assigned_grafana_user_id = EXCLUDED.assigned_grafana_user_id,
  assigned_user_login = EXCLUDED.assigned_user_login,
  assigned_user_name = EXCLUDED.assigned_user_name,
  payload = EXCLUDED.payload,
  updated_at = now()
`, alert.ID, alert.Fingerprint, alert.GroupKey, alert.Status, alert.Severity, alert.AlertName, alert.Service, alert.Summary, alert.StartsAt, alert.EndsAt, alert.AssignedUserID, alert.AssignedUserLogin, alert.AssignedUserName, string(alert.Payload))
	return err
}

func (s *Store) memorySchedules() []Schedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Schedule, len(s.schedules))
	copy(result, s.schedules)
	return result
}

func (s *Store) memoryUpsertAlert(alert Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	alert.UpdatedAt = now
	for index, current := range s.alerts {
		if current.Fingerprint == alert.Fingerprint {
			alert.CreatedAt = current.CreatedAt
			s.alerts[index] = alert
			return
		}
	}
	alert.CreatedAt = now
	s.alerts = append(s.alerts, alert)
}

func buildShift(schedule Schedule, input CreateShift) Shift {
	name := input.UserName
	if name == "" {
		name = input.UserLogin
	}

	return Shift{
		ID:           fmt.Sprintf("shift-%d", time.Now().UnixNano()),
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
		UserID:       input.UserID,
		UserLogin:    input.UserLogin,
		UserName:     name,
		StartsAt:     input.StartsAt,
		EndsAt:       input.EndsAt,
		Status:       statusFor(input.StartsAt, input.EndsAt),
		Layer:        input.Layer,
	}
}

func parseAlert(raw map[string]any, groupKey string) Alert {
	labels := stringMap(raw["labels"])
	annotations := stringMap(raw["annotations"])
	status, _ := raw["status"].(string)
	fingerprint, _ := raw["fingerprint"].(string)
	startsAt := parseOptionalTime(raw["startsAt"])
	endsAt := parseOptionalTime(raw["endsAt"])

	return Alert{
		ID:          fingerprint,
		Fingerprint: fingerprint,
		GroupKey:    groupKey,
		Status:      status,
		Severity:    labels["severity"],
		AlertName:   labels["alertname"],
		Service:     firstNonEmpty(labels["service"], labels["job"], labels["namespace"]),
		Summary:     firstNonEmpty(annotations["summary"], annotations["description"]),
		StartsAt:    startsAt,
		EndsAt:      endsAt,
	}
}

func assignAlertToShift(ctx context.Context, db *pgxpool.Pool, alert *Alert) {
	at := time.Now()
	if alert.StartsAt != nil {
		at = *alert.StartsAt
	}

	_ = db.QueryRow(ctx, `
SELECT grafana_user_id, user_login, user_name
FROM oncall_shifts
WHERE starts_at <= $1 AND ends_at > $1
ORDER BY layer ASC, starts_at DESC
LIMIT 1
`, at).Scan(&alert.AssignedUserID, &alert.AssignedUserLogin, &alert.AssignedUserName)
}

func stringMap(value any) map[string]string {
	result := map[string]string{}
	raw, ok := value.(map[string]any)
	if !ok {
		return result
	}
	for key, item := range raw {
		if text, ok := item.(string); ok {
			result[key] = text
		}
	}
	return result
}

func parseOptionalTime(value any) *time.Time {
	text, ok := value.(string)
	if !ok || text == "" || strings.HasPrefix(text, "0001-01-01") {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return nil
	}
	return &parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func sortShifts(shifts []Shift) {
	sort.Slice(shifts, func(i, j int) bool {
		return shifts[i].StartsAt.Before(shifts[j].StartsAt)
	})
}

func statusFor(start, end time.Time) string {
	now := time.Now()
	if now.Before(start) {
		return "planned"
	}
	if now.After(end) {
		return "finished"
	}
	return "active"
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
