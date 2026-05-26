package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/local/oncall-plugin/backend/internal/config"
	"github.com/local/oncall-plugin/backend/internal/httpx"
	"github.com/local/oncall-plugin/backend/internal/store"
)

type Server struct {
	config config.Config
	log    *slog.Logger
	store  *store.Store
	mux    *http.ServeMux
}

func NewServer(cfg config.Config, log *slog.Logger, st *store.Store) *Server {
	s := &Server{
		config: cfg,
		log:    log,
		store:  st,
		mux:    http.NewServeMux(),
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
	httpx.JSON(w, http.StatusOK, map[string]any{
		"service": "oncall-api",
		"features": map[string]bool{
			"grafanaUsers": s.config.GrafanaServiceToken != "" || s.config.GrafanaBasicAuth != "",
			"jiraCloud":    s.config.JiraBaseURL != "" && s.config.JiraEmail != "" && s.config.JiraAPIToken != "",
			"telegram":     s.config.TelegramBotToken != "",
			"lark":         s.config.LarkWebhookURL != "",
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
			"to":   to.Format(time.DateOnly),
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

	httpx.JSON(w, http.StatusCreated, shift)
}

func (s *Server) alerts(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"alerts": []map[string]any{},
	})
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

	alertCount := 0
	if alerts, ok := payload["alerts"].([]any); ok {
		alertCount = len(alerts)
	}

	s.log.Info("received alertmanager webhook", "alerts", alertCount)
	httpx.JSON(w, http.StatusAccepted, map[string]any{
		"status": "accepted",
		"alerts": alertCount,
	})
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
