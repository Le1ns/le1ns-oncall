package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	email   string
	token   string
	http    *http.Client
}

type Config struct {
	BaseURL  string
	Email    string
	APIToken string
}

type CreateIssueInput struct {
	ProjectKey   string
	IssueType    string
	Summary      string
	Description  string
	Labels       []string
	AssigneeMail string
}

type Issue struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

func New(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		email:   strings.TrimSpace(cfg.Email),
		token:   strings.TrimSpace(cfg.APIToken),
		http:    &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c.baseURL != "" && c.email != "" && c.token != ""
}

func (c *Client) CreateIssue(ctx context.Context, input CreateIssueInput) (Issue, error) {
	if !c.Enabled() {
		return Issue{}, fmt.Errorf("jira is not configured")
	}

	fields := map[string]any{
		"project":   map[string]string{"key": input.ProjectKey},
		"issuetype": map[string]string{"name": input.IssueType},
		"summary":   strings.TrimSpace(input.Summary),
		"description": map[string]any{
			"type":    "doc",
			"version": 1,
			"content": []any{
				map[string]any{
					"type": "paragraph",
					"content": []any{
						map[string]string{"type": "text", "text": strings.TrimSpace(input.Description)},
					},
				},
			},
		},
	}
	if len(input.Labels) > 0 {
		fields["labels"] = input.Labels
	}

	if email := strings.TrimSpace(input.AssigneeMail); email != "" {
		if accountID, err := c.lookupAccountID(ctx, email); err == nil && accountID != "" {
			fields["assignee"] = map[string]string{"id": accountID}
		}
	}

	requestBody, _ := json.Marshal(map[string]any{"fields": fields})
	endpoint, _ := url.JoinPath(c.baseURL, "/rest/api/3/issue")
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth(c.email, c.token)

	response, err := c.http.Do(request)
	if err != nil {
		return Issue{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return Issue{}, fmt.Errorf("jira create issue failed with %s", response.Status)
	}

	var payload struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Issue{}, err
	}
	if payload.Key == "" {
		return Issue{}, fmt.Errorf("jira returned empty issue key")
	}
	return Issue{
		Key: payload.Key,
		URL: c.baseURL + "/browse/" + payload.Key,
	}, nil
}

func (c *Client) lookupAccountID(ctx context.Context, email string) (string, error) {
	endpoint, _ := url.JoinPath(c.baseURL, "/rest/api/3/user/search")
	query := url.Values{}
	query.Set("query", email)

	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	request.SetBasicAuth(c.email, c.token)

	response, err := c.http.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("jira user search failed with %s", response.Status)
	}

	var users []struct {
		AccountID string `json:"accountId"`
		Email     string `json:"emailAddress"`
	}
	if err := json.NewDecoder(response.Body).Decode(&users); err != nil {
		return "", err
	}
	for _, user := range users {
		if strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(email)) {
			return user.AccountID, nil
		}
	}
	if len(users) > 0 {
		return users[0].AccountID, nil
	}
	return "", nil
}
