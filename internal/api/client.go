package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Eventid3/umbraco-cli/internal/config"
)

const apiBase = "/umbraco/management/api/v1"

// Client is an authenticated HTTP client for the Umbraco Management API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// tokenResponse is the shape returned by the token endpoint.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewClient creates a new API client using the given profile,
// fetching a fresh Bearer token immediately.
func NewClient(p *config.Profile) (*Client, error) {
	transport := http.DefaultTransport
	if p.Insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // intentional for local dev
		}
	}
	c := &Client{
		baseURL: strings.TrimRight(p.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
	if err := c.fetchToken(p.ClientID, p.ClientSecret); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) fetchToken(clientID, clientSecret string) error {
	tokenURL := c.baseURL + apiBase + "/security/back-office/token"

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	resp, err := c.httpClient.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return fmt.Errorf("cannot parse token response: %w", err)
	}
	c.token = tr.AccessToken
	return nil
}

// Get performs an authenticated GET request and decodes the JSON response into dest.
func (c *Client) Get(path string, dest interface{}) error {
	return c.do(http.MethodGet, path, nil, dest)
}

// Post performs an authenticated POST request.
func (c *Client) Post(path string, body interface{}, dest interface{}) error {
	return c.do(http.MethodPost, path, body, dest)
}

// Put performs an authenticated PUT request.
func (c *Client) Put(path string, body interface{}, dest interface{}) error {
	return c.do(http.MethodPut, path, body, dest)
}

// Delete performs an authenticated DELETE request.
func (c *Client) Delete(path string) error {
	return c.do(http.MethodDelete, path, nil, nil)
}

func (c *Client) do(method, path string, body interface{}, dest interface{}) error {
	fullURL := c.baseURL + apiBase + path

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("cannot marshal request body: %w", err)
		}
		reqBody = strings.NewReader(string(b))
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to extract a structured error message.
		var apiErr struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		}
		_ = json.Unmarshal(respBody, &apiErr)
		if apiErr.Detail != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Detail)
		}
		if apiErr.Title != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Title)
		}
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return fmt.Errorf("cannot parse response: %w", err)
		}
	}
	return nil
}

// URL returns the full URL for an API path (useful for debugging).
func (c *Client) URL(path string) string {
	return c.baseURL + apiBase + path
}

// UploadTempFile uploads a local file as a temporary file and returns its ID.
// The temporary file ID can then be referenced when creating media items.
func (c *Client) UploadTempFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("cannot create form file: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", fmt.Errorf("cannot read file: %w", err)
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, c.baseURL+apiBase+"/temporary-file", &buf)
	if err != nil {
		return "", fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("temp file upload returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("cannot parse temp file response: %w", err)
	}
	return result.ID, nil
}
