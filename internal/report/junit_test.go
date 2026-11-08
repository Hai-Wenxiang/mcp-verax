package report

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

func TestRenderJUnitStructure(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJUnit(&buf, sampleReports(), time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	raw := buf.String()

	if !strings.HasPrefix(raw, "<?xml") {
		t.Error("missing XML header")
	}

	var parsed struct {
		XMLName  xml.Name `xml:"testsuites"`
		Tests    int      `xml:"tests,attr"`
		Failures int      `xml:"failures,attr"`
		Errors   int      `xml:"errors,attr"`
		Suites   []struct {
			Name  string `xml:"name,attr"`
			Tests int    `xml:"tests,attr"`
			Cases []struct {
				Name    string `xml:"name,attr"`
				Failure *struct {
					Message string `xml:"message,attr"`
				} `xml:"failure"`
				Error *struct {
					Message string `xml:"message,attr"`
				} `xml:"error"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output not valid JUnit XML: %v\n%s", err, raw)
	}
	if parsed.Tests != 3 || parsed.Failures != 1 || parsed.Errors != 1 {
		t.Errorf("totals: tests=%d failures=%d errors=%d", parsed.Tests, parsed.Failures, parsed.Errors)
	}
	if len(parsed.Suites) != 1 || parsed.Suites[0].Name != "demo" {
		t.Fatalf("suite wrong: %+v", parsed.Suites)
	}
	cases := parsed.Suites[0].Cases
	if cases[0].Failure != nil || cases[0].Error != nil {
		t.Error("passing case must carry no failure/error")
	}
	if cases[1].Failure == nil || !strings.Contains(cases[1].Failure.Message, "ok") {
		t.Errorf("failed case should carry failure node: %+v", cases[1])
	}
	if cases[2].Error == nil || !strings.Contains(cases[2].Error.Message, "start server") {
		t.Errorf("error case should carry error node: %+v", cases[2])
	}
}

func TestRenderJUnitEscapesHostileNames(t *testing.T) {
	reports := []*runner.SuiteReport{{
		Suite: "esc<>&\"", Source: "esc.suite.yaml",
		Cases: []runner.CaseResult{{
			Name:   `case <b>&"'`,
			Status: runner.StatusFail,
		}},
	}}
	var buf bytes.Buffer
	if err := RenderJUnit(&buf, reports, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "<b>") {
		t.Error("hostile case name leaked as markup")
	}
}
