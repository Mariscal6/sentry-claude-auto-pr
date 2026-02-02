# SentryAgent

Automatically generates pull requests to fix errors reported by Sentry using Claude Code.

## How It Works

1. Sentry detects an error and sends a webhook to SentryAgent
2. SentryAgent clones the repository and invokes Claude Code CLI
3. Claude Code analyzes the error, explores the codebase, and generates a fix
4. SentryAgent creates a pull request with the proposed fix

## Requirements

- Go 1.23+
- [Claude Code CLI](https://github.com/anthropics/claude-code) installed and in PATH
- GitHub token with repo permissions
- Sentry account with Internal Integration configured

## Installation

```bash
git clone https://github.com/maris/sentryagent.git
cd sentryagent
go build -o sentryagent ./cmd/server
```

## Configuration

Set the following environment variables:

```bash
# Required
SENTRY_WEBHOOK_SECRET=your-sentry-webhook-secret
GITHUB_TOKEN=ghp_your_github_token
REPO_MAPPINGS=sentry-project:owner/repo

# Optional
PORT=8080
ANTHROPIC_API_KEY=sk-ant-...  # If not set, uses 'claude login' auth
```

### Repository Mappings

Map Sentry projects to GitHub repositories:

```bash
# Single repo
REPO_MAPPINGS=my-project:myorg/myrepo

# Multiple repos
REPO_MAPPINGS=project1:org/repo1,project2:org/repo2
```

## Sentry Setup

1. Go to **Settings** → **Integrations** → **Internal Integrations**
2. Click **Create New Integration**
3. Configure:
   - **Name:** SentryAgent
   - **Webhook URL:** `https://your-server.com/webhook/sentry`
   - Enable **Alert Rule Action**
4. Set permissions:
   - **Issue & Event:** Read
   - **Project:** Read
5. Enable webhooks for `issue` events
6. Copy the **Client Secret** → use as `SENTRY_WEBHOOK_SECRET`
7. Save and install on your project(s)

### Create Alert Rule

1. Go to **Alerts** → **Create Alert Rule**
2. Choose **Issue Alert**
3. Set conditions (e.g., "A new issue is created")
4. Add action: **Send a notification via SentryAgent**
5. Save

## Usage

```bash
# Start the server
./sentryagent

# Or with environment variables inline
SENTRY_WEBHOOK_SECRET=secret \
GITHUB_TOKEN=ghp_token \
REPO_MAPPINGS=my-project:myorg/myrepo \
./sentryagent
```

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/webhook/sentry` | POST | Receives Sentry webhooks |
| `/health` | GET | Health check |

## Local Development

For local testing, use a tunnel to expose your server:

```bash
# Start ngrok
ngrok http 8080

# Use the ngrok URL as your Sentry webhook URL
# https://xxxx.ngrok.io/webhook/sentry
```

## Architecture

```
┌─────────┐     ┌─────────────┐     ┌─────────────┐     ┌────────┐
│  Sentry │────▶│ SentryAgent │────▶│ Claude Code │────▶│ GitHub │
└─────────┘     └─────────────┘     └─────────────┘     └────────┘
   webhook         server              CLI               PR
```

## License

MIT
