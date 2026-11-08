package jsonrpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Envelope is a decoded incoming message before classification: it carries
// request fields (method/params) and response fields (result/error) side by
// side; exactly one interpretation is valid.
type Envelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *ID             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
}

// IsRequest reports whether the envelope is a request/notification.
func (e *Envelope) IsRequest() bool { return e.Method != "" }

// AsRequest projects the envelope as a Request.
func (e *Envelope) AsRequest() *Request {
	return &Request{JSONRPC: "2.0", ID: e.ID, Method: e.Method, Params: e.Params}
}

// AsResponse projects the envelope as a Response.
func (e *Envelope) AsResponse() *Response {
	return &Response{JSONRPC: "2.0", ID: e.ID, Result: e.Result, Error: e.Error}
}

// Decoder reads newline-delimited JSON messages (the MCP stdio framing).
// Blank lines are skipped; a trailing "\r" (Windows writers) is tolerated.
type Decoder struct {
	br *bufio.Reader
}

// NewDecoder wraps r.
func NewDecoder(r io.Reader) *Decoder { return &Decoder{br: bufio.NewReader(r)} }

// Read returns the next message. io.EOF is returned (unwrapped) when the
// stream ends cleanly. Malformed lines produce a syntax error but do not
// stop the stream; callers may keep reading.
func (d *Decoder) Read() (*Envelope, error) {
	for {
		line, err := d.br.ReadBytes('\n')
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			var env Envelope
			if jerr := json.Unmarshal(trimmed, &env); jerr != nil {
				return nil, fmt.Errorf("jsonrpc: malformed message: %w", jerr)
			}
			return &env, nil
		}
		if err == io.EOF {
			return nil, io.EOF
		}
		if err != nil {
			return nil, fmt.Errorf("jsonrpc: read: %w", err)
		}
		// blank line: skip
	}
}

// Encoder writes newline-delimited JSON messages.
type Encoder struct {
	w   io.Writer
	buf bytes.Buffer
}

// NewEncoder wraps w.
func NewEncoder(w io.Writer) *Encoder { return &Encoder{w: w} }

// Write marshals v and appends a newline. Marshal errors are returned before
// anything is written, so a failed Write never emits a partial line.
func (e *Encoder) Write(v any) error {
	e.buf.Reset()
	if err := json.NewEncoder(&e.buf).Encode(v); err != nil {
		return fmt.Errorf("jsonrpc: encode: %w", err)
	}
	_, err := e.w.Write(e.buf.Bytes())
	return err
}
