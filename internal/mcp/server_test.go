package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPServerInitialization tests server initialization
func TestMCPServerInitialization(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")
	assert.NotNil(t, server)
	assert.NotNil(t, server.obsidianClient)
}

// TestHandleInitialize tests the initialize request handling
func TestHandleInitialize(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-1",
		Method:  "initialize",
		Params:  make(map[string]any),
	}

	response := server.handleRequest(request)
	require.NotNil(t, response)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, "test-1", response.ID)
	assert.Nil(t, response.Error)

	result, ok := response.Result.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "2024-11-05", result["protocolVersion"])
	assert.Contains(t, result, "capabilities")
	assert.Contains(t, result, "serverInfo")

	serverInfo, ok := result["serverInfo"].(ServerInfo)
	require.True(t, ok)
	assert.Equal(t, "obsidian-mcp-server", serverInfo.Name)
	assert.Equal(t, "1.0.0", serverInfo.Version)
}

// TestHandleToolsList tests the tools/list request handling
func TestHandleToolsList(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-2",
		Method:  "tools/list",
	}

	response := server.handleRequest(request)
	require.NotNil(t, response)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, "test-2", response.ID)
	assert.Nil(t, response.Error)

	result, ok := response.Result.(map[string]any)
	require.True(t, ok)

	tools, ok := result["tools"].([]ToolInfo)
	require.True(t, ok)
	assert.Greater(t, len(tools), 0)

	// Check that we have expected tools
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name
	}

	expectedTools := []string{
		"get_server_info",
		"list_vault_files",
		"get_file_content",
		"create_or_update_file",
		"append_to_file",
		"patch_file_content",
		"delete_file",
		"search_vault_simple",
		"search_vault_advanced",
		"list_commands",
		"execute_command",
		"open_file",
	}

	for _, expectedTool := range expectedTools {
		assert.Contains(t, toolNames, expectedTool)
	}
}

// TestHandleToolsCallMissingTool tests calling a non-existent tool
func TestHandleToolsCallMissingTool(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-5",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "nonexistent_tool",
			"arguments": map[string]any{},
		},
	}

	response := server.handleRequest(request)
	require.NotNil(t, response)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, "test-5", response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, -32603, response.Error.Code)
	assert.Contains(t, response.Error.Message, "unknown tool")
}

// TestHandleToolsCallMissingParameters tests calling a tool with missing required parameters
func TestHandleToolsCallMissingParameters(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-6",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "get_file_content",
			"arguments": map[string]any{
				// Missing required "filename" parameter
			},
		},
	}

	response := server.handleRequest(request)
	require.NotNil(t, response)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, "test-6", response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, -32603, response.Error.Code)
	assert.Contains(t, response.Error.Message, "filename is required")
}

// TestHandlePing tests ping request handling
func TestHandlePing(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-ping",
		Method:  "ping",
	}

	response := server.handleRequest(request)
	require.NotNil(t, response)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, "test-ping", response.ID)
	assert.Nil(t, response.Error)

	result, ok := response.Result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{}, result)
}

// TestHandleUnknownMethod tests handling of unknown methods
func TestHandleUnknownMethod(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-unknown",
		Method:  "unknown_method",
	}

	response := server.handleRequest(request)
	require.NotNil(t, response)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, "test-unknown", response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, -32601, response.Error.Code)
	assert.Equal(t, "Method not found", response.Error.Message)
	assert.Equal(t, "unknown_method", response.Error.Data)
}

// TestJSONSerialization tests that responses can be properly serialized to JSON
func TestJSONSerialization(t *testing.T) {
	server := NewMCPServer("test-token", "http://localhost:27123")

	request := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-json",
		Method:  "initialize",
	}

	response := server.handleRequest(request)
	require.NotNil(t, response)

	// Test that the response can be marshaled to JSON
	data, err := json.Marshal(response)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test that it can be unmarshaled back
	var unmarshaled MCPResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, response.JSONRPC, unmarshaled.JSONRPC)
	assert.Equal(t, response.ID, unmarshaled.ID)
}

// TestSendResponse tests response sending functionality
func TestSendResponse(t *testing.T) {
	var output bytes.Buffer
	server := NewMCPServer("test-token", "http://localhost:27123")
	server.stdout = &output

	response := &MCPResponse{
		JSONRPC: "2.0",
		ID:      "test",
		Result:  map[string]any{"status": "ok"},
	}

	err := server.sendResponse(response)
	require.NoError(t, err)

	outputStr := output.String()
	assert.NotEmpty(t, outputStr)
	assert.True(t, strings.HasSuffix(outputStr, "\n"))

	// Verify it's valid JSON
	var parsed MCPResponse
	err = json.Unmarshal([]byte(strings.TrimSpace(outputStr)), &parsed)
	require.NoError(t, err)
	assert.Equal(t, response.JSONRPC, parsed.JSONRPC)
	assert.Equal(t, response.ID, parsed.ID)
}
