package jsonrpc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRequestMarshal(t *testing.T) {
	req, err := NewRequest(NewNumID(7), "tools/call", map[string]any{"name": "echo"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	want := `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"echo"}}`
	if got != want {
		t.Errorf("request marshal:\n got %s\nwant %s", got, want)
	}
}

func TestNotificationOmitsID(t *testing.T) {
	n, err := NewNotification("notifications/initialized", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(n)
	got := string(raw)
	if strings.Contains(got, `"id"`) {
		t.Errorf("notification must omit id, got %s", got)
	}
	if got != `{"jsonrpc":"2.0","method":"notifications/initialized"}` {
		t.Errorf("unexpected notification encoding: %s", got)
	}
	if !n.IsNotification() {
		t.Error("NewNotification should be a notification")
	}
}

func TestStringIDRoundTrip(t *testing.T) {
	req, err := NewRequest(NewStrID("abc-1"), "ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(req)
	if !strings.Contains(string(raw), `"id":"abc-1"`) {
		t.Fatalf("string id not encoded: %s", raw)
	}

	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env.ID == nil || env.ID.Str != "abc-1" || !env.ID.IsStr {
		t.Fatalf("string id not decoded: %+v", env.ID)
	}
}

func TestIDUnmarshal(t *testing.T) {
	cases := []struct {
		data  string
		isStr bool
		num   int64
		str   string
		nilID bool
	}{
		{`42`, false, 42, "", false},
		{`"s1"`, true, 0, "s1", false},
		{`null`, false, 0, "", true},
	}
	for _, c := range cases {
		var id ID
		if err := id.UnmarshalJSON([]byte(c.data)); err != nil {
			t.Errorf("UnmarshalJSON(%s): %v", c.data, err)
			continue
		}
		if id.IsStr != c.isStr || id.Num != c.num || id.Str != c.str {
			t.Errorf("UnmarshalJSON(%s) = %+v", c.data, id)
		}
	}
	if err := (&ID{}).UnmarshalJSON([]byte(`true`)); err == nil {
		t.Error("bool id should be rejected")
	}
}

func TestResponseResultErrorExclusive(t *testing.T) {
	resp, err := NewResultResponse(NewNumID(1), map[string]any{"ok": true})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(resp)
	if strings.Contains(string(raw), `"error"`) {
		t.Errorf("result response must not carry error: %s", raw)
	}

	errResp := NewErrorResponse(NewNumID(2), CodeInternalError, "boom", nil)
	raw2, _ := json.Marshal(errResp)
	if strings.Contains(string(raw2), `"result"`) {
		t.Errorf("error response must not carry result: %s", raw2)
	}
	if !strings.Contains(string(raw2), `"code":-32603`) {
		t.Errorf("error code not encoded: %s", raw2)
	}
}

func TestRPCErrorMessage(t *testing.T) {
	e := &RPCError{Code: CodeMethodNotFound, Message: "no such tool"}
	want := "jsonrpc error -32631: no such tool"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}
