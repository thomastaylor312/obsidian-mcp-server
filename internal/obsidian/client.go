package obsidian

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/obsidian-mcp-server/obsidian-mcp-server/pkg/obsidian"
)

// Client wraps the generated API client with convenience methods
type Client struct {
	apiClient  *obsidian.Client
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Obsidian API client
func NewClient(apiToken, baseURL string) *Client {
	httpClient := &http.Client{}

	// Create the generated client
	apiClient, _ := obsidian.NewClient(baseURL,
		obsidian.WithHTTPClient(httpClient),
		obsidian.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiToken)
			return nil
		}))

	return &Client{
		apiClient:  apiClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiToken:   apiToken,
		httpClient: httpClient,
	}
}

// makeRequest makes an HTTP request to the Obsidian API
func (c *Client) makeRequest(method, path string, headers map[string]string, body io.Reader) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authorization header
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// GetServerInfo gets basic server information
func (c *Client) GetServerInfo() (string, error) {
	data, err := c.makeRequest("GET", "/", nil, nil)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

// ListVaultFiles lists files in the vault
func (c *Client) ListVaultFiles(path string) (string, error) {
	apiPath := "/vault/"
	if path != "" {
		apiPath = "/vault/" + strings.Trim(path, "/") + "/"
	}

	data, err := c.makeRequest("GET", apiPath, nil, nil)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

// GetFileContent gets the content of a specific file
func (c *Client) GetFileContent(filename, format string) (string, error) {
	apiPath := "/vault/" + strings.TrimPrefix(filename, "/")
	headers := make(map[string]string)

	if format == "json" {
		headers["Accept"] = "application/vnd.olrapi.note+json"
	}

	data, err := c.makeRequest("GET", apiPath, headers, nil)
	if err != nil {
		return "", err
	}

	if format == "json" {
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return "", fmt.Errorf("failed to parse JSON response: %w", err)
		}
		output, _ := json.MarshalIndent(result, "", "  ")
		return string(output), nil
	}

	return string(data), nil
}

// CreateOrUpdateFile creates or updates a file
func (c *Client) CreateOrUpdateFile(filename, content, contentType string) (string, error) {
	apiPath := "/vault/" + strings.TrimPrefix(filename, "/")
	headers := map[string]string{
		"Content-Type": contentType,
	}

	body := strings.NewReader(content)
	_, err := c.makeRequest("PUT", apiPath, headers, body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully created/updated file: %s", filename), nil
}

// AppendToFile appends content to a file
func (c *Client) AppendToFile(filename, content string) (string, error) {
	apiPath := "/vault/" + strings.TrimPrefix(filename, "/")
	headers := map[string]string{
		"Content-Type": "text/markdown",
	}

	body := strings.NewReader(content)
	_, err := c.makeRequest("POST", apiPath, headers, body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully appended to file: %s", filename), nil
}

// PatchFileContent patches content in a file
func (c *Client) PatchFileContent(filename, operation, targetType, target, content, contentType, delimiter string) (string, error) {
	apiPath := "/vault/" + strings.TrimPrefix(filename, "/")
	headers := map[string]string{
		"Content-Type":       contentType,
		"Operation":          operation,
		"Target-Type":        targetType,
		"Target":             url.QueryEscape(target),
		"Target-Delimiter":   delimiter,
	}

	body := strings.NewReader(content)
	_, err := c.makeRequest("PATCH", apiPath, headers, body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully patched file: %s (operation: %s, target: %s)", filename, operation, target), nil
}

// DeleteFile deletes a file
func (c *Client) DeleteFile(filename string) (string, error) {
	apiPath := "/vault/" + strings.TrimPrefix(filename, "/")

	_, err := c.makeRequest("DELETE", apiPath, nil, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully deleted file: %s", filename), nil
}

// SearchVaultSimple performs a simple text search
func (c *Client) SearchVaultSimple(query string, contextLength int) (string, error) {
	apiPath := "/search/simple/?query=" + url.QueryEscape(query)
	if contextLength > 0 {
		apiPath += "&contextLength=" + strconv.Itoa(contextLength)
	}

	data, err := c.makeRequest("POST", apiPath, nil, nil)
	if err != nil {
		return "", err
	}

	var results []interface{}
	if err := json.Unmarshal(data, &results); err != nil {
		return "", fmt.Errorf("failed to parse search results: %w", err)
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return string(output), nil
}

// SearchVaultAdvanced performs an advanced search
func (c *Client) SearchVaultAdvanced(query, queryType string) (string, error) {
	apiPath := "/search/"
	var contentType string
	var body io.Reader

	switch queryType {
	case "dataview":
		contentType = "application/vnd.olrapi.dataview.dql+txt"
		body = strings.NewReader(query)
	case "jsonlogic":
		contentType = "application/vnd.olrapi.jsonlogic+json"
		// Validate JSON
		var jsonQuery interface{}
		if err := json.Unmarshal([]byte(query), &jsonQuery); err != nil {
			return "", fmt.Errorf("invalid JSON query: %w", err)
		}
		body = strings.NewReader(query)
	default:
		return "", fmt.Errorf("unsupported query type: %s", queryType)
	}

	headers := map[string]string{
		"Content-Type": contentType,
	}

	data, err := c.makeRequest("POST", apiPath, headers, body)
	if err != nil {
		return "", err
	}

	var results []interface{}
	if err := json.Unmarshal(data, &results); err != nil {
		return "", fmt.Errorf("failed to parse search results: %w", err)
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return string(output), nil
}

// ListCommands gets available Obsidian commands
func (c *Client) ListCommands() (string, error) {
	data, err := c.makeRequest("GET", "/commands/", nil, nil)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

// ExecuteCommand executes a specific command
func (c *Client) ExecuteCommand(commandId string) (string, error) {
	apiPath := "/commands/" + url.PathEscape(commandId) + "/"

	_, err := c.makeRequest("POST", apiPath, nil, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully executed command: %s", commandId), nil
}

// OpenFile opens a file in Obsidian
func (c *Client) OpenFile(filename string, newLeaf bool) (string, error) {
	apiPath := "/open/" + strings.TrimPrefix(filename, "/")
	if newLeaf {
		apiPath += "?newLeaf=true"
	}

	_, err := c.makeRequest("POST", apiPath, nil, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully opened file: %s", filename), nil
}