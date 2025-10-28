// Package github provides GitHub App and OAuth integration for pgsquash.
package github

//
// This package exposes GitHub integration functionality for building API servers
// and tools that interact with GitHub repositories for automated migration analysis.
//
// # GitHub App Authentication
//
// GitHub Apps provide the most secure and scalable way to integrate with GitHub:
//
//	appClient, err := github.NewAppClientFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get client for specific repository
//	installClient, err := appClient.GetInstallationClientForRepo(ctx, "owner", "repo")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create check run on PR
//	checkRun := &github.CheckRun{
//	    Name:       "pgsquash/analysis",
//	    HeadSHA:    pr.HeadSHA,
//	    Status:     "completed",
//	    Conclusion: "success",
//	    Output: &github.CheckRunOutput{
//	        Title:   "Migration Analysis",
//	        Summary: "Analysis complete",
//	    },
//	}
//	run, err := installClient.CreateCheckRun(ctx, "owner", "repo", checkRun)
//
// # Webhook Handling
//
// Handle GitHub webhooks for automated PR analysis:
//
//	handler := github.NewWebhookHandlerWithApp(webhookSecret, appClient, engine)
//	http.HandleFunc("/github/webhook", func(w http.ResponseWriter, r *http.Request) {
//	    if err := handler.HandleWebhook(w, r); err != nil {
//	        log.Printf("Webhook error: %v", err)
//	    }
//	})
//
// # OAuth Flow
//
// For user authentication and personal access:
//
//	oauth := github.NewOAuthHandler(clientID, clientSecret, redirectURL)
//
//	// Generate authorization URL
//	state := generateRandomState()
//	authURL := oauth.GetAuthorizationURL(state)
//	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
//
//	// Handle callback
//	token, err := oauth.ExchangeCodeForToken(ctx, code)
//
// # Environment Variables
//
// GitHub App:
//   - GITHUB_APP_ID: GitHub App ID
//   - GITHUB_APP_PRIVATE_KEY: Private key (PEM format)
//   - GITHUB_APP_PRIVATE_KEY_PATH: Path to private key file (alternative)
//   - GITHUB_WEBHOOK_SECRET: Webhook secret for signature verification
//
// OAuth:
//   - GITHUB_CLIENT_ID: OAuth app client ID
//   - GITHUB_CLIENT_SECRET: OAuth app client secret
//   - GITHUB_REDIRECT_URL: OAuth callback URL

import (
	"context"
	"net/http"

	internal_github "github.com/CAPYSQUASH/pgsquash-engine/internal/github"
	go_github "github.com/google/go-github/v57/github"
)

// Re-export types from internal package
type (
	// AppClient wraps GitHub App authentication and provides multi-repository support
	AppClient = internal_github.AppClient

	// AppConfig holds GitHub App configuration
	AppConfig = internal_github.AppConfig

	// InstallationClient represents a GitHub App installation for a specific repository/organization
	InstallationClient = internal_github.InstallationClient

	// WebhookHandler processes GitHub webhook events and triggers migration analysis
	WebhookHandler = internal_github.WebhookHandler

	// OAuthHandler manages OAuth authentication flow for user access
	OAuthHandler = internal_github.OAuthHandler

	// Client wraps basic GitHub API client with personal access token
	Client = internal_github.Client
)

// Re-export GitHub types for convenience
type (
	CheckRun       = go_github.CheckRun
	CheckRunOutput = go_github.CheckRunOutput
	Installation   = go_github.Installation
	PullRequest    = go_github.PullRequest
	Repository     = go_github.Repository
)

// AppClient constructors
var (
	// NewAppClient creates a new GitHub App client with provided configuration
	NewAppClient = internal_github.NewAppClient

	// NewAppClientFromEnv creates a GitHub App client from environment variables:
	//   - GITHUB_APP_ID
	//   - GITHUB_APP_PRIVATE_KEY or GITHUB_APP_PRIVATE_KEY_PATH
	NewAppClientFromEnv = internal_github.NewAppClientFromEnv
)

// WebhookHandler constructors
var (
	// NewWebhookHandler creates a webhook handler with personal access token authentication
	NewWebhookHandler = internal_github.NewWebhookHandler

	// NewWebhookHandlerWithApp creates a webhook handler with GitHub App authentication (recommended)
	NewWebhookHandlerWithApp = internal_github.NewWebhookHandlerWithApp
)

// OAuthHandler constructors
var (
	// NewOAuthHandler creates a new OAuth handler for user authentication
	NewOAuthHandler = internal_github.NewOAuthHandler
)

// Client constructors
var (
	// NewClient creates a basic GitHub client with personal access token
	NewClient = internal_github.NewClient
)

// AppClient methods
// These are available on *AppClient instances:
//
//   GetInstallationClient(ctx, installationID) (*InstallationClient, error)
//   GetInstallationForRepo(ctx, owner, repo) (int64, error)
//   GetInstallationClientForRepo(ctx, owner, repo) (*InstallationClient, error)
//   ListInstallations(ctx) ([]*Installation, error)

// InstallationClient methods
// These are available on *InstallationClient instances:
//
//   CreateCheckRun(ctx, owner, repo, checkRun) (*CheckRun, error)
//   UpdateCheckRun(ctx, owner, repo, checkRunID, updates) (*CheckRun, error)
//   CreateIssueComment(ctx, owner, repo, issueNumber, comment) error
//   GetPullRequest(ctx, owner, repo, number) (*PullRequest, error)
//   ListPullRequestFiles(ctx, owner, repo, number) ([]*go_github.CommitFile, error)

// WebhookHandler methods
// These are available on *WebhookHandler instances:
//
//   HandleWebhook(w http.ResponseWriter, r *http.Request) error

// OAuthHandler methods
// These are available on *OAuthHandler instances:
//
//   GetAuthorizationURL(state) string
//   ExchangeCodeForToken(ctx, code) (string, error)
//   HandleCallback(w http.ResponseWriter, r *http.Request)
//   ValidateState(state) bool

// Example usage patterns

// ExampleAppAuthentication demonstrates GitHub App authentication
func ExampleAppAuthentication() {
	ctx := context.Background()

	// Create app client from environment
	appClient, err := NewAppClientFromEnv()
	if err != nil {
		panic(err)
	}

	// Get installation for specific repository
	installationID, err := appClient.GetInstallationForRepo(ctx, "owner", "repo")
	if err != nil {
		panic(err)
	}

	// Get authenticated client for that installation
	installClient, err := appClient.GetInstallationClient(ctx, installationID)
	if err != nil {
		panic(err)
	}

	// Create check run
	_ = installClient // Use for API calls
}

// ExampleWebhookHandler demonstrates webhook handling
func ExampleWebhookHandler() {
	// Create webhook handler
	// handler := NewWebhookHandlerWithApp(webhookSecret, appClient, engine)

	// Register HTTP handler
	http.HandleFunc("/github/webhook", func(w http.ResponseWriter, r *http.Request) {
		// handler.HandleWebhook(w, r)
	})
}

// ExampleOAuthFlow demonstrates OAuth authentication
func ExampleOAuthFlow() {
	// ctx := context.Background()

	// Create OAuth handler (configuration would come from environment or config)
	// oauth := NewOAuthHandler(clientID, clientSecret, redirectURL)

	// Generate authorization URL
	// state := generateRandomState()
	// authURL := oauth.GetAuthorizationURL(state)
	// http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)

	// Handle callback
	// code := r.URL.Query().Get("code")
	// token, err := oauth.ExchangeCodeForToken(ctx, code)
	// if err != nil {
	//     panic(err)
	// }
	// _ = token // Use token for API calls
}
