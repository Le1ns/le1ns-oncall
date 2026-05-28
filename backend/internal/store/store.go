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
	JiraIssueKey      string          `json:"jiraIssueKey,omitempty"`
	JiraIssueURL      string          `json:"jiraIssueUrl,omitempty"`
}

type JiraSettings struct {
	Enabled             bool      `json:"enabled"`
	BaseURL             string    `json:"baseUrl"`
	ProjectKey          string    `json:"projectKey"`
	IssueType           string    `json:"issueType"`
	AutoCreateOnFiring  bool      `json:"autoCreateOnFiring"`
	SummaryTemplate     string    `json:"summaryTemplate"`
	DescriptionTemplate string    `json:"descriptionTemplate"`
	Labels              []string  `json:"labels"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type UpdateJiraSettings struct {
	Enabled             bool     `json:"enabled"`
	BaseURL             string   `json:"baseUrl"`
	ProjectKey          string   `json:"projectKey"`
	IssueType           string   `json:"issueType"`
	AutoCreateOnFiring  bool     `json:"autoCreateOnFiring"`
	SummaryTemplate     string   `json:"summaryTemplate"`
	DescriptionTemplate string   `json:"descriptionTemplate"`
	Labels              []string `json:"labels"`
}

type JiraIssue struct {
	ID            string    `json:"id"`
	AlertID       string    `json:"alertId"`
	Fingerprint   string    `json:"fingerprint"`
	IssueKey      string    `json:"issueKey"`
	IssueURL      string    `json:"issueUrl"`
	CreatedByRule string    `json:"createdByRule"`
	CreatedAt     time.Time `json:"createdAt"`
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

type NotificationReceiver struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	BotToken   string    `json:"botToken,omitempty"`
	WebhookURL string    `json:"webhookUrl,omitempty"`
	HasSecret  bool      `json:"hasSecret"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type CreateNotificationReceiver struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	BotToken   string `json:"botToken"`
	WebhookURL string `json:"webhookUrl"`
	Enabled    bool   `json:"enabled"`
}

type NotificationChannel struct {
	ID            string    `json:"id"`
	Kind          string    `json:"kind"`
	ReceiverID    string    `json:"receiverId"`
	ReceiverName  string    `json:"receiverName"`
	TargetType    string    `json:"targetType"`
	Name          string    `json:"name"`
	GrafanaUserID int64     `json:"grafanaUserId"`
	UserLogin     string    `json:"userLogin"`
	UserName      string    `json:"userName"`
	ChatID        string    `json:"chatId"`
	Severities    []string  `json:"severities"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CreateNotificationChannel struct {
	Kind          string   `json:"kind"`
	ReceiverID    string   `json:"receiverId"`
	TargetType    string   `json:"targetType"`
	Name          string   `json:"name"`
	GrafanaUserID int64    `json:"grafanaUserId"`
	UserLogin     string   `json:"userLogin"`
	UserName      string   `json:"userName"`
	ChatID        string   `json:"chatId"`
	Severities    []string `json:"severities"`
	Enabled       bool     `json:"enabled"`
}

type NotificationRoute struct {
	Channel  NotificationChannel
	Receiver NotificationReceiver
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

type AlertIntakeConfig struct {
	AllowedSeverities []string
	MaxPerGroup       int
	RateWindow        time.Duration
	MaxPerWebhook     int
}

type AlertIntakeResult struct {
	Received    int     `json:"received"`
	Saved       int     `json:"saved"`
	Filtered    int     `json:"filtered"`
	RateLimited int     `json:"rateLimited"`
	SavedAlerts []Alert `json:"-"`
}

type alertIntakeWindow struct {
	startedAt time.Time
	count     int
}

type Store struct {
	db            *pgxpool.Pool
	mu            sync.Mutex
	intakeMu      sync.Mutex
	schedules     []Schedule
	shifts        []Shift
	alerts        []Alert
	receivers     []NotificationReceiver
	notifications []NotificationChannel
	jiraSettings  JiraSettings
	jiraIssues    []JiraIssue
	intakeWindows map[string]alertIntakeWindow
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
		intakeWindows: map[string]alertIntakeWindow{},
		schedules:     []Schedule{schedule},
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
		jiraSettings: JiraSettings{
			Enabled:             false,
			BaseURL:             "",
			ProjectKey:          "OPS",
			IssueType:           "Task",
			AutoCreateOnFiring:  true,
			SummaryTemplate:     "[{{severity}}] {{alertName}}",
			DescriptionTemplate: "{{summary}}",
			Labels:              []string{"oncall"},
		},
	}
}

func (s *Store) NotificationReceivers() []NotificationReceiver {
	if s.db != nil {
		return s.pgNotificationReceivers()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]NotificationReceiver, 0, len(s.receivers))
	for _, receiver := range s.receivers {
		result = append(result, receiver.public())
	}
	sortNotificationReceivers(result)
	return result
}

func (s *Store) CreateNotificationReceiver(input CreateNotificationReceiver) (NotificationReceiver, error) {
	receiver, err := normalizeNotificationReceiver(input)
	if err != nil {
		return NotificationReceiver{}, err
	}
	if s.db != nil {
		return s.pgCreateNotificationReceiver(receiver)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.receivers = append(s.receivers, receiver)
	return receiver.public(), nil
}

func (s *Store) DeleteNotificationReceiver(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("notification receiver id is required")
	}
	if s.db != nil {
		return s.pgDeleteNotificationReceiver(id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.receivers {
		if s.receivers[index].ID == id {
			s.receivers = append(s.receivers[:index], s.receivers[index+1:]...)
			s.notifications = deleteChannelsByReceiver(s.notifications, id)
			return nil
		}
	}
	return fmt.Errorf("notification receiver not found")
}

func (s *Store) NotificationChannels() []NotificationChannel {
	if s.db != nil {
		return s.pgNotificationChannels()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]NotificationChannel, len(s.notifications))
	copy(result, s.notifications)
	sortNotificationChannels(result)
	return result
}

func (s *Store) CreateNotificationChannel(input CreateNotificationChannel) (NotificationChannel, error) {
	channel, err := s.normalizeNotificationChannel(input)
	if err != nil {
		return NotificationChannel{}, err
	}
	if s.db != nil {
		return s.pgCreateNotificationChannel(channel)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.notifications = append(s.notifications, channel)
	return channel, nil
}

func (s *Store) UpdateNotificationChannel(id string, input CreateNotificationChannel) (NotificationChannel, error) {
	channel, err := s.normalizeNotificationChannel(input)
	if err != nil {
		return NotificationChannel{}, err
	}
	channel.ID = strings.TrimSpace(id)
	if channel.ID == "" {
		return NotificationChannel{}, fmt.Errorf("notification channel id is required")
	}

	if s.db != nil {
		return s.pgUpdateNotificationChannel(channel)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.notifications {
		if s.notifications[index].ID == channel.ID {
			channel.CreatedAt = s.notifications[index].CreatedAt
			s.notifications[index] = channel
			return channel, nil
		}
	}
	return NotificationChannel{}, fmt.Errorf("notification channel not found")
}

func (s *Store) DeleteNotificationChannel(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("notification channel id is required")
	}
	if s.db != nil {
		return s.pgDeleteNotificationChannel(id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.notifications {
		if s.notifications[index].ID == id {
			s.notifications = append(s.notifications[:index], s.notifications[index+1:]...)
			return nil
		}
	}
	return fmt.Errorf("notification channel not found")
}

func (s *Store) AlertNotificationChannels(alert Alert) []NotificationChannel {
	routes := s.AlertNotificationRoutes(alert)
	channels := make([]NotificationChannel, 0, len(routes))
	for _, route := range routes {
		channels = append(channels, route.Channel)
	}
	return channels
}

func (s *Store) AlertNotificationRoutes(alert Alert) []NotificationRoute {
	channels := s.NotificationChannels()
	result := make([]NotificationRoute, 0, len(channels))
	for _, channel := range channels {
		receiver, ok := s.notificationReceiverForChannel(channel)
		if !ok || !channel.Enabled || !receiver.Enabled {
			continue
		}
		if !severityAllowed(alert.Severity, severitySet(channel.Severities)) {
			continue
		}
		if channel.TargetType == "group" {
			result = append(result, NotificationRoute{Channel: channel, Receiver: receiver})
			continue
		}
		if channel.TargetType == "person" && channelMatchesAlertUser(channel, alert) {
			result = append(result, NotificationRoute{Channel: channel, Receiver: receiver})
		}
	}
	return dedupeNotificationRoutes(result)
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

func (s *Store) AlertsByPeriod(from, to time.Time, limit int) []Alert {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	if s.db != nil {
		return s.pgAlertsByPeriod(from, to, limit)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		t := alert.CreatedAt
		if alert.StartsAt != nil {
			t = *alert.StartsAt
		}
		if !t.Before(from) && t.Before(to) {
			result = append(result, alert)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if len(result) > limit {
		return result[:limit]
	}
	return result
}

func (s *Store) JiraSettings() JiraSettings {
	if s.db != nil {
		return s.pgJiraSettings()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneJiraSettings(s.jiraSettings)
}

func (s *Store) UpdateJiraSettings(input UpdateJiraSettings) (JiraSettings, error) {
	settings, err := normalizeJiraSettings(input)
	if err != nil {
		return JiraSettings{}, err
	}
	if s.db != nil {
		return s.pgUpsertJiraSettings(settings)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	settings.UpdatedAt = time.Now().UTC()
	s.jiraSettings = settings
	return cloneJiraSettings(settings), nil
}

func (s *Store) JiraIssueByFingerprint(fingerprint string) (JiraIssue, bool) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return JiraIssue{}, false
	}
	if s.db != nil {
		return s.pgJiraIssueByFingerprint(fingerprint)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, issue := range s.jiraIssues {
		if issue.Fingerprint == fingerprint {
			return issue, true
		}
	}
	return JiraIssue{}, false
}

func (s *Store) CreateJiraIssue(issue JiraIssue) (JiraIssue, error) {
	if s.db != nil {
		return s.pgCreateJiraIssue(issue)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, current := range s.jiraIssues {
		if current.Fingerprint == issue.Fingerprint {
			return JiraIssue{}, fmt.Errorf("jira issue for this alert already exists")
		}
	}
	if issue.ID == "" {
		issue.ID = fmt.Sprintf("jira-%d", time.Now().UnixNano())
	}
	if issue.CreatedAt.IsZero() {
		issue.CreatedAt = time.Now().UTC()
	}
	s.jiraIssues = append(s.jiraIssues, issue)
	return issue, nil
}

func (s *Store) JiraIssues(from, to time.Time) []JiraIssue {
	if s.db != nil {
		return s.pgJiraIssues(from, to)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]JiraIssue, 0, len(s.jiraIssues))
	for _, issue := range s.jiraIssues {
		if !issue.CreatedAt.Before(from) && issue.CreatedAt.Before(to) {
			result = append(result, issue)
		}
	}
	return result
}

func (s *Store) SaveAlertmanagerPayload(payload map[string]any, config AlertIntakeConfig) (AlertIntakeResult, error) {
	alertItems, ok := payload["alerts"].([]any)
	if !ok {
		return AlertIntakeResult{}, nil
	}

	result := AlertIntakeResult{Received: len(alertItems)}
	groupKey, _ := payload["groupKey"].(string)
	severities := severitySet(config.AllowedSeverities)
	now := time.Now()
	for index, item := range alertItems {
		if config.MaxPerWebhook > 0 && index >= config.MaxPerWebhook {
			result.RateLimited += len(alertItems) - index
			break
		}
		alertMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		alert := parseAlert(alertMap, groupKey)
		if !severityAllowed(alert.Severity, severities) {
			result.Filtered++
			continue
		}
		if !s.allowAlertIntake(intakeKey(alert), now, config) {
			result.RateLimited++
			continue
		}
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
			s.pgAssignAlertToShift(&alert)
			if err := s.pgUpsertAlert(alert); err != nil {
				return result, err
			}
		} else {
			s.memoryUpsertAlert(alert)
		}
		result.Saved++
		result.SavedAlerts = append(result.SavedAlerts, alert)
	}
	return result, nil
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
	for _, issue := range s.JiraIssues(from, to) {
		totals.JiraIssuesCreated++
		if issue.AlertID == "" {
			continue
		}
		alert, ok := findAlertByID(alerts, issue.AlertID)
		if !ok || alert.AssignedUserID == 0 {
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
		person.JiraIssuesCreated++
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

CREATE TABLE IF NOT EXISTS oncall_jira_settings (
  id BOOLEAN PRIMARY KEY DEFAULT true,
  enabled BOOLEAN NOT NULL DEFAULT false,
  base_url TEXT NOT NULL DEFAULT '',
  project_key TEXT NOT NULL DEFAULT 'OPS',
  issue_type TEXT NOT NULL DEFAULT 'Task',
  auto_create_on_firing BOOLEAN NOT NULL DEFAULT true,
  summary_template TEXT NOT NULL DEFAULT '[{{severity}}] {{alertName}}',
  description_template TEXT NOT NULL DEFAULT '{{summary}}',
  labels TEXT NOT NULL DEFAULT 'oncall',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oncall_jira_issues (
  id TEXT PRIMARY KEY,
  alert_id TEXT NOT NULL DEFAULT '',
  fingerprint TEXT NOT NULL UNIQUE,
  issue_key TEXT NOT NULL UNIQUE,
  issue_url TEXT NOT NULL,
  created_by_rule TEXT NOT NULL DEFAULT 'manual',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS oncall_jira_issues_created_at_idx ON oncall_jira_issues(created_at DESC);

CREATE TABLE IF NOT EXISTS oncall_notification_receivers (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  bot_token TEXT NOT NULL DEFAULT '',
  webhook_url TEXT NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS oncall_notification_receivers_kind_idx ON oncall_notification_receivers(kind);

CREATE TABLE IF NOT EXISTS oncall_notification_channels (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  receiver_id TEXT NOT NULL DEFAULT '',
  target_type TEXT NOT NULL,
  name TEXT NOT NULL,
  grafana_user_id BIGINT NOT NULL DEFAULT 0,
  user_login TEXT NOT NULL DEFAULT '',
  user_name TEXT NOT NULL DEFAULT '',
  chat_id TEXT NOT NULL,
  severities TEXT NOT NULL DEFAULT 'critical',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS oncall_notification_channels_kind_idx ON oncall_notification_channels(kind, target_type);

ALTER TABLE oncall_notification_channels ADD COLUMN IF NOT EXISTS receiver_id TEXT NOT NULL DEFAULT '';
ALTER TABLE oncall_notification_channels ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'telegram';
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
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, `
INSERT INTO oncall_jira_settings (id, enabled, base_url, project_key, issue_type, auto_create_on_firing, summary_template, description_template, labels)
VALUES (true, $1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING
`, s.jiraSettings.Enabled, s.jiraSettings.BaseURL, s.jiraSettings.ProjectKey, s.jiraSettings.IssueType, s.jiraSettings.AutoCreateOnFiring, s.jiraSettings.SummaryTemplate, s.jiraSettings.DescriptionTemplate, strings.Join(s.jiraSettings.Labels, ","))
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
  a.id,
  a.fingerprint,
  a.group_key,
  a.status,
  a.severity,
  a.alert_name,
  a.service,
  a.summary,
  a.starts_at,
  a.ends_at,
  a.assigned_grafana_user_id,
  a.assigned_user_login,
  a.assigned_user_name,
  a.payload,
  a.created_at,
  a.updated_at,
  COALESCE(j.issue_key, ''),
  COALESCE(j.issue_url, '')
FROM oncall_alerts a
LEFT JOIN oncall_jira_issues j ON j.fingerprint = a.fingerprint
ORDER BY a.created_at DESC
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
			&alert.JiraIssueKey,
			&alert.JiraIssueURL,
		); err != nil {
			return nil
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func (s *Store) pgAlertsByPeriod(from, to time.Time, limit int) []Alert {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
SELECT
  a.id,
  a.fingerprint,
  a.group_key,
  a.status,
  a.severity,
  a.alert_name,
  a.service,
  a.summary,
  a.starts_at,
  a.ends_at,
  a.assigned_grafana_user_id,
  a.assigned_user_login,
  a.assigned_user_name,
  a.payload,
  a.created_at,
  a.updated_at,
  COALESCE(j.issue_key, ''),
  COALESCE(j.issue_url, '')
FROM oncall_alerts a
LEFT JOIN oncall_jira_issues j ON j.fingerprint = a.fingerprint
WHERE COALESCE(a.starts_at, a.created_at) >= $1
  AND COALESCE(a.starts_at, a.created_at) < $2
ORDER BY a.created_at DESC
LIMIT $3
`, from, to, limit)
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
			&alert.JiraIssueKey,
			&alert.JiraIssueURL,
		); err != nil {
			return nil
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func (s *Store) pgJiraSettings() JiraSettings {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var settings JiraSettings
	var labels string
	err := s.db.QueryRow(ctx, `
SELECT enabled, base_url, project_key, issue_type, auto_create_on_firing, summary_template, description_template, labels, updated_at
FROM oncall_jira_settings
WHERE id = true
`).Scan(&settings.Enabled, &settings.BaseURL, &settings.ProjectKey, &settings.IssueType, &settings.AutoCreateOnFiring, &settings.SummaryTemplate, &settings.DescriptionTemplate, &labels, &settings.UpdatedAt)
	if err != nil {
		return cloneJiraSettings(s.jiraSettings)
	}
	settings.Labels = splitCSV(labels)
	return settings
}

func (s *Store) pgUpsertJiraSettings(settings JiraSettings) (JiraSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	settings.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(ctx, `
INSERT INTO oncall_jira_settings (id, enabled, base_url, project_key, issue_type, auto_create_on_firing, summary_template, description_template, labels, updated_at)
VALUES (true, $1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  base_url = EXCLUDED.base_url,
  project_key = EXCLUDED.project_key,
  issue_type = EXCLUDED.issue_type,
  auto_create_on_firing = EXCLUDED.auto_create_on_firing,
  summary_template = EXCLUDED.summary_template,
  description_template = EXCLUDED.description_template,
  labels = EXCLUDED.labels,
  updated_at = EXCLUDED.updated_at
`, settings.Enabled, settings.BaseURL, settings.ProjectKey, settings.IssueType, settings.AutoCreateOnFiring, settings.SummaryTemplate, settings.DescriptionTemplate, strings.Join(settings.Labels, ","), settings.UpdatedAt)
	if err != nil {
		return JiraSettings{}, err
	}
	return settings, nil
}

func (s *Store) pgJiraIssueByFingerprint(fingerprint string) (JiraIssue, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var issue JiraIssue
	err := s.db.QueryRow(ctx, `
SELECT id, alert_id, fingerprint, issue_key, issue_url, created_by_rule, created_at
FROM oncall_jira_issues
WHERE fingerprint = $1
`, fingerprint).Scan(&issue.ID, &issue.AlertID, &issue.Fingerprint, &issue.IssueKey, &issue.IssueURL, &issue.CreatedByRule, &issue.CreatedAt)
	if err != nil {
		return JiraIssue{}, false
	}
	return issue, true
}

func (s *Store) pgCreateJiraIssue(issue JiraIssue) (JiraIssue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if issue.ID == "" {
		issue.ID = fmt.Sprintf("jira-%d", time.Now().UnixNano())
	}
	if issue.CreatedAt.IsZero() {
		issue.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(ctx, `
INSERT INTO oncall_jira_issues (id, alert_id, fingerprint, issue_key, issue_url, created_by_rule, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, issue.ID, issue.AlertID, issue.Fingerprint, issue.IssueKey, issue.IssueURL, issue.CreatedByRule, issue.CreatedAt)
	if err != nil {
		return JiraIssue{}, err
	}
	return issue, nil
}

func (s *Store) pgJiraIssues(from, to time.Time) []JiraIssue {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.db.Query(ctx, `
SELECT id, alert_id, fingerprint, issue_key, issue_url, created_by_rule, created_at
FROM oncall_jira_issues
WHERE created_at >= $1 AND created_at < $2
ORDER BY created_at DESC
`, from, to)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := []JiraIssue{}
	for rows.Next() {
		var issue JiraIssue
		if err := rows.Scan(&issue.ID, &issue.AlertID, &issue.Fingerprint, &issue.IssueKey, &issue.IssueURL, &issue.CreatedByRule, &issue.CreatedAt); err != nil {
			return nil
		}
		result = append(result, issue)
	}
	return result
}

func (s *Store) pgNotificationReceivers() []NotificationReceiver {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
SELECT id, kind, name, bot_token <> '' OR webhook_url <> '' AS has_secret, enabled, created_at, updated_at
FROM oncall_notification_receivers
ORDER BY created_at DESC
`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	receivers := []NotificationReceiver{}
	for rows.Next() {
		var receiver NotificationReceiver
		if err := rows.Scan(
			&receiver.ID,
			&receiver.Kind,
			&receiver.Name,
			&receiver.HasSecret,
			&receiver.Enabled,
			&receiver.CreatedAt,
			&receiver.UpdatedAt,
		); err != nil {
			return nil
		}
		receivers = append(receivers, receiver)
	}
	return receivers
}

func (s *Store) pgCreateNotificationReceiver(receiver NotificationReceiver) (NotificationReceiver, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.db.Exec(ctx, `
INSERT INTO oncall_notification_receivers (
  id, kind, name, bot_token, webhook_url, enabled, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, receiver.ID, receiver.Kind, receiver.Name, receiver.BotToken, receiver.WebhookURL, receiver.Enabled, receiver.CreatedAt, receiver.UpdatedAt)
	if err != nil {
		return NotificationReceiver{}, err
	}
	return receiver.public(), nil
}

func (s *Store) pgDeleteNotificationReceiver(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := s.db.Exec(ctx, `DELETE FROM oncall_notification_channels WHERE receiver_id = $1`, id); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM oncall_notification_receivers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification receiver not found")
	}
	return nil
}

func (s *Store) pgNotificationReceiver(id string) (NotificationReceiver, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var receiver NotificationReceiver
	err := s.db.QueryRow(ctx, `
SELECT id, kind, name, bot_token, webhook_url, enabled, created_at, updated_at
FROM oncall_notification_receivers
WHERE id = $1
`, id).Scan(
		&receiver.ID,
		&receiver.Kind,
		&receiver.Name,
		&receiver.BotToken,
		&receiver.WebhookURL,
		&receiver.Enabled,
		&receiver.CreatedAt,
		&receiver.UpdatedAt,
	)
	if err != nil {
		return NotificationReceiver{}, false
	}
	receiver.HasSecret = receiver.BotToken != "" || receiver.WebhookURL != ""
	return receiver, true
}

func (s *Store) pgNotificationChannels() []NotificationChannel {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
SELECT
  ch.id,
  COALESCE(NULLIF(rc.kind, ''), ch.kind) AS kind,
  ch.receiver_id,
  COALESCE(rc.name, '') AS receiver_name,
  ch.target_type,
  ch.name,
  ch.grafana_user_id,
  ch.user_login,
  ch.user_name,
  ch.chat_id,
  ch.severities,
  ch.enabled,
  ch.created_at,
  ch.updated_at
FROM oncall_notification_channels ch
LEFT JOIN oncall_notification_receivers rc ON rc.id = ch.receiver_id
ORDER BY ch.created_at DESC
`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	channels := []NotificationChannel{}
	for rows.Next() {
		var channel NotificationChannel
		var severities string
		if err := rows.Scan(
			&channel.ID,
			&channel.Kind,
			&channel.ReceiverID,
			&channel.ReceiverName,
			&channel.TargetType,
			&channel.Name,
			&channel.GrafanaUserID,
			&channel.UserLogin,
			&channel.UserName,
			&channel.ChatID,
			&severities,
			&channel.Enabled,
			&channel.CreatedAt,
			&channel.UpdatedAt,
		); err != nil {
			return nil
		}
		channel.Severities = splitCSV(severities)
		channels = append(channels, channel)
	}
	return channels
}

func (s *Store) pgCreateNotificationChannel(channel NotificationChannel) (NotificationChannel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.db.Exec(ctx, `
INSERT INTO oncall_notification_channels (
  id, kind, receiver_id, target_type, name, grafana_user_id, user_login, user_name, chat_id, severities, enabled, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`, channel.ID, channel.Kind, channel.ReceiverID, channel.TargetType, channel.Name, channel.GrafanaUserID, channel.UserLogin, channel.UserName, channel.ChatID, strings.Join(channel.Severities, ","), channel.Enabled, channel.CreatedAt, channel.UpdatedAt)
	if err != nil {
		return NotificationChannel{}, err
	}
	return channel, nil
}

func (s *Store) pgUpdateNotificationChannel(channel NotificationChannel) (NotificationChannel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var createdAt time.Time
	err := s.db.QueryRow(ctx, `
UPDATE oncall_notification_channels
SET kind = $2,
    receiver_id = $3,
    target_type = $4,
    name = $5,
    grafana_user_id = $6,
    user_login = $7,
    user_name = $8,
    chat_id = $9,
    severities = $10,
    enabled = $11,
    updated_at = $12
WHERE id = $1
RETURNING created_at
`, channel.ID, channel.Kind, channel.ReceiverID, channel.TargetType, channel.Name, channel.GrafanaUserID, channel.UserLogin, channel.UserName, channel.ChatID, strings.Join(channel.Severities, ","), channel.Enabled, channel.UpdatedAt).Scan(&createdAt)
	if err != nil {
		return NotificationChannel{}, err
	}
	channel.CreatedAt = createdAt
	return channel, nil
}

func (s *Store) pgDeleteNotificationChannel(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tag, err := s.db.Exec(ctx, `DELETE FROM oncall_notification_channels WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification channel not found")
	}
	return nil
}

func (s *Store) pgUpsertAlert(alert Alert) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

func (s *Store) pgAssignAlertToShift(alert *Alert) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assignAlertToShift(ctx, s.db, alert)
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

func (s *Store) allowAlertIntake(key string, now time.Time, config AlertIntakeConfig) bool {
	if config.MaxPerGroup <= 0 || config.RateWindow <= 0 {
		return true
	}

	s.intakeMu.Lock()
	defer s.intakeMu.Unlock()

	if s.intakeWindows == nil {
		s.intakeWindows = map[string]alertIntakeWindow{}
	}

	window := s.intakeWindows[key]
	if window.startedAt.IsZero() || now.Sub(window.startedAt) >= config.RateWindow {
		s.intakeWindows[key] = alertIntakeWindow{startedAt: now, count: 1}
		return true
	}
	if window.count >= config.MaxPerGroup {
		return false
	}
	window.count++
	s.intakeWindows[key] = window
	return true
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

func severitySet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized != "" {
			result[normalized] = true
		}
	}
	return result
}

func severityAllowed(severity string, allowed map[string]bool) bool {
	if len(allowed) == 0 || allowed["*"] {
		return true
	}
	return allowed[strings.ToLower(strings.TrimSpace(severity))]
}

func intakeKey(alert Alert) string {
	if alert.GroupKey != "" {
		return alert.GroupKey
	}
	if alert.Service != "" || alert.AlertName != "" {
		return alert.Service + ":" + alert.AlertName
	}
	return "ungrouped"
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

func normalizeJiraSettings(input UpdateJiraSettings) (JiraSettings, error) {
	projectKey := strings.ToUpper(strings.TrimSpace(input.ProjectKey))
	if projectKey == "" {
		return JiraSettings{}, fmt.Errorf("projectKey is required")
	}
	issueType := strings.TrimSpace(input.IssueType)
	if issueType == "" {
		issueType = "Task"
	}
	summaryTemplate := strings.TrimSpace(input.SummaryTemplate)
	if summaryTemplate == "" {
		summaryTemplate = "[{{severity}}] {{alertName}}"
	}
	descriptionTemplate := strings.TrimSpace(input.DescriptionTemplate)
	if descriptionTemplate == "" {
		descriptionTemplate = "{{summary}}"
	}
	return JiraSettings{
		Enabled:             input.Enabled,
		BaseURL:             strings.TrimSpace(input.BaseURL),
		ProjectKey:          projectKey,
		IssueType:           issueType,
		AutoCreateOnFiring:  input.AutoCreateOnFiring,
		SummaryTemplate:     summaryTemplate,
		DescriptionTemplate: descriptionTemplate,
		Labels:              normalizeSeverities(input.Labels),
	}, nil
}

func cloneJiraSettings(settings JiraSettings) JiraSettings {
	out := settings
	out.Labels = append([]string{}, settings.Labels...)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func findAlertByID(alerts []Alert, id string) (Alert, bool) {
	for _, alert := range alerts {
		if alert.ID == id {
			return alert, true
		}
	}
	return Alert{}, false
}

func normalizeNotificationReceiver(input CreateNotificationReceiver) (NotificationReceiver, error) {
	kind := strings.ToLower(strings.TrimSpace(input.Kind))
	name := strings.TrimSpace(input.Name)
	botToken := strings.TrimSpace(input.BotToken)
	webhookURL := strings.TrimSpace(input.WebhookURL)
	if kind == "" {
		kind = "telegram"
	}
	if kind != "telegram" && kind != "lark" {
		return NotificationReceiver{}, fmt.Errorf("receiver kind must be telegram or lark")
	}
	if name == "" {
		return NotificationReceiver{}, fmt.Errorf("receiver name is required")
	}
	if kind == "telegram" && botToken == "" {
		return NotificationReceiver{}, fmt.Errorf("telegram bot token is required")
	}
	if kind == "lark" && webhookURL == "" {
		return NotificationReceiver{}, fmt.Errorf("lark webhook url is required")
	}

	now := time.Now().UTC()
	return NotificationReceiver{
		ID:         fmt.Sprintf("receiver-%d", now.UnixNano()),
		Kind:       kind,
		Name:       name,
		BotToken:   botToken,
		WebhookURL: webhookURL,
		HasSecret:  botToken != "" || webhookURL != "",
		Enabled:    input.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (s *Store) normalizeNotificationChannel(input CreateNotificationChannel) (NotificationChannel, error) {
	receiverID := strings.TrimSpace(input.ReceiverID)
	if receiverID == "" {
		return NotificationChannel{}, fmt.Errorf("receiverId is required")
	}
	receiver, ok := s.notificationReceiverByID(receiverID)
	if !ok {
		return NotificationChannel{}, fmt.Errorf("notification receiver not found")
	}

	kind := receiver.Kind
	targetType := strings.ToLower(strings.TrimSpace(input.TargetType))
	if targetType != "person" && targetType != "group" {
		return NotificationChannel{}, fmt.Errorf("targetType must be person or group")
	}
	if kind == "lark" && targetType != "group" {
		return NotificationChannel{}, fmt.Errorf("lark notifications support group targets only")
	}

	chatID := strings.TrimSpace(input.ChatID)
	if kind == "telegram" && chatID == "" {
		return NotificationChannel{}, fmt.Errorf("chatId is required")
	}
	if kind == "lark" {
		chatID = ""
	}

	name := strings.TrimSpace(input.Name)
	userLogin := strings.TrimSpace(input.UserLogin)
	userName := strings.TrimSpace(input.UserName)
	if targetType == "person" {
		if input.GrafanaUserID == 0 || userLogin == "" {
			return NotificationChannel{}, fmt.Errorf("grafana user is required for person notification")
		}
		if name == "" {
			name = firstNonEmpty(userName, userLogin)
		}
	}
	if targetType == "group" && name == "" {
		return NotificationChannel{}, fmt.Errorf("name is required for group notification")
	}

	severities := normalizeSeverities(input.Severities)
	if len(severities) == 0 {
		severities = []string{"critical"}
	}

	now := time.Now().UTC()
	return NotificationChannel{
		ID:            fmt.Sprintf("notification-%d", now.UnixNano()),
		Kind:          kind,
		ReceiverID:    receiver.ID,
		ReceiverName:  receiver.Name,
		TargetType:    targetType,
		Name:          name,
		GrafanaUserID: input.GrafanaUserID,
		UserLogin:     userLogin,
		UserName:      userName,
		ChatID:        chatID,
		Severities:    severities,
		Enabled:       input.Enabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (s *Store) notificationReceiverByID(id string) (NotificationReceiver, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return NotificationReceiver{}, false
	}
	if s.db != nil {
		return s.pgNotificationReceiver(id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, receiver := range s.receivers {
		if receiver.ID == id {
			return receiver, true
		}
	}
	return NotificationReceiver{}, false
}

func (s *Store) notificationReceiverForChannel(channel NotificationChannel) (NotificationReceiver, bool) {
	return s.notificationReceiverByID(channel.ReceiverID)
}

func (r NotificationReceiver) public() NotificationReceiver {
	r.HasSecret = r.BotToken != "" || r.WebhookURL != ""
	r.BotToken = ""
	r.WebhookURL = ""
	return r
}

func normalizeSeverities(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	return result
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	return normalizeSeverities(strings.Split(value, ","))
}

func channelMatchesAlertUser(channel NotificationChannel, alert Alert) bool {
	if channel.GrafanaUserID != 0 && alert.AssignedUserID != 0 && channel.GrafanaUserID == alert.AssignedUserID {
		return true
	}
	return channel.UserLogin != "" && alert.AssignedUserLogin != "" && strings.EqualFold(channel.UserLogin, alert.AssignedUserLogin)
}

func dedupeNotificationRoutes(routes []NotificationRoute) []NotificationRoute {
	seen := map[string]bool{}
	result := make([]NotificationRoute, 0, len(routes))
	for _, route := range routes {
		key := route.Receiver.ID + ":" + route.Channel.ChatID + ":" + route.Channel.TargetType + ":" + route.Channel.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, route)
	}
	return result
}

func deleteChannelsByReceiver(channels []NotificationChannel, receiverID string) []NotificationChannel {
	result := channels[:0]
	for _, channel := range channels {
		if channel.ReceiverID != receiverID {
			result = append(result, channel)
		}
	}
	return result
}

func sortNotificationReceivers(receivers []NotificationReceiver) {
	sort.Slice(receivers, func(i, j int) bool {
		return receivers[i].CreatedAt.After(receivers[j].CreatedAt)
	})
}

func sortNotificationChannels(channels []NotificationChannel) {
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].CreatedAt.After(channels[j].CreatedAt)
	})
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
