package jsonrpc

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDecoderRequestResponse(t *testing.T) {
	in := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n" +
		`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}` + "\n"
	d := NewDecoder(strings.NewReader(in))

	env, err := d.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !env.IsRequest() || env.Method != "tools/list" {
		t.Errorf("first message should be request, got %+v", env)
	}

	env, err = d.Read()
	if err != nil {
		t.Fatal(err)
	}
	if env.IsRequest() {
		t.Errorf("second message should be response, got %+v", env)
	}
	if string(env.Result) != `{"tools":[]}` {
		t.Errorf("result payload mismatch: %s", env.Result)
	}

	if _, err := d.Read(); !errors.Is(err, io.EOF) {
		t.Errorf("want EOF, got %v", err)
	}
}

func TestDecoderSkipsBlankLinesAndCRLF(t *testing.T) {
	in := "\r\n" +
		`{"jsonrpc":"2.0","id":"x","result":1}` + "\r\n" +
		"\n"
	d := NewDecoder(strings.NewReader(in))
	env, err := d.Read()
	if err != nil {
		t.Fatal(err)
	}
	if env.ID == nil || env.ID.Str != "x" {
		t.Errorf("crlf response not decoded: %+v", env)
	}
	if _, err := d.Read(); !errors.Is(err, io.EOF) {
		t.Errorf("want EOF after trailing blank line, got %v", err)
	}
}

func TestDecoderMalformedThenRecover(t *testing.T) {
	in := "{not json}\n" + `{"jsonrpc":"2.0","id":2,"result":true}` + "\n"
	d := NewDecoder(strings.NewReader(in))
	if _, err := d.Read(); err == nil {
		t.Error("malformed line should error")
	}
	env, err := d.Read()
	if err != nil || env.ID == nil || env.ID.Num != 2 {
		t.Errorf("decoder should recover after malformed line: %v %+v", err, env)
	}
}

func TestDecoderNotificationNoID(t *testing.T) {
	d := NewDecoder(strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"))
	env, err := d.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !env.IsRequest() || env.ID != nil {
		t.Errorf("notification should have method and nil id: %+v", env)
	}
}

func TestEncoderWrite(t *testing.T) {
	var sb strings.Builder
	enc := NewEncoder(&sb)
	req, _ := NewRequest(NewNumID(3), "ping", nil)
	if err := enc.Write(req); err != nil {
		t.Fatal(err)
	}
	resp, _ := NewResultResponse(NewNumID(3), "pong")
	if err := enc.Write(resp); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), sb.String())
	}
	if !strings.HasSuffix(lines[0], `"method":"ping"}`) {
		t.Errorf("line1 = %s", lines[0])
	}
	if !strings.Contains(lines[1], `"result":"pong"`) {
		t.Errorf("line2 = %s", lines[1])
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	var sb strings.Builder
	enc := NewEncoder(&sb)
	resp, err := NewResultResponse(NewStrID("rt-9"), map[string]any{"nested": []any{1, "two", false}})
	if err != nil {
		t.Fatal(err)
	}
	if err := enc.Write(resp); err != nil {
		t.Fatal(err)
	}
	env, err := NewDecoder(strings.NewReader(sb.String())).Read()
	if err != nil {
		t.Fatal(err)
	}
	if env.ID == nil || env.ID.Str != "rt-9" {
		t.Errorf("id round trip failed: %+v", env)
	}
	if string(env.Result) != `{"nested":[1,"two",false]}` {
		t.Errorf("result round trip failed: %s", env.Result)
	}
}
