# Obsidian MCP Server

[![CI](https://github.com/thomastaylor312/obsidian-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/thomastaylor312/obsidian-mcp-server/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/obsidian-mcp-server/obsidian-mcp-server)](https://goreportcard.com/report/github.com/obsidian-mcp-server/obsidian-mcp-server)

A Model Context Protocol (MCP) server for Obsidian that provides programmatic access to your Obsidian vault through the Local REST API plugin. This is mostly coded via AI, with some checks on my end. Several projects like this already exist, but I wanted complete control over what the server is doing.

## Features

### 🗂️ Vault Management
- **File Operations**: Create, read, update, and delete files in your vault
- **Directory Browsing**: List files and directories within your vault
- **Content Patching**: Insert content relative to headings, blocks, or frontmatter fields
- **Flexible Formats**: Support for both markdown text and structured JSON responses

### 🔍 Search Capabilities
- **Simple Text Search**: Fast text-based search across your entire vault
- **Advanced Queries**: Support for Dataview DQL and JsonLogic queries
- **Contextual Results**: Configurable context length around search matches

### ⚡ Command Integration
- **Command Discovery**: List all available Obsidian commands
- **Command Execution**: Programmatically execute Obsidian commands
- **File Navigation**: Open files directly in the Obsidian UI

### 🔐 Secure & Local
- **Bearer Token Authentication**: Secure API access using tokens from Obsidian
- **Local-Only**: Communicates only with your local Obsidian instance
- **Stdio Transport**: Uses standard input/output for MCP communication

## Prerequisites

1. **Obsidian** with the **Local REST API** plugin installed and enabled
2. **Go 1.22+** (for building from source)
3. **Nix** (optional, for development environment)

## Installation

### From Source

```bash
git clone https://github.com/obsidian-mcp-server/obsidian-mcp-server.git
cd obsidian-mcp-server
make build
```

The binary will be available at `./bin/obsidian-mcp-server`.

### Using Nix (Development)

```bash
git clone https://github.com/obsidian-mcp-server/obsidian-mcp-server.git
cd obsidian-mcp-server
direnv allow  # If using direnv
nix develop   # Otherwise, enter the development shell manually
make build
```

## Setup

### 1. Configure Obsidian Local REST API

1. Install the **Local REST API** plugin in Obsidian
2. Enable the plugin in your plugin settings
3. Note the API token shown in the plugin settings
4. Ensure the server is running (default: `http://127.0.0.1:27123`)

### 2. Run the MCP Server

```bash
# Using environment variable (recommended)
export OBSIDIAN_API_TOKEN="your-api-token-here"
./bin/obsidian-mcp-server

# Or using command-line flag
./bin/obsidian-mcp-server -token "your-api-token-here"

# Custom server URL (if needed)
./bin/obsidian-mcp-server -token "your-token" -url "http://localhost:27123"
```

### 3. Connect Your MCP Client

The server communicates via stdin/stdout using the MCP protocol. Connect your MCP-compatible client to interact with your Obsidian vault programmatically.

## Available Tools

### File Management
- `get_server_info` - Get Obsidian server status and authentication info
- `list_vault_files` - List files in vault root or specific directory
- `get_file_content` - Read file content (markdown or JSON format with metadata)
- `create_or_update_file` - Create new files or update existing ones
- `append_to_file` - Append content to existing files
- `patch_file_content` - Insert content relative to headings, blocks, or frontmatter
- `delete_file` - Delete files from the vault

### Search & Discovery
- `search_vault_simple` - Simple text search with configurable context
- `search_vault_advanced` - Advanced search using Dataview DQL or JsonLogic

### Command & Navigation
- `list_commands` - Get all available Obsidian commands
- `execute_command` - Execute specific Obsidian commands
- `open_file` - Open files in the Obsidian UI

## Development

### Development Environment

This project uses Nix for reproducible development environments:

```bash
# Setup development environment
direnv allow  # Automatically loads the environment
# or manually:
nix develop

# Available commands
make help        # Show all available commands
make generate    # Generate OpenAPI client code
make build       # Build the binary
make test        # Run unit tests
make test-e2e    # Run end-to-end tests (requires Obsidian setup)
make lint        # Run linter
make fmt         # Format code
make clean       # Clean build artifacts
```

### Testing

#### Unit Tests

```bash
make test-unit
```

#### End-to-End Tests

E2E tests require a running Obsidian instance with the Local REST API plugin:

```bash
# Set up your API token
export OBSIDIAN_API_TOKEN="your-token"

# Ensure Obsidian is running with the REST API plugin enabled
# Then run the tests
make test-e2e
```

### Project Structure

```
├── cmd/obsidian-mcp-server/    # Main application entry point
├── internal/
│   ├── mcp/                    # MCP server implementation
│   └── obsidian/              # Obsidian client wrapper
├── pkg/obsidian/              # Generated OpenAPI client code
├── test/e2e/                  # End-to-end tests
├── .github/workflows/         # CI/CD pipelines
├── flake.nix                  # Nix development environment
├── Makefile                   # Build and development commands
└── openapi.yaml              # Obsidian REST API specification
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make your changes and add tests
4. Run the full test suite: `make test lint`
5. Commit your changes: `git commit -am 'Add new feature'`
6. Push to the branch: `git push origin feature/new-feature`
7. Submit a Pull Request

### Development Guidelines

- Follow Go best practices and conventions
- Add unit tests for all new functionality
- Update documentation for user-facing changes
- Run `make fmt lint` before committing
- Ensure all CI checks pass

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
