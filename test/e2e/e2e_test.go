//go:build e2e

package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E tests require a running Obsidian instance with the Local REST API plugin
// These tests interact with a real Obsidian server to verify end-to-end functionality

const (
	testTimeout    = 30 * time.Second
	obsidianURL    = "http://127.0.0.1:27123"
	testFileName   = "mcp-test-file.md"
	testContent    = "# MCP Test File\n\nThis is a test file created by the MCP server e2e tests."
	updatedContent = "# Updated MCP Test File\n\nThis file has been updated by the e2e tests."
)

// TestE2ESetup verifies the test environment is properly configured
func TestE2ESetup(t *testing.T) {
	apiToken := os.Getenv("OBSIDIAN_API_TOKEN")
	require.NotEmpty(t, apiToken, "OBSIDIAN_API_TOKEN environment variable must be set")

	t.Logf("Using Obsidian API Token: %s...", apiToken[:min(8, len(apiToken))])
	t.Logf("Testing against Obsidian server at: %s", obsidianURL)
}

// mcpRequest represents a JSON-RPC request to the MCP server
type mcpRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// mcpResponse represents a JSON-RPC response from the MCP server
type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
}

type mcpError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// startMCPServer starts the MCP server and returns a function to stop it
func startMCPServer(t *testing.T) (*exec.Cmd, io.WriteCloser, io.ReadCloser) {
	apiToken := os.Getenv("OBSIDIAN_API_TOKEN")
	require.NotEmpty(t, apiToken)

	// Binary should already be built by the Makefile dependency
	binaryPath := "../../bin/obsidian-mcp-server"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary not found at %s. Make sure 'make build' was run.", binaryPath)
	}

	// Start the server
	cmd := exec.Command(binaryPath, "-token", apiToken, "-url", obsidianURL)

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	return cmd, stdin, stdout
}

// sendMCPRequest sends a request to the MCP server and returns the response
func sendMCPRequest(t *testing.T, stdin io.WriteCloser, stdout io.ReadCloser, req *mcpRequest) *mcpResponse {
	// Send request
	requestData, err := json.Marshal(req)
	require.NoError(t, err)

	_, err = stdin.Write(append(requestData, '\n'))
	require.NoError(t, err)

	// Read response line by line to handle complete JSON responses
	scanner := bufio.NewScanner(stdout)
	scanner.Scan()
	responseData := strings.TrimSpace(scanner.Text())

	if err := scanner.Err(); err != nil {
		require.NoError(t, err, "Error reading response")
	}

	if responseData == "" {
		t.Fatal("Received empty response from server")
	}

	var resp mcpResponse
	err = json.Unmarshal([]byte(responseData), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v\nRaw response: %q", err, responseData)
	}

	return &resp
}

// TestE2EInitialize tests the MCP server initialization
func TestE2EInitialize(t *testing.T) {
	cmd, stdin, stdout := startMCPServer(t)
	defer func() {
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	req := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}

	resp := sendMCPRequest(t, stdin, stdout, req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, "test-init", resp.ID)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	result := resp.Result.(map[string]interface{})
	assert.Equal(t, "2024-11-05", result["protocolVersion"])
	assert.Contains(t, result, "capabilities")
	assert.Contains(t, result, "serverInfo")
}

// TestE2EToolsList tests listing available tools
func TestE2EToolsList(t *testing.T) {
	cmd, stdin, stdout := startMCPServer(t)
	defer func() {
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Initialize first
	initReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	sendMCPRequest(t, stdin, stdout, initReq)

	// List tools
	req := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-tools-list",
		Method:  "tools/list",
	}

	resp := sendMCPRequest(t, stdin, stdout, req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, "test-tools-list", resp.ID)
	assert.Nil(t, resp.Error)

	result := resp.Result.(map[string]interface{})
	tools := result["tools"].([]interface{})
	assert.Greater(t, len(tools), 0)

	// Verify expected tools are present
	toolNames := make([]string, 0)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		toolNames = append(toolNames, toolMap["name"].(string))
	}

	expectedTools := []string{
		"get_server_info",
		"list_vault_files",
		"get_file_content",
		"create_or_update_file",
		"delete_file",
	}

	for _, expectedTool := range expectedTools {
		assert.Contains(t, toolNames, expectedTool)
	}
}

// TestE2EGetServerInfo tests getting server information
func TestE2EGetServerInfo(t *testing.T) {
	cmd, stdin, stdout := startMCPServer(t)
	defer func() {
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Initialize
	initReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	sendMCPRequest(t, stdin, stdout, initReq)

	// Get server info
	req := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-server-info",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "get_server_info",
			"arguments": map[string]interface{}{},
		},
	}

	resp := sendMCPRequest(t, stdin, stdout, req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, "test-server-info", resp.ID)
	assert.Nil(t, resp.Error)

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	assert.Len(t, content, 1)

	contentItem := content[0].(map[string]interface{})
	assert.Equal(t, "text", contentItem["type"])

	text := contentItem["text"].(string)
	assert.Contains(t, text, "Obsidian Local REST API")
	assert.Contains(t, text, "authenticated")
}

// TestE2EFileOperations tests comprehensive file operations
func TestE2EFileOperations(t *testing.T) {
	cmd, stdin, stdout := startMCPServer(t)
	defer func() {
		// Clean up test file first (before closing pipes)
		cleanupReq := &mcpRequest{
			JSONRPC: "2.0",
			ID:      "cleanup",
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name": "delete_file",
				"arguments": map[string]interface{}{
					"filename": testFileName,
				},
			},
		}

		// Try to clean up the test file, but don't fail the test if it errors
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Log cleanup failure but don't fail the test
					fmt.Printf("Cleanup failed (this is OK): %v\n", r)
				}
			}()

			requestData, err := json.Marshal(cleanupReq)
			if err == nil {
				stdin.Write(append(requestData, '\n'))
			}
		}()

		// Now close pipes and kill process
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Initialize
	initReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	sendMCPRequest(t, stdin, stdout, initReq)

	// 1. Create a test file
	createReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-create",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "create_or_update_file",
			"arguments": map[string]interface{}{
				"filename": testFileName,
				"content":  testContent,
			},
		},
	}

	resp := sendMCPRequest(t, stdin, stdout, createReq)
	assert.Nil(t, resp.Error, "Failed to create file: %v", resp.Error)

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	contentItem := content[0].(map[string]interface{})
	assert.Contains(t, contentItem["text"], "Successfully created/updated file")

	// 2. Verify the file was created by reading it
	readReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-read",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "get_file_content",
			"arguments": map[string]interface{}{
				"filename": testFileName,
				"format":   "markdown",
			},
		},
	}

	resp = sendMCPRequest(t, stdin, stdout, readReq)
	assert.Nil(t, resp.Error, "Failed to read file: %v", resp.Error)

	result = resp.Result.(map[string]interface{})
	content = result["content"].([]interface{})
	contentItem = content[0].(map[string]interface{})
	assert.Contains(t, contentItem["text"], "MCP Test File")

	// 3. Update the file
	updateReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-update",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "create_or_update_file",
			"arguments": map[string]interface{}{
				"filename": testFileName,
				"content":  updatedContent,
			},
		},
	}

	resp = sendMCPRequest(t, stdin, stdout, updateReq)
	assert.Nil(t, resp.Error, "Failed to update file: %v", resp.Error)

	// 4. Verify the update
	resp = sendMCPRequest(t, stdin, stdout, readReq)
	assert.Nil(t, resp.Error, "Failed to read updated file: %v", resp.Error)

	result = resp.Result.(map[string]interface{})
	content = result["content"].([]interface{})
	contentItem = content[0].(map[string]interface{})
	assert.Contains(t, contentItem["text"], "Updated MCP Test File")

	// 5. Append to the file
	appendReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-append",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "append_to_file",
			"arguments": map[string]interface{}{
				"filename": testFileName,
				"content":  "\n\nAppended content from e2e test.",
			},
		},
	}

	resp = sendMCPRequest(t, stdin, stdout, appendReq)
	assert.Nil(t, resp.Error, "Failed to append to file: %v", resp.Error)

	// 6. Verify the append
	resp = sendMCPRequest(t, stdin, stdout, readReq)
	assert.Nil(t, resp.Error, "Failed to read file after append: %v", resp.Error)

	result = resp.Result.(map[string]interface{})
	content = result["content"].([]interface{})
	contentItem = content[0].(map[string]interface{})
	text := contentItem["text"].(string)
	assert.Contains(t, text, "Updated MCP Test File")
	assert.Contains(t, text, "Appended content from e2e test")

	// 7. Delete the file
	deleteReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-delete",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "delete_file",
			"arguments": map[string]interface{}{
				"filename": testFileName,
			},
		},
	}

	resp = sendMCPRequest(t, stdin, stdout, deleteReq)
	assert.Nil(t, resp.Error, "Failed to delete file: %v", resp.Error)

	result = resp.Result.(map[string]interface{})
	content = result["content"].([]interface{})
	contentItem = content[0].(map[string]interface{})
	assert.Contains(t, contentItem["text"], "Successfully deleted file")
}

// TestE2EListVaultFiles tests listing vault files
func TestE2EListVaultFiles(t *testing.T) {
	cmd, stdin, stdout := startMCPServer(t)
	defer func() {
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Initialize
	initReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	sendMCPRequest(t, stdin, stdout, initReq)

	// List vault files
	req := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-list-files",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "list_vault_files",
			"arguments": map[string]interface{}{},
		},
	}

	resp := sendMCPRequest(t, stdin, stdout, req)
	assert.Nil(t, resp.Error, "Failed to list vault files: %v", resp.Error)

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	contentItem := content[0].(map[string]interface{})

	text := contentItem["text"].(string)
	assert.Contains(t, text, "files")

	// Parse the JSON response to verify it's valid
	var filesResult map[string]interface{}
	err := json.Unmarshal([]byte(text), &filesResult)
	require.NoError(t, err)

	files, ok := filesResult["files"]
	assert.True(t, ok)
	assert.NotNil(t, files)
}

// TestE2ESearchVault tests vault search functionality
func TestE2ESearchVault(t *testing.T) {
	cmd, stdin, stdout := startMCPServer(t)
	defer func() {
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Initialize
	initReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	sendMCPRequest(t, stdin, stdout, initReq)

	// Search vault with a simple query
	req := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-search",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "search_vault_simple",
			"arguments": map[string]interface{}{
				"query":         "note",
				"contextLength": 50,
			},
		},
	}

	resp := sendMCPRequest(t, stdin, stdout, req)

	// Note: Search might return empty results if no files contain "note",
	// but the request should still succeed
	assert.Nil(t, resp.Error, "Search request failed: %v", resp.Error)

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	contentItem := content[0].(map[string]interface{})

	text := contentItem["text"].(string)
	// Verify it's valid JSON (even if empty array)
	var searchResults []interface{}
	err := json.Unmarshal([]byte(text), &searchResults)
	require.NoError(t, err, "Search results should be valid JSON")
}

// TestE2EListCommands tests listing Obsidian commands
func TestE2EListCommands(t *testing.T) {
	cmd, stdin, stdout := startMCPServer(t)
	defer func() {
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Initialize
	initReq := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	sendMCPRequest(t, stdin, stdout, initReq)

	// List commands
	req := &mcpRequest{
		JSONRPC: "2.0",
		ID:      "test-list-commands",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "list_commands",
			"arguments": map[string]interface{}{},
		},
	}

	resp := sendMCPRequest(t, stdin, stdout, req)
	assert.Nil(t, resp.Error, "Failed to list commands: %v", resp.Error)

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]interface{})
	contentItem := content[0].(map[string]interface{})

	text := contentItem["text"].(string)
	assert.Contains(t, text, "commands")

	// Parse to verify it's valid JSON
	var commandsResult map[string]interface{}
	err := json.Unmarshal([]byte(text), &commandsResult)
	require.NoError(t, err)

	commands, ok := commandsResult["commands"]
	assert.True(t, ok)
	assert.NotNil(t, commands)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
