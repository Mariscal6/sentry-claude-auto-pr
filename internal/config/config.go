package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// RepoMapping maps a Sentry project to a GitHub repository.
type RepoMapping struct {
	SentryProject string
	Owner         string
	Repo          string
}

// Config holds all application configuration.
type Config struct {
	Port                string
	SentryWebhookSecret string
	GitHubToken         string
	AnthropicAPIKey     string
	RepoMappings        []RepoMapping
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		SentryWebhookSecret: os.Getenv("SENTRY_WEBHOOK_SECRET"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		AnthropicAPIKey:     os.Getenv("ANTHROPIC_API_KEY"),
	}

	// Validate required fields
	if cfg.SentryWebhookSecret == "" {
		return nil, errors.New("SENTRY_WEBHOOK_SECRET is required")
	}
	if cfg.GitHubToken == "" {
		return nil, errors.New("GITHUB_TOKEN is required")
	}

	// Parse repo mappings
	// Format: sentry-project1:owner1/repo1,sentry-project2:owner2/repo2
	mappingsStr := os.Getenv("REPO_MAPPINGS")
	if mappingsStr == "" {
		return nil, errors.New("REPO_MAPPINGS is required")
	}

	mappings, err := parseRepoMappings(mappingsStr)
	if err != nil {
		return nil, err
	}
	cfg.RepoMappings = mappings

	return cfg, nil
}

// parseRepoMappings parses the REPO_MAPPINGS environment variable.
// Format: sentry-project1:owner1/repo1,sentry-project2:owner2/repo2
func parseRepoMappings(s string) ([]RepoMapping, error) {
	var mappings []RepoMapping

	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split sentry-project:owner/repo
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid repo mapping format: %q (expected sentry-project:owner/repo)", pair)
		}

		sentryProject := strings.TrimSpace(parts[0])
		repoPath := strings.TrimSpace(parts[1])

		// Split owner/repo
		repoParts := strings.SplitN(repoPath, "/", 2)
		if len(repoParts) != 2 {
			return nil, fmt.Errorf("invalid repo path format: %q (expected owner/repo)", repoPath)
		}

		mappings = append(mappings, RepoMapping{
			SentryProject: sentryProject,
			Owner:         strings.TrimSpace(repoParts[0]),
			Repo:          strings.TrimSpace(repoParts[1]),
		})
	}

	if len(mappings) == 0 {
		return nil, errors.New("REPO_MAPPINGS must contain at least one mapping")
	}

	return mappings, nil
}

// GetRepoMapping returns the repo mapping for a Sentry project, or nil if not found.
func (c *Config) GetRepoMapping(sentryProject string) *RepoMapping {
	for i := range c.RepoMappings {
		if c.RepoMappings[i].SentryProject == sentryProject {
			return &c.RepoMappings[i]
		}
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
