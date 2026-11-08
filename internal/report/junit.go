package report

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

// junitSuites is the JUnit XML root (widely consumed by CI systems).
type junitSuites struct {
	XMLName  xml.Name     `xml:"testsuites"`
	Tests    int          `xml:"tests,attr"`
	Failures int          `xml:"failures,attr"`
	Errors   int          `xml:"errors,attr"`
	Time     string       `xml:"time,attr"`
	Suites   []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Errors   int         `xml:"errors,attr"`
	Time     string      `xml:"time,attr"`
	Cases    []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name      string     `xml:"name,attr"`
	Classname string     `xml:"classname,attr"`
	Time      string     `xml:"time,attr"`
	Failure   *junitBody `xml:"failure,omitempty"`
	Error     *junitBody `xml:"error,omitempty"`
}

type junitBody struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Inner   string `xml:",chardata"`
}

// RenderJUnit writes reports in JUnit XML format.
func RenderJUnit(w io.Writer, reports []*runner.SuiteReport, generatedAt time.Time) error {
	root := junitSuites{Time: fmt.Sprintf("%.3f", totalSeconds(reports))}
	for _, r := range reports {
		js := junitSuite{Name: r.Suite, Time: fmt.Sprintf("%.3f", float64(r.DurationMS)/1000)}
		for _, c := range r.Cases {
			jc := junitCase{
				Name:      c.Name,
				Classname: r.Suite,
				Time:      fmt.Sprintf("%.3f", float64(c.DurationMS)/1000),
			}
			switch c.Status {
			case runner.StatusFail:
				js.Failures++
				jc.Failure = &junitBody{Message: firstFailedCheck(c), Type: "AssertionFailed", Inner: caseDetails(c)}
			case runner.StatusError:
				js.Errors++
				jc.Error = &junitBody{Message: c.Err, Type: "SuiteError", Inner: caseDetails(c)}
			}
			js.Cases = append(js.Cases, jc)
			js.Tests++
		}
		root.Tests += js.Tests
		root.Failures += js.Failures
		root.Errors += js.Errors
		root.Suites = append(root.Suites, js)
	}

	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

func totalSeconds(reports []*runner.SuiteReport) float64 {
	var ms int64
	for _, r := range reports {
		ms += r.DurationMS
	}
	return float64(ms) / 1000
}

func firstFailedCheck(c runner.CaseResult) string {
	for _, ck := range c.Checks {
		if !ck.Pass {
			return ck.Desc
		}
	}
	return "case failed"
}

func caseDetails(c runner.CaseResult) string {
	details := c.Err
	for _, ck := range c.Checks {
		if !ck.Pass {
			details += "\n" + ck.Desc + ": " + ck.Detail
		}
	}
	return details
}
