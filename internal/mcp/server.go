package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/obsidian-mcp-server/obsidian-mcp-server/internal/obsidian"
)

// MCPServer represents the MCP server instance
type MCPServer struct {
	obsidianClient *obsidian.Client
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(apiToken, baseURL string) *MCPServer {
	client := obsidian.NewClient(apiToken, baseURL)
	return &MCPServer{
		obsidianClient: client,
		stdin:          os.Stdin,
		stdout:         os.Stdout,
		stderr:         os.Stderr,
	}
}

// MCPRequest represents an incoming MCP request
type MCPRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
}

// MCPError represents an MCP error response
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ToolInfo represents information about available tools
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities represents server capabilities
type Capabilities struct {
	Tools     any `json:"tools,omitempty"`
	Logging   any `json:"logging,omitempty"`
	Prompts   any `json:"prompts,omitempty"`
	Resources any `json:"resources,omitempty"`
	Sampling  any `json:"sampling,omitempty"`
}

// Run starts the MCP server and handles requests
func (s *MCPServer) Run() error {
	decoder := json.NewDecoder(s.stdin)

	for {
		var request MCPRequest
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				break
			}
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		response := s.handleRequest(&request)
		if response != nil {
			if err := s.sendResponse(response); err != nil {
				return fmt.Errorf("failed to send response: %w", err)
			}
		}
	}

	return nil
}

// handleRequest processes incoming MCP requests
func (s *MCPServer) handleRequest(request *MCPRequest) *MCPResponse {
	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "tools/list":
		return s.handleToolsList(request)
	case "tools/call":
		return s.handleToolsCall(request)
	case "ping":
		return s.handlePing(request)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
				Data:    request.Method,
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *MCPServer) handleInitialize(request *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": Capabilities{
				Tools: map[string]any{},
			},
			"serverInfo": ServerInfo{
				Name:    "obsidian-mcp-server",
				Version: "1.0.0",
			},
		},
	}
}

// handlePing handles ping requests
func (s *MCPServer) handlePing(request *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  map[string]any{},
	}
}

// handleToolsList returns the list of available tools
func (s *MCPServer) handleToolsList(request *MCPRequest) *MCPResponse {
	tools := []ToolInfo{
		{
			Name:        "get_server_info",
			Description: "Get basic server details and authentication status from Obsidian",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "list_vault_files",
			Description: "List files in the vault root or a specific directory",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Directory path relative to vault root (optional, defaults to root)",
					},
				},
			},
		},
		{
			Name:        "get_file_content",
			Description: "Get the content of a specific file, supports both markdown and JSON format",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file relative to vault root",
					},
					"format": map[string]any{
						"type":        "string",
						"description": "Response format: 'markdown' (default) or 'json' (includes metadata)",
						"enum":        []string{"markdown", "json"},
					},
				},
				"required": []string{"filename"},
			},
		},
		{
			Name:        "create_or_update_file",
			Description: "Create a new file or update an existing one",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file relative to vault root",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to write to the file",
					},
					"contentType": map[string]any{
						"type":        "string",
						"description": "Content type (defaults to 'text/markdown')",
					},
				},
				"required": []string{"filename", "content"},
			},
		},
		{
			Name:        "append_to_file",
			Description: "Append content to the end of an existing file",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file relative to vault root",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to append to the file",
					},
				},
				"required": []string{"filename", "content"},
			},
		},
		{
			Name:        "patch_file_content",
			Description: "Insert content relative to headings, blocks, or frontmatter fields",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file relative to vault root",
					},
					"operation": map[string]any{
						"type":        "string",
						"description": "Patch operation to perform",
						"enum":        []string{"append", "prepend", "replace"},
					},
					"targetType": map[string]any{
						"type":        "string",
						"description": "Type of target to patch",
						"enum":        []string{"heading", "block", "frontmatter"},
					},
					"target": map[string]any{
						"type":        "string",
						"description": "Target to patch (heading path, block ID, or frontmatter field)",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to insert",
					},
					"contentType": map[string]any{
						"type":        "string",
						"description": "Content type (defaults to 'text/markdown')",
					},
					"delimiter": map[string]any{
						"type":        "string",
						"description": "Delimiter for nested targets (defaults to '::')",
					},
				},
				"required": []string{"filename", "operation", "targetType", "target", "content"},
			},
		},
		{
			Name:        "delete_file",
			Description: "Delete a specific file from the vault",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file relative to vault root",
					},
				},
				"required": []string{"filename"},
			},
		},
		{
			Name:        "search_vault_simple",
			Description: "Simple text search across the vault",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
					"contextLength": map[string]any{
						"type":        "integer",
						"description": "Amount of context to return around matches (default: 100)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "search_vault_advanced",
			Description: "Advanced search using Dataview DQL or JsonLogic queries",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query (DQL or JsonLogic)",
					},
					"queryType": map[string]any{
						"type":        "string",
						"description": "Query type",
						"enum":        []string{"dataview", "jsonlogic"},
					},
				},
				"required": []string{"query", "queryType"},
			},
		},
		{
			Name:        "list_commands",
			Description: "Get a list of available Obsidian commands",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "execute_command",
			Description: "Execute a specific Obsidian command",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"commandId": map[string]any{
						"type":        "string",
						"description": "ID of the command to execute",
					},
				},
				"required": []string{"commandId"},
			},
		},
		{
			Name:        "open_file",
			Description: "Open a file in the Obsidian UI",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file relative to vault root",
					},
					"newLeaf": map[string]any{
						"type":        "boolean",
						"description": "Open in a new leaf (default: false)",
					},
				},
				"required": []string{"filename"},
			},
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

// handleToolsCall handles tool execution requests
func (s *MCPServer) handleToolsCall(request *MCPRequest) *MCPResponse {
	params, ok := request.Params["arguments"].(map[string]any)
	if !ok {
		params = make(map[string]any)
	}

	name, ok := request.Params["name"].(string)
	if !ok {
		return s.createErrorResponse(request.ID, -32602, "Invalid params: missing tool name")
	}

	result, err := s.executeTool(name, params)
	if err != nil {
		return s.createErrorResponse(request.ID, -32603, err.Error())
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

// executeTool executes the specified tool with given parameters
func (s *MCPServer) executeTool(name string, params map[string]any) (string, error) {
	switch name {
	case "get_server_info":
		return s.obsidianClient.GetServerInfo()
	case "list_vault_files":
		path, _ := params["path"].(string)
		return s.obsidianClient.ListVaultFiles(path)
	case "get_file_content":
		filename, ok := params["filename"].(string)
		if !ok {
			return "", fmt.Errorf("filename is required")
		}
		format, _ := params["format"].(string)
		if format == "" {
			format = "markdown"
		}
		return s.obsidianClient.GetFileContent(filename, format)
	case "create_or_update_file":
		filename, ok := params["filename"].(string)
		if !ok {
			return "", fmt.Errorf("filename is required")
		}
		content, ok := params["content"].(string)
		if !ok {
			return "", fmt.Errorf("content is required")
		}
		contentType, _ := params["contentType"].(string)
		if contentType == "" {
			contentType = "text/markdown"
		}
		return s.obsidianClient.CreateOrUpdateFile(filename, content, contentType)
	case "append_to_file":
		filename, ok := params["filename"].(string)
		if !ok {
			return "", fmt.Errorf("filename is required")
		}
		content, ok := params["content"].(string)
		if !ok {
			return "", fmt.Errorf("content is required")
		}
		return s.obsidianClient.AppendToFile(filename, content)
	case "patch_file_content":
		filename, ok := params["filename"].(string)
		if !ok {
			return "", fmt.Errorf("filename is required")
		}
		operation, ok := params["operation"].(string)
		if !ok {
			return "", fmt.Errorf("operation is required")
		}
		targetType, ok := params["targetType"].(string)
		if !ok {
			return "", fmt.Errorf("targetType is required")
		}
		target, ok := params["target"].(string)
		if !ok {
			return "", fmt.Errorf("target is required")
		}
		content, ok := params["content"].(string)
		if !ok {
			return "", fmt.Errorf("content is required")
		}
		contentType, _ := params["contentType"].(string)
		if contentType == "" {
			contentType = "text/markdown"
		}
		delimiter, _ := params["delimiter"].(string)
		if delimiter == "" {
			delimiter = "::"
		}
		return s.obsidianClient.PatchFileContent(filename, operation, targetType, target, content, contentType, delimiter)
	case "delete_file":
		filename, ok := params["filename"].(string)
		if !ok {
			return "", fmt.Errorf("filename is required")
		}
		return s.obsidianClient.DeleteFile(filename)
	case "search_vault_simple":
		query, ok := params["query"].(string)
		if !ok {
			return "", fmt.Errorf("query is required")
		}
		contextLength := 100
		if cl, ok := params["contextLength"].(float64); ok {
			contextLength = int(cl)
		}
		return s.obsidianClient.SearchVaultSimple(query, contextLength)
	case "search_vault_advanced":
		query, ok := params["query"].(string)
		if !ok {
			return "", fmt.Errorf("query is required")
		}
		queryType, ok := params["queryType"].(string)
		if !ok {
			return "", fmt.Errorf("queryType is required")
		}
		return s.obsidianClient.SearchVaultAdvanced(query, queryType)
	case "list_commands":
		return s.obsidianClient.ListCommands()
	case "execute_command":
		commandId, ok := params["commandId"].(string)
		if !ok {
			return "", fmt.Errorf("commandId is required")
		}
		return s.obsidianClient.ExecuteCommand(commandId)
	case "open_file":
		filename, ok := params["filename"].(string)
		if !ok {
			return "", fmt.Errorf("filename is required")
		}
		newLeaf, _ := params["newLeaf"].(bool)
		return s.obsidianClient.OpenFile(filename, newLeaf)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// sendResponse sends a response to stdout
func (s *MCPServer) sendResponse(response *MCPResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	_, err = s.stdout.Write(append(data, '\n'))
	return err
}

// sendError sends an error response
func (s *MCPServer) sendError(id any, code int, message, data string) {
	response := &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.sendResponse(response)
}

// createErrorResponse creates an error response
func (s *MCPServer) createErrorResponse(id any, code int, message string) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
}
