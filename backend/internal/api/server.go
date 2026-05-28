package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/local/oncall-plugin/backend/internal/config"
	"github.com/local/oncall-plugin/backend/internal/httpx"
	"github.com/local/oncall-plugin/backend/internal/jira"
	"github.com/local/oncall-plugin/backend/internal/notify"
	"github.com/local/oncall-plugin/backend/internal/store"
)

type Server struct {
	config config.Config
	log    *slog.Logger
	store  *store.Store
	tg     *notify.Telegram
	jira   *jira.Client
	mux    *http.ServeMux
}

func NewServer(cfg config.Config, log *slog.Logger, st *store.Store) *Server {
	s := &Server{
		config: cfg,
		log:    log,
		store:  st,
		tg: notify.NewTelegram(notify.TelegramConfig{
			BotToken:    cfg.TelegramBotToken,
			ChatID:      cfg.TelegramChatID,
			ChatMapping: cfg.TelegramChatMapping,
		}),
		jira: jira.New(jira.Config{
			BaseURL:  cfg.JiraBaseURL,
			Email:    cfg.JiraEmail,
			APIToken: cfg.JiraAPIToken,
		}),
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return cors(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /api/v1/meta", s.meta)
	s.mux.HandleFunc("GET /api/v1/grafana/users", s.grafanaUsers)
	s.mux.HandleFunc("GET /api/v1/schedules", s.schedules)
	s.mux.HandleFunc("GET /api/v1/calendar", s.calendar)
	s.mux.HandleFunc("GET /api/v1/shifts", s.calendar)
	s.mux.HandleFunc("POST /api/v1/shifts", s.createShift)
	s.mux.HandleFunc("GET /api/v1/alerts", s.alerts)
	s.mux.HandleFunc("POST /api/v1/alerts/{id}/jira-issue", s.createAlertJiraIssue)
	s.mux.HandleFunc("GET /api/v1/integrations/jira/settings", s.jiraSettings)
	s.mux.HandleFunc("PUT /api/v1/integrations/jira/settings", s.updateJiraSettings)
	s.mux.HandleFunc("GET /api/v1/notification-receivers", s.notificationReceivers)
	s.mux.HandleFunc("POST /api/v1/notification-receivers", s.createNotificationReceiver)
	s.mux.HandleFunc("DELETE /api/v1/notification-receivers/{id}", s.deleteNotificationReceiver)
	s.mux.HandleFunc("GET /api/v1/notification-channels", s.notificationChannels)
	s.mux.HandleFunc("POST /api/v1/notification-channels", s.createNotificationChannel)
	s.mux.HandleFunc("PUT /api/v1/notification-channels/{id}", s.updateNotificationChannel)
	s.mux.HandleFunc("DELETE /api/v1/notification-channels/{id}", s.deleteNotificationChannel)
	s.mux.HandleFunc("GET /api/v1/reports/daily", s.dailyReport)
	s.mux.HandleFunc("GET /api/v1/reports/period", s.periodReport)
	s.mux.HandleFunc("POST /api/v1/integrations/alertmanager/webhook", s.alertmanagerWebhook)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	receivers := s.store.NotificationReceivers()
	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "oncall-api",
		"features": map[string]bool{
			"grafanaUsers": s.config.GrafanaServiceToken != "" || s.config.GrafanaBasicAuth != "",
			"jiraCloud":    s.config.JiraBaseURL != "" && s.config.JiraEmail != "" && s.config.JiraAPIToken != "",
			"telegram":     s.tg.Enabled() || hasReceiverKind(receivers, "telegram"),
			"lark":         s.config.LarkWebhookURL != "" || hasReceiverKind(receivers, "lark"),
		},
	})
}

func (s *Server) grafanaUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.fetchGrafanaUsers(r.Context())
	if err != nil {
		s.log.Warn("grafana user fetch failed, returning local fallback", "error", err)
		httpx.JSON(w, http.StatusOK, map[string]any{
			"source": s.config.GrafanaURL,
			"users": []store.GrafanaUser{
				{ID: 1, Login: "admin", Name: "Grafana Admin", Email: "admin@localhost", Role: "Admin"},
			},
			"note": fmt.Sprintf("Grafana user sync failed: %s", err.Error()),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"source": s.config.GrafanaURL,
		"users":  users,
	})
}

func (s *Server) schedules(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"schedules": s.store.Schedules(),
	})
}

func (s *Server) calendar(w http.ResponseWriter, r *http.Request) {
	from, to := periodFromQuery(r)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"range": map[string]string{
			"from": from.Format(time.DateOnly),
			"to":   to.Add(-time.Nanosecond).Format(time.DateOnly),
		},
		"shifts": s.store.Shifts(from, to),
	})
}

func (s *Server) createShift(w http.ResponseWriter, r *http.Request) {
	var input store.CreateShift
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	shift, err := s.store.CreateShift(input)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	s.notifyShiftCreated(r.Context(), shift)

	httpx.JSON(w, http.StatusCreated, shift)
}

func (s *Server) alerts(w http.ResponseWriter, r *http.Request) {
	from, to := periodFromQuery(r)
	limit := 200
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"range": map[string]string{
			"from": from.Format(time.DateOnly),
			"to":   to.Add(-time.Nanosecond).Format(time.DateOnly),
		},
		"alerts": s.store.AlertsByPeriod(from, to, limit),
	})
}

func (s *Server) notificationChannels(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"channels": s.store.NotificationChannels(),
	})
}

func (s *Server) notificationReceivers(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"receivers": s.store.NotificationReceivers(),
	})
}

func (s *Server) createNotificationReceiver(w http.ResponseWriter, r *http.Request) {
	var input store.CreateNotificationReceiver
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	receiver, err := s.store.CreateNotificationReceiver(input)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, receiver)
}

func (s *Server) deleteNotificationReceiver(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteNotificationReceiver(r.PathValue("id")); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) createNotificationChannel(w http.ResponseWriter, r *http.Request) {
	var input store.CreateNotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	channel, err := s.store.CreateNotificationChannel(input)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, channel)
}

func (s *Server) updateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	var input store.CreateNotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	channel, err := s.store.UpdateNotificationChannel(r.PathValue("id"), input)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, channel)
}

func (s *Server) deleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteNotificationChannel(r.PathValue("id")); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) dailyReport(w http.ResponseWriter, _ *http.Request) {
	now := time.Now()
	report := s.store.Report(startOfDay(now), startOfDay(now).Add(24*time.Hour))
	httpx.JSON(w, http.StatusOK, map[string]any{
		"date":              now.Format(time.DateOnly),
		"primaryOnCall":     primaryOnCall(report.People),
		"alertsTotal":       report.Totals.Alerts,
		"jiraIssuesCreated": report.Totals.JiraIssuesCreated,
		"unresolvedAlerts":  0,
	})
}

func (s *Server) periodReport(w http.ResponseWriter, r *http.Request) {
	from, to := periodFromQuery(r)
	httpx.JSON(w, http.StatusOK, s.store.Report(from, to))
}

func (s *Server) alertmanagerWebhook(w http.ResponseWriter, r *http.Request) {
	if s.config.AlertmanagerToken != "" {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token != s.config.AlertmanagerToken {
			httpx.Error(w, http.StatusUnauthorized, "invalid alertmanager token")
			return
		}
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	result, err := s.store.SaveAlertmanagerPayload(payload, store.AlertIntakeConfig{
		AllowedSeverities: s.config.AlertSeverities,
		MaxPerGroup:       s.config.AlertMaxPerGroup,
		RateWindow:        s.config.AlertRateWindow,
		MaxPerWebhook:     s.config.AlertMaxPerWebhook,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.log.Info(
		"received alertmanager webhook",
		"received_alerts", result.Received,
		"saved_alerts", result.Saved,
		"filtered_alerts", result.Filtered,
		"rate_limited_alerts", result.RateLimited,
	)
	s.notifyCriticalAlerts(r.Context(), result.SavedAlerts)
	s.createJiraIssuesForAlerts(r.Context(), result.SavedAlerts)
	httpx.JSON(w, http.StatusAccepted, map[string]any{
		"status":      "accepted",
		"alerts":      result.Saved,
		"received":    result.Received,
		"filtered":    result.Filtered,
		"rateLimited": result.RateLimited,
	})
}

func (s *Server) jiraSettings(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, s.store.JiraSettings())
}

func (s *Server) updateJiraSettings(w http.ResponseWriter, r *http.Request) {
	var input store.UpdateJiraSettings
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	settings, err := s.store.UpdateJiraSettings(input)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, settings)
}

func (s *Server) createAlertJiraIssue(w http.ResponseWriter, r *http.Request) {
	alertID := strings.TrimSpace(r.PathValue("id"))
	alerts := s.store.Alerts(200)
	var selected *store.Alert
	for _, alert := range alerts {
		if alert.ID == alertID {
			copy := alert
			selected = &copy
			break
		}
	}
	if selected == nil {
		httpx.Error(w, http.StatusNotFound, "alert not found")
		return
	}
	issue, existed, err := s.ensureJiraIssue(r.Context(), *selected, "manual")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"issue": issue, "existing": existed})
}

func (s *Server) createJiraIssuesForAlerts(ctx context.Context, alerts []store.Alert) {
	settings := s.store.JiraSettings()
	if !settings.Enabled || !settings.AutoCreateOnFiring {
		return
	}
	for _, alert := range alerts {
		if strings.ToLower(alert.Status) != "firing" {
			continue
		}
		if _, _, err := s.ensureJiraIssue(ctx, alert, "auto_firing"); err != nil {
			s.log.Warn("jira issue create failed", "alert", alert.Fingerprint, "error", err)
		}
	}
}

func (s *Server) ensureJiraIssue(ctx context.Context, alert store.Alert, rule string) (store.JiraIssue, bool, error) {
	if existing, ok := s.store.JiraIssueByFingerprint(alert.Fingerprint); ok {
		return existing, true, nil
	}
	settings := s.store.JiraSettings()
	if !settings.Enabled {
		return store.JiraIssue{}, false, fmt.Errorf("jira integration is disabled")
	}
	client := s.jira
	if settings.BaseURL != "" && settings.BaseURL != s.config.JiraBaseURL {
		client = jira.New(jira.Config{
			BaseURL:  settings.BaseURL,
			Email:    s.config.JiraEmail,
			APIToken: s.config.JiraAPIToken,
		})
	}
	if !client.Enabled() {
		return store.JiraIssue{}, false, fmt.Errorf("jira credentials are not configured")
	}
	assigneeEmail := s.grafanaEmailByLogin(ctx, alert.AssignedUserLogin)
	summary := renderAlertTemplate(settings.SummaryTemplate, alert)
	description := renderAlertTemplate(settings.DescriptionTemplate, alert)
	createdIssue, err := client.CreateIssue(ctx, jira.CreateIssueInput{
		ProjectKey:   settings.ProjectKey,
		IssueType:    settings.IssueType,
		Summary:      summary,
		Description:  description,
		Labels:       settings.Labels,
		AssigneeMail: assigneeEmail,
	})
	if err != nil {
		return store.JiraIssue{}, false, err
	}
	issue, err := s.store.CreateJiraIssue(store.JiraIssue{
		AlertID:       alert.ID,
		Fingerprint:   alert.Fingerprint,
		IssueKey:      createdIssue.Key,
		IssueURL:      createdIssue.URL,
		CreatedByRule: rule,
	})
	return issue, false, err
}

func (s *Server) notifyCriticalAlerts(ctx context.Context, alerts []store.Alert) {
	for _, alert := range alerts {
		if alert.Status != "firing" {
			continue
		}
		routes := s.store.AlertNotificationRoutes(alert)
		if len(routes) == 0 && s.tg.Enabled() && isCriticalSeverity(alert.Severity) {
			routes = append(routes, store.NotificationRoute{
				Channel: store.NotificationChannel{ChatID: s.tg.ChatIDFor(alert.AssignedUserLogin)},
				Receiver: store.NotificationReceiver{
					Kind:     "telegram",
					Name:     "env telegram fallback",
					BotToken: s.config.TelegramBotToken,
					Enabled:  true,
				},
			})
		}
		assignee := firstNonEmpty(alert.AssignedUserName, alert.AssignedUserLogin)
		text := notify.AlertMessage(
			alert.Status,
			alert.Severity,
			alert.AlertName,
			alert.Service,
			alert.Summary,
			assignee,
		)
		for _, route := range routes {
			switch route.Receiver.Kind {
			case "telegram":
				client := notify.NewTelegram(notify.TelegramConfig{BotToken: route.Receiver.BotToken})
				if err := client.Send(ctx, notify.Message{ChatID: route.Channel.ChatID, Text: text}); err != nil {
					s.log.Warn("telegram alert notification failed", "error", err, "channel", route.Channel.Name, "receiver", route.Receiver.Name)
				}
			case "lark":
				client := notify.NewLark(route.Receiver.WebhookURL)
				if err := client.Send(ctx, text); err != nil {
					s.log.Warn("lark alert notification failed", "error", err, "channel", route.Channel.Name, "receiver", route.Receiver.Name)
				}
			}
		}
	}
}

func (s *Server) notifyShiftCreated(ctx context.Context, shift store.Shift) {
	if !s.tg.Enabled() {
		return
	}
	err := s.tg.Send(ctx, notify.Message{
		ChatID: s.tg.ChatIDFor(shift.UserLogin),
		Text: notify.ShiftMessage(
			shift.UserName,
			shift.UserLogin,
			shift.StartsAt.Format(time.RFC3339),
			shift.EndsAt.Format(time.RFC3339),
		),
	})
	if err != nil {
		s.log.Warn("telegram shift notification failed", "error", err)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) fetchGrafanaUsers(ctx context.Context) ([]store.GrafanaUser, error) {
	apiURL, err := url.JoinPath(s.config.GrafanaURL, "/api/users")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"?perpage=1000", nil)
	if err != nil {
		return nil, err
	}

	if s.config.GrafanaServiceToken != "" {
		request.Header.Set("Authorization", "Bearer "+s.config.GrafanaServiceToken)
	} else if s.config.GrafanaBasicAuth != "" {
		login, password, ok := strings.Cut(s.config.GrafanaBasicAuth, ":")
		if ok {
			request.SetBasicAuth(login, password)
		}
	} else {
		return nil, fmt.Errorf("ONCALL_GRAFANA_SERVICE_TOKEN or ONCALL_GRAFANA_BASIC_AUTH is required")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("grafana api returned %s", response.Status)
	}

	var raw []struct {
		ID      int64  `json:"id"`
		Login   string `json:"login"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		IsAdmin bool   `json:"isAdmin"`
	}
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		return nil, err
	}

	users := make([]store.GrafanaUser, 0, len(raw))
	for _, user := range raw {
		role := "User"
		if user.IsAdmin {
			role = "Admin"
		}
		users = append(users, store.GrafanaUser{
			ID:    user.ID,
			Login: user.Login,
			Name:  user.Name,
			Email: user.Email,
			Role:  role,
		})
	}
	return users, nil
}

func isCriticalSeverity(severity string) bool {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "fatal":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *Server) grafanaEmailByLogin(ctx context.Context, login string) string {
	login = strings.TrimSpace(login)
	if login == "" {
		return ""
	}
	users, err := s.fetchGrafanaUsers(ctx)
	if err != nil {
		return ""
	}
	for _, user := range users {
		if strings.EqualFold(user.Login, login) {
			return strings.TrimSpace(user.Email)
		}
	}
	return ""
}

func renderAlertTemplate(template string, alert store.Alert) string {
	value := template
	replacer := strings.NewReplacer(
		"{{alertName}}", firstNonEmpty(alert.AlertName, alert.Fingerprint),
		"{{summary}}", firstNonEmpty(alert.Summary, alert.AlertName),
		"{{severity}}", firstNonEmpty(alert.Severity, "unknown"),
		"{{service}}", alert.Service,
		"{{status}}", alert.Status,
		"{{assignee}}", firstNonEmpty(alert.AssignedUserName, alert.AssignedUserLogin),
		"{{fingerprint}}", alert.Fingerprint,
	)
	value = replacer.Replace(value)
	if strings.TrimSpace(value) == "" {
		return firstNonEmpty(alert.Summary, alert.AlertName, alert.Fingerprint)
	}
	return value
}

func hasReceiverKind(receivers []store.NotificationReceiver, kind string) bool {
	for _, receiver := range receivers {
		if receiver.Enabled && receiver.Kind == kind {
			return true
		}
	}
	return false
}

func periodFromQuery(r *http.Request) (time.Time, time.Time) {
	now := time.Now()
	from := startOfDay(now)
	to := from.AddDate(0, 0, 7)

	if value := r.URL.Query().Get("from"); value != "" {
		if parsed, err := time.Parse(time.DateOnly, value); err == nil {
			from = parsed
		}
	}
	if value := r.URL.Query().Get("to"); value != "" {
		if parsed, err := time.Parse(time.DateOnly, value); err == nil {
			to = parsed.Add(24 * time.Hour)
		}
	}
	if !to.After(from) {
		to = from.AddDate(0, 0, 7)
	}
	return from, to
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func primaryOnCall(people []store.PersonReport) string {
	if len(people) == 0 {
		return "No active duty"
	}
	return people[0].UserName
}
