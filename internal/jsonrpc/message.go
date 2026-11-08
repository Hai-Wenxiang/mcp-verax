// Package jsonrpc implements JSON-RPC 2.0 messages as used by MCP: shared
// envelope decoding, request/notification/response types and standard error
// codes. MCP transports are line-delimited JSON (stdio) or HTTP, both of
// which consume these types.
package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32631 // MCP reserves -32631..-32632 for tools/prompts
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// ID is a JSON-RPC request identifier. It may be a number or a string;
// an absent ID marks a notification. Zero value = notification.
type ID struct {
	Num   int64
	Str   string
	IsStr bool
}

// NewNumID returns a numeric ID.
func NewNumID(n int64) ID { return ID{Num: n} }

// NewStrID returns a string ID.
func NewStrID(s string) ID { return ID{Str: s, IsStr: true} }

func (id ID) String() string {
	if id.IsStr {
		return fmt.Sprintf("%q", id.Str)
	}
	return fmt.Sprintf("%d", id.Num)
}

// MarshalJSON encodes the ID; used by Request/Response via pointer so that
// nil encodes as absent (notifications have no id field).
func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsStr {
		return json.Marshal(id.Str)
	}
	return json.Marshal(id.Num)
}

// UnmarshalJSON decodes a numeric or string ID. JSON null decodes to the
// zero ID (notification-style) without error.
func (id *ID) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		*id = ID{}
		return nil
	}
	if len(s) > 0 && s[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*id = ID{Str: str, IsStr: true}
		return nil
	}
	var n int64
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("jsonrpc: id must be number or string, got %s", s)
	}
	*id = ID{Num: n}
	return nil
}

// Request is a JSON-RPC request or notification (when ID is nil).
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *ID             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// NewRequest builds a call carrying the given ID.
func NewRequest(id ID, method string, params any) (*Request, error) {
	raw, err := optionalMarshal(params)
	if err != nil {
		return nil, err
	}
	return &Request{JSONRPC: "2.0", ID: &id, Method: method, Params: raw}, nil
}

// NewNotification builds a request without an ID.
func NewNotification(method string, params any) (*Request, error) {
	raw, err := optionalMarshal(params)
	if err != nil {
		return nil, err
	}
	return &Request{JSONRPC: "2.0", Method: method, Params: raw}, nil
}

// IsNotification reports whether the request expects no response.
func (r *Request) IsNotification() bool { return r.ID == nil }

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// Response is a JSON-RPC response: exactly one of Result / Error is set.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *ID             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// NewResultResponse builds a success response.
func NewResultResponse(id ID, result any) (*Response, error) {
	raw, err := optionalMarshal(result)
	if err != nil {
		return nil, err
	}
	return &Response{JSONRPC: "2.0", ID: &id, Result: raw}, nil
}

// NewErrorResponse builds a failure response.
func NewErrorResponse(id ID, code int, message string, data any) *Response {
	return &Response{JSONRPC: "2.0", ID: &id, Error: &RPCError{Code: code, Message: message, Data: data}}
}

func optionalMarshal(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}
