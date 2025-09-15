package obsidian

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	client := NewClient("test-token", "http://localhost:27123")
	assert.NotNil(t, client)
	assert.Equal(t, "test-token", client.apiToken)
	assert.Equal(t, "http://localhost:27123", client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.apiClient)
}

// TestNewClientTrimsTrailingSlash tests that trailing slashes are handled correctly
func TestNewClientTrimsTrailingSlash(t *testing.T) {
	client := NewClient("test-token", "http://localhost:27123/")
	assert.Equal(t, "http://localhost:27123", client.baseURL)
}

// TestMakeRequest tests the makeRequest helper method
func TestMakeRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token", authHeader)

		// Verify custom header
		customHeader := r.Header.Get("Custom-Header")
		assert.Equal(t, "test-value", customHeader)

		// Return test response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status": "ok"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	headers := map[string]string{
		"Custom-Header": "test-value",
	}

	data, err := client.makeRequest("GET", "/test", headers, nil)
	require.NoError(t, err)
	assert.Equal(t, `{"status": "ok"}`, string(data))
}

// TestMakeRequestErrorHandling tests error handling in makeRequest
func TestMakeRequestErrorHandling(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(`{"error": "Not Found"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.makeRequest("GET", "/nonexistent", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error (status 404)")
}

// TestGetServerInfo tests the GetServerInfo method
func TestGetServerInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"ok": "OK",
			"service": "Obsidian Local REST API",
			"authenticated": true,
			"versions": {
				"obsidian": "1.0.0",
				"self": "3.0.0"
			}
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.GetServerInfo()
	require.NoError(t, err)
	assert.Contains(t, result, "Obsidian Local REST API")
	assert.Contains(t, result, "authenticated")
}

// TestListVaultFiles tests the ListVaultFiles method
func TestListVaultFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/vault/"
		assert.Equal(t, expectedPath, r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"files": [
				"note1.md",
				"note2.md",
				"subfolder/"
			]
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.ListVaultFiles("")
	require.NoError(t, err)
	assert.Contains(t, result, "note1.md")
	assert.Contains(t, result, "note2.md")
}

// TestListVaultFilesWithPath tests listing files in a subdirectory
func TestListVaultFilesWithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/vault/subfolder/"
		assert.Equal(t, expectedPath, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"files": [
				"subnote1.md",
				"subnote2.md"
			]
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.ListVaultFiles("subfolder")
	require.NoError(t, err)
	assert.Contains(t, result, "subnote1.md")
}

// TestGetFileContent tests getting file content in markdown format
func TestGetFileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/vault/test.md", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.NotEqual(t, "application/vnd.olrapi.note+json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "text/markdown")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("# Test Note\n\nThis is test content."))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.GetFileContent("test.md", "markdown")
	require.NoError(t, err)
	assert.Equal(t, "# Test Note\n\nThis is test content.", result)
}

// TestGetFileContentJSON tests getting file content in JSON format
func TestGetFileContentJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/vault/test.md", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/vnd.olrapi.note+json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/vnd.olrapi.note+json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"content": "# Test Note\n\nThis is test content.",
			"path": "test.md",
			"tags": ["test"],
			"frontmatter": {},
			"stat": {
				"ctime": 1640995200,
				"mtime": 1640995200,
				"size": 1024
			}
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.GetFileContent("test.md", "json")
	require.NoError(t, err)
	assert.Contains(t, result, "Test Note")
	assert.Contains(t, result, "tags")
	assert.Contains(t, result, "frontmatter")
}

// TestCreateOrUpdateFile tests creating/updating a file
func TestCreateOrUpdateFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/vault/test.md", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "text/markdown", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.CreateOrUpdateFile("test.md", "# New Note", "text/markdown")
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully created/updated file: test.md")
}

// TestAppendToFile tests appending to a file
func TestAppendToFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/vault/test.md", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/markdown", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.AppendToFile("test.md", "\n\nAppended content")
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully appended to file: test.md")
}

// TestPatchFileContent tests patching file content
func TestPatchFileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/vault/test.md", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "text/markdown", r.Header.Get("Content-Type"))
		assert.Equal(t, "append", r.Header.Get("Operation"))
		assert.Equal(t, "heading", r.Header.Get("Target-Type"))
		target := r.Header.Get("Target")
		assert.True(t, target == "Test%20Heading" || target == "Test+Heading", "Target should be URL encoded, got: %s", target)
		assert.Equal(t, "::", r.Header.Get("Target-Delimiter"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.PatchFileContent("test.md", "append", "heading", "Test Heading", "New content", "text/markdown", "::")
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully patched file: test.md")
}

// TestDeleteFile tests deleting a file
func TestDeleteFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/vault/test.md", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.DeleteFile("test.md")
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully deleted file: test.md")
}

// TestSearchVaultSimple tests simple vault search
func TestSearchVaultSimple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search/simple/", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.RawQuery, "query=test")
		assert.Contains(t, r.URL.RawQuery, "contextLength=50")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`[
			{
				"filename": "note1.md",
				"matches": [
					{
						"context": "This is test content with the search term.",
						"match": {"start": 10, "end": 14}
					}
				],
				"score": 0.95
			}
		]`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.SearchVaultSimple("test", 50)
	require.NoError(t, err)
	assert.Contains(t, result, "note1.md")
	assert.Contains(t, result, "test content")
}

// TestSearchVaultAdvanced tests advanced vault search with Dataview
func TestSearchVaultAdvanced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search/", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/vnd.olrapi.dataview.dql+txt", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`[
			{
				"filename": "note1.md",
				"result": ["value1", "value2"]
			}
		]`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.SearchVaultAdvanced("TABLE field FROM #tag", "dataview")
	require.NoError(t, err)
	assert.Contains(t, result, "note1.md")
	assert.Contains(t, result, "value1")
}

// TestListCommands tests listing available commands
func TestListCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/commands/", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"commands": [
				{
					"id": "global-search:open",
					"name": "Search: Search in all files"
				},
				{
					"id": "graph:open",
					"name": "Graph view: Open graph view"
				}
			]
		}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.ListCommands()
	require.NoError(t, err)
	assert.Contains(t, result, "global-search:open")
	assert.Contains(t, result, "Graph view")
}

// TestExecuteCommand tests executing a command
func TestExecuteCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The URL path might not be URL encoded by the test server
		expectedPaths := []string{"/commands/global-search%3Aopen/", "/commands/global-search:open/"}
		assert.Contains(t, expectedPaths, r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.ExecuteCommand("global-search:open")
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully executed command: global-search:open")
}

// TestOpenFile tests opening a file
func TestOpenFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/open/test.md", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "newLeaf=true", r.URL.RawQuery)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	result, err := client.OpenFile("test.md", true)
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully opened file: test.md")
}
