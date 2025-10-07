package github

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

    "gopkg.in/yaml.v3"
)

// Client is a GitHub API client
type Client struct {
    token      string
    httpClient *http.Client
    baseURL    string
}

// NewClient creates a new GitHub client
func NewClient(token string) *Client {
    return &Client{
        token:      token,
        httpClient: &http.Client{},
        baseURL:    "https://api.github.com",
    }
}

// PRFile represents a file in a pull request
type PRFile struct {
    Filename string `json:"filename"`
    SHA      string `json:"sha"`
    Status   string `json:"status"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
    Number int    `json:"number"`
    Head   Branch `json:"head"`
    Base   Branch `json:"base"`
}

// RepoConfig represents pg-squash configuration in a repository
type RepoConfig struct {
    AutoAnalyze        bool   `yaml:"auto_analyze"`
    AutoPR             bool   `yaml:"auto_pr"`
    MigrationThreshold int    `yaml:"migration_threshold"`
    SafetyLevel        string `yaml:"safety_level"`
}

// GetPRFiles retrieves files changed in a pull request
func (c *Client) GetPRFiles(ctx context.Context, repo string, prNumber int) ([]PRFile, error) {
    url := fmt.Sprintf("%s/repos/%s/pulls/%d/files", c.baseURL, repo, prNumber)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    c.setHeaders(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
    }

    var files []PRFile
    if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
        return nil, err
    }

    return files, nil
}

// GetPullRequest retrieves pull request details
func (c *Client) GetPullRequest(ctx context.Context, repo string, prNumber int) (*PullRequest, error) {
    url := fmt.Sprintf("%s/repos/%s/pulls/%d", c.baseURL, repo, prNumber)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    c.setHeaders(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
    }

    var pr PullRequest
    if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
        return nil, err
    }

    return &pr, nil
}

// GetFileContent retrieves the content of a file
func (c *Client) GetFileContent(ctx context.Context, repo, path, ref string) (string, error) {
    url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", c.baseURL, repo, path, ref)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return "", err
    }

    c.setHeaders(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("GitHub API error: %d", resp.StatusCode)
    }

    var fileData struct {
        Content  string `json:"content"`
        Encoding string `json:"encoding"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&fileData); err != nil {
        return "", err
    }

    if fileData.Encoding == "base64" {
        decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fileData.Content, "\n", ""))
        if err != nil {
            return "", err
        }
        return string(decoded), nil
    }

    return fileData.Content, nil
}

// PostPRComment posts a comment on a pull request
func (c *Client) PostPRComment(ctx context.Context, repo string, prNumber int, comment string) error {
    url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.baseURL, repo, prNumber)

    body := map[string]string{"body": comment}
    jsonBody, err := json.Marshal(body)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
    if err != nil {
        return err
    }

    c.setHeaders(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(respBody))
    }

    return nil
}

// GetRepoConfig retrieves pg-squash configuration from repository
func (c *Client) GetRepoConfig(ctx context.Context, repo, ref string) (*RepoConfig, error) {
    content, err := c.GetFileContent(ctx, repo, ".github/pgsquash.yml", ref)
    if err != nil {
        // Return default config if file doesn't exist
        return &RepoConfig{
            AutoAnalyze:        true,
            AutoPR:             false,
            MigrationThreshold: 15,
            SafetyLevel:        "standard",
        }, nil
    }

    var config RepoConfig
    if err := yaml.Unmarshal([]byte(content), &config); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    return &config, nil
}

// CreateBranch creates a new branch
func (c *Client) CreateBranch(ctx context.Context, repo, branch, sha string) error {
    url := fmt.Sprintf("%s/repos/%s/git/refs", c.baseURL, repo)

    body := map[string]string{
        "ref": "refs/heads/" + branch,
        "sha": sha,
    }
    jsonBody, err := json.Marshal(body)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
    if err != nil {
        return err
    }

    c.setHeaders(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("create branch failed: %d - %s", resp.StatusCode, string(respBody))
    }

    return nil
}

// CreateOrUpdateFile creates or updates a file in a repository
func (c *Client) CreateOrUpdateFile(ctx context.Context, repo, path, content, message, branch string) error {
    url := fmt.Sprintf("%s/repos/%s/contents/%s", c.baseURL, repo, path)

    // Check if file exists to get SHA
    var sha string
    existingReq, _ := http.NewRequestWithContext(ctx, "GET", url+"?ref="+branch, nil)
    c.setHeaders(existingReq)

    if existingResp, err := c.httpClient.Do(existingReq); err == nil {
        defer existingResp.Body.Close()
        if existingResp.StatusCode == http.StatusOK {
            var existingFile struct {
                SHA string `json:"sha"`
            }
            if err := json.NewDecoder(existingResp.Body).Decode(&existingFile); err == nil {
                sha = existingFile.SHA
            }
        }
    }

    body := map[string]string{
        "message": message,
        "content": base64.StdEncoding.EncodeToString([]byte(content)),
        "branch":  branch,
    }
    if sha != "" {
        body["sha"] = sha
    }

    jsonBody, err := json.Marshal(body)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(jsonBody)))
    if err != nil {
        return err
    }

    c.setHeaders(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("create/update file failed: %d - %s", resp.StatusCode, string(respBody))
    }

    return nil
}

// CreatePullRequest creates a new pull request
func (c *Client) CreatePullRequest(ctx context.Context, repo, title, body, head, base string) (int, error) {
    url := fmt.Sprintf("%s/repos/%s/pulls", c.baseURL, repo)

    requestBody := map[string]string{
        "title": title,
        "body":  body,
        "head":  head,
        "base":  base,
    }
    jsonBody, err := json.Marshal(requestBody)
    if err != nil {
        return 0, err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
    if err != nil {
        return 0, err
    }

    c.setHeaders(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        respBody, _ := io.ReadAll(resp.Body)
        return 0, fmt.Errorf("create PR failed: %d - %s", resp.StatusCode, string(respBody))
    }

    var pr struct {
        Number int `json:"number"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
        return 0, err
    }

    return pr.Number, nil
}

// setHeaders sets common headers for GitHub API requests
func (c *Client) setHeaders(req *http.Request) {
    req.Header.Set("Authorization", "Bearer "+c.token)
    req.Header.Set("Accept", "application/vnd.github+json")
    req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
    req.Header.Set("Content-Type", "application/json")
}
