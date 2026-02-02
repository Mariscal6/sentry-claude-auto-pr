package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mariscal6/sentry-claude-auto-pr/internal/agent"
	"github.com/Mariscal6/sentry-claude-auto-pr/internal/config"
	"github.com/Mariscal6/sentry-claude-auto-pr/internal/gitprovider"
	"github.com/Mariscal6/sentry-claude-auto-pr/internal/webhook"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Log configured repo mappings
	log.Printf("Configured %d repo mapping(s):", len(cfg.RepoMappings))
	for _, m := range cfg.RepoMappings {
		log.Printf("  %s -> %s/%s", m.SentryProject, m.Owner, m.Repo)
	}

	// Create agent pipeline (uses Claude Code internally)
	pipeline := agent.NewPipeline(cfg.AnthropicAPIKey)

	// Create job queue for async webhook processing
	jobQueue := make(chan webhook.Job, 100)

	// Start job processor
	go processJobs(ctx, jobQueue, cfg, pipeline)

	// Set up HTTP server
	mux := http.NewServeMux()

	// Webhook endpoint with signature verification
	signatureVerifier := webhook.NewSignatureVerifier(cfg.SentryWebhookSecret)
	webhookHandler := webhook.NewHandler(jobQueue)
	mux.Handle("/webhook/sentry", signatureVerifier.Middleware(webhookHandler))

	// Health check
	mux.HandleFunc("/health", webhook.HealthHandler())

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting server on :%s", cfg.Port)
	log.Println("Endpoints:")
	log.Println("  POST /webhook/sentry - Sentry webhook endpoint")
	log.Println("  GET /health - Health check")
	log.Println("")
	log.Println("Note: This service uses Claude Code CLI for fix generation.")
	log.Println("Ensure 'claude' is installed and available in PATH.")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}

// processJobs processes webhook jobs from the queue.
func processJobs(ctx context.Context, jobs <-chan webhook.Job, cfg *config.Config, pipeline *agent.Pipeline) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-jobs:
			processJob(ctx, job, cfg, pipeline)
		}
	}
}

// processJob handles a single webhook job.
func processJob(ctx context.Context, job webhook.Job, cfg *config.Config, pipeline *agent.Pipeline) {
	log.Printf("Processing job for issue %s (project: %s)", job.ParsedError.IssueID, job.ParsedError.ProjectSlug)

	// Look up repository configuration
	repoMapping := cfg.GetRepoMapping(job.ParsedError.ProjectSlug)
	if repoMapping == nil {
		log.Printf("No repo mapping found for project %s, skipping", job.ParsedError.ProjectSlug)
		return
	}

	// Build repo URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", repoMapping.Owner, repoMapping.Repo)

	// Run the agent pipeline (uses Claude Code)
	fix, err := pipeline.Run(ctx, repoURL, cfg.GitHubToken, job.ParsedError)
	if err != nil {
		log.Printf("Pipeline failed for issue %s: %v", job.ParsedError.IssueID, err)
		return
	}

	// Create GitHub provider for PR creation
	provider := gitprovider.NewGitHubProvider(cfg.GitHubToken, repoMapping.Owner, repoMapping.Repo)

	// Create PR with the fix
	prURL, err := agent.CreatePullRequest(ctx, provider, job.ParsedError, fix)
	if err != nil {
		log.Printf("Failed to create PR for issue %s: %v", job.ParsedError.IssueID, err)
		return
	}

	log.Printf("Created PR for issue %s: %s", job.ParsedError.IssueID, prURL)
}
