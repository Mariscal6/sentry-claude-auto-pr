package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandler_ServeHTTP(t *testing.T) {
	jobQueue := make(chan Job, 10)
	handler := NewHandler(jobQueue)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantJob    bool
	}{
		{
			name:       "valid created webhook",
			method:     http.MethodPost,
			body:       validWebhookPayload("created"),
			wantStatus: http.StatusAccepted,
			wantJob:    true,
		},
		{
			name:       "valid triggered webhook",
			method:     http.MethodPost,
			body:       validWebhookPayload("triggered"),
			wantStatus: http.StatusAccepted,
			wantJob:    true,
		},
		{
			name:       "resolved webhook ignored",
			method:     http.MethodPost,
			body:       validWebhookPayload("resolved"),
			wantStatus: http.StatusAccepted,
			wantJob:    false,
		},
		{
			name:       "GET method not allowed",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
			wantJob:    false,
		},
		{
			name:       "invalid JSON",
			method:     http.MethodPost,
			body:       "not valid json",
			wantStatus: http.StatusBadRequest,
			wantJob:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Drain any existing jobs
			select {
			case <-jobQueue:
			default:
			}

			req := httptest.NewRequest(tt.method, "/webhook/sentry", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.wantStatus)
			}

			// Check if job was queued
			select {
			case job := <-jobQueue:
				if !tt.wantJob {
					t.Errorf("unexpected job queued: %v", job)
				}
			case <-time.After(10 * time.Millisecond):
				if tt.wantJob {
					t.Error("expected job to be queued, but none received")
				}
			}
		})
	}
}

func TestParseWebhook(t *testing.T) {
	webhook := &SentryWebhook{
		Action: "created",
		Data: WebhookData{
			Issue: &Issue{
				ID:      "12345",
				ShortID: "PROJ-1",
				Title:   "NullPointerException in Handler",
				Culprit: "com.example.Handler.handle",
				Level:   "error",
				Platform: "java",
				Project: Project{
					Slug: "my-project",
				},
				Metadata: Metadata{
					Type:  "NullPointerException",
					Value: "null reference at line 42",
				},
			},
		},
	}

	parsed := ParseWebhook(webhook)

	if parsed == nil {
		t.Fatal("ParseWebhook returned nil")
	}

	if parsed.IssueID != "12345" {
		t.Errorf("IssueID = %q, want %q", parsed.IssueID, "12345")
	}

	if parsed.ProjectSlug != "my-project" {
		t.Errorf("ProjectSlug = %q, want %q", parsed.ProjectSlug, "my-project")
	}

	if parsed.ErrorType != "NullPointerException" {
		t.Errorf("ErrorType = %q, want %q", parsed.ErrorType, "NullPointerException")
	}

	if parsed.Level != "error" {
		t.Errorf("Level = %q, want %q", parsed.Level, "error")
	}
}

func TestParseWebhook_NilIssue(t *testing.T) {
	webhook := &SentryWebhook{
		Action: "created",
		Data:   WebhookData{},
	}

	parsed := ParseWebhook(webhook)

	if parsed != nil {
		t.Errorf("expected nil for webhook without issue, got %v", parsed)
	}
}

func validWebhookPayload(action string) string {
	webhook := SentryWebhook{
		Action: action,
		Data: WebhookData{
			Issue: &Issue{
				ID:      "12345",
				ShortID: "PROJ-1",
				Title:   "Test Error",
				Level:   "error",
				Project: Project{
					Slug: "test-project",
				},
			},
		},
	}

	data, _ := json.Marshal(webhook)
	return string(data)
}
