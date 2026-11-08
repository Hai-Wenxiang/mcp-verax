package mcpclient

import "encoding/json"

// SupportedProtocolVersions lists MCP protocol versions Verax can negotiate,
// newest first. The client proposes the newest and accepts any listed.
var SupportedProtocolVersions = []string{
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

// LatestProtocolVersion is the version the client proposes by default.
func LatestProtocolVersion() string { return SupportedProtocolVersions[0] }

// ClientInfo identifies Verax during the initialize handshake.
var ClientInfo = struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}{Name: "verax", Version: "0.0.1-dev"}

// InitializeParams is the initialize request payload.
type InitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    struct {
		Roots struct {
			ListChanged bool `json:"listChanged"`
		} `json:"roots"`
	} `json:"capabilities"`
	ClientInfo any `json:"clientInfo"`
}

// ServerCapabilities mirrors the handshake result capabilities subset Verax
// consumes. Unknown capability keys are ignored.
type ServerCapabilities struct {
	Tools     *CapabilityFlag `json:"tools,omitempty"`
	Resources *CapabilityFlag `json:"resources,omitempty"`
	Prompts   *CapabilityFlag `json:"prompts,omitempty"`
	Logging   *CapabilityFlag `json:"logging,omitempty"`
}

// CapabilityFlag is the common {listChanged?, subscribe?} capability shape.
type CapabilityFlag struct {
	ListChanged bool `json:"listChanged,omitempty"`
	Subscribe   bool `json:"subscribe,omitempty"`
}

// Implementation is the {name, version} identity block.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is the handshake response payload.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// Tool is an entry of tools/list.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ToolsResult is the tools/list payload.
type ToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams is the tools/call request payload.
type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// Content is a content block of a tool call result.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallToolResult is the tools/call payload.
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Resource is an entry of resources/list.
type Resource struct {
	URI      string `json:"uri"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType,omitempty"`
}

// ResourcesResult is the resources/list payload.
type ResourcesResult struct {
	Resources []Resource `json:"resources"`
}

// Prompt is an entry of prompts/list.
type Prompt struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PromptsResult is the prompts/list payload.
type PromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}
