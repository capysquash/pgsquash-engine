package github

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
)

// OAuthHandler handles GitHub OAuth flow
type OAuthHandler struct {
    clientID     string
    clientSecret string
    redirectURL  string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(clientID, clientSecret, redirectURL string) *OAuthHandler {
    return &OAuthHandler{
        clientID:     clientID,
        clientSecret: clientSecret,
        redirectURL:  redirectURL,
    }
}

// GetAuthorizationURL returns the GitHub OAuth authorization URL
func (h *OAuthHandler) GetAuthorizationURL(state string) string {
    params := url.Values{
        "client_id":    {h.clientID},
        "redirect_uri": {h.redirectURL},
        "scope":        {"repo,write:repo_hook"},
        "state":        {state},
    }

    return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func (h *OAuthHandler) ExchangeCodeForToken(ctx context.Context, code string) (string, error) {
    data := url.Values{
        "client_id":     {h.clientID},
        "client_secret": {h.clientSecret},
        "code":          {code},
        "redirect_uri":  {h.redirectURL},
    }

    req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token",
        strings.NewReader(data.Encode()))
    if err != nil {
        return "", err
    }

    req.Header.Set("Accept", "application/json")
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result struct {
        AccessToken string `json:"access_token"`
        TokenType   string `json:"token_type"`
        Scope       string `json:"scope"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }

    if result.AccessToken == "" {
        return "", fmt.Errorf("no access token received")
    }

    return result.AccessToken, nil
}

// HandleCallback handles the OAuth callback
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")

    if code == "" {
        http.Error(w, "No code provided", http.StatusBadRequest)
        return
    }

    // Verify state parameter for CSRF protection
    // In production, validate against stored state
    if state == "" {
        http.Error(w, "Invalid state", http.StatusBadRequest)
        return
    }

    token, err := h.ExchangeCodeForToken(r.Context(), code)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
        return
    }

    // Store token (implement your storage logic)
    // For now, just return success
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "token":  token,
    }); err != nil {
        http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
    }
}

// InstallationConfig represents GitHub App installation configuration
type InstallationConfig struct {
    AppID          int64
    PrivateKey     []byte
    WebhookSecret  string
    InstallationID int64
}

// AppAuthHandler handles GitHub App authentication
type AppAuthHandler struct {
    config InstallationConfig
}

// NewAppAuthHandler creates a new GitHub App auth handler
func NewAppAuthHandler(config InstallationConfig) *AppAuthHandler {
    return &AppAuthHandler{
        config: config,
    }
}

// GetInstallationToken generates an installation access token
// Note: This requires GitHub App JWT authentication (implementation simplified)
func (h *AppAuthHandler) GetInstallationToken(ctx context.Context) (string, error) {
    // In production, implement proper GitHub App JWT token generation
    // using the crypto/rsa package and the App's private key
    //
    // Steps:
    // 1. Generate JWT with App ID and current timestamp
    // 2. Sign with private key
    // 3. Use JWT to request installation access token
    // 4. Return installation access token

    return "", fmt.Errorf("GitHub App authentication not fully implemented - use OAuth token")
}
