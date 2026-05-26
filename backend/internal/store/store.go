package store

import (
	"fmt"
	"sort"
	"sync"
	"time"
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
	mu        sync.Mutex
	schedules []Schedule
	shifts    []Shift
}

func New() *Store {
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
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Schedule, len(s.schedules))
	copy(result, s.schedules)
	return result
}

func (s *Store) Shifts(from, to time.Time) []Shift {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Shift, 0, len(s.shifts))
	for _, shift := range s.shifts {
		shift.Status = statusFor(shift.StartsAt, shift.EndsAt)
		if shift.EndsAt.After(from) && shift.StartsAt.Before(to) {
			result = append(result, shift)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartsAt.Before(result[j].StartsAt)
	})
	return result
}

func (s *Store) CreateShift(input CreateShift) (Shift, error) {
	if !input.EndsAt.After(input.StartsAt) {
		return Shift{}, fmt.Errorf("endsAt must be after startsAt")
	}
	if input.UserID == 0 || input.UserLogin == "" {
		return Shift{}, fmt.Errorf("grafana user is required")
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

	name := input.UserName
	if name == "" {
		name = input.UserLogin
	}

	shift := Shift{
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

	s.shifts = append(s.shifts, shift)
	return shift, nil
}

func (s *Store) Report(from, to time.Time) PeriodReport {
	shifts := s.Shifts(from, to)
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
