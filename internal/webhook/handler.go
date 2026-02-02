package webhook

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// Job represents a webhook processing job.
type Job struct {
	Webhook     *SentryWebhook
	ParsedError *ParsedError
}

// Handler handles incoming Sentry webhooks.
type Handler struct {
	jobQueue chan<- Job
}

// NewHandler creates a new webhook handler.
func NewHandler(jobQueue chan<- Job) *Handler {
	return &Handler{
		jobQueue: jobQueue,
	}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("failed to read webhook body: %v", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Parse webhook payload
	var webhook SentryWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		log.Printf("failed to parse webhook payload: %v", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	log.Printf("received webhook: action=%s, issue_id=%s", webhook.Action, webhook.Data.Issue.ID)
	// Only process error/issue events
	if webhook.Action != "created" && webhook.Action != "triggered" {
		log.Printf("ignoring webhook action: %s", webhook.Action)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Parse error information
	parsed := ParseWebhook(&webhook)
	if parsed == nil {
		log.Printf("webhook has no issue data")
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Queue job for async processing (non-blocking)
	select {
	case h.jobQueue <- Job{Webhook: &webhook, ParsedError: parsed}:
		log.Printf("queued job for issue %s (project: %s)", parsed.IssueID, parsed.ProjectSlug)
	default:
		log.Printf("job queue full, dropping webhook for issue %s", parsed.IssueID)
	}

	// Respond immediately (Sentry requires <1 second response)
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"queued"}`))
}

// HealthHandler returns a simple health check handler.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}
