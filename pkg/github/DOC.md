# pkg/github package map

## Domain Summary
- Public GitHub integration SDK: wraps internal GitHub automation logic into reusable clients for apps, OAuth, webhooks, and token storage.
- Depends on `go-github` and `ghinstallation` to authenticate GitHub Apps and manage installations.

## Files (alphabetical)

### app.go
- **Purpose**: App-level client that caches installation transports, loads private keys, and creates GitHub REST clients per repo/org.
- **Key Types**: `AppClient`, `InstallationClient`, `AppConfig`.
- **Functions**: `NewAppClient`, `WithPrivateKey`, `GetInstallationClient`, `ListInstallations`, `InvalidateInstallation`.

### client.go
- **Purpose**: Helper constructors for generic GitHub REST clients with retry logic and custom user agents.
- **Functions**: `NewRESTClient`, `NewGraphQLClient`, `NewHTTPClient`, error translation utilities.

### oauth.go
- **Purpose**: OAuth flow helpers for CLI/TUI integrations (device flow, web-based flow, token exchange).
- **Key Types**: `OAuthClient`, `OAuthConfig`, `DeviceAuthSession`.
- **Functions**: `NewOAuthClient`, `StartDeviceFlow`, `PollDeviceFlow`, `ExchangeCode`, `RefreshToken`.

### token_storage.go
- **Purpose**: Local secure token storage (file-based with encryption/sanitization) used by CLI.
- **Key Types**: `TokenStore`, `TokenRecord`.
- **Functions**: `NewTokenStore`, `SaveToken`, `LoadToken`, `DeleteToken`, path helpers.

### webhook.go
- **Purpose**: GitHub webhook verification and event dispatch utilities.
- **Key Types**: `WebhookHandler`, `WebhookConfig`.
- **Functions**: `NewWebhookHandler`, `VerifySignature`, `HandleEvent`, event routing helpers.

## Subdirectories
- _None._
