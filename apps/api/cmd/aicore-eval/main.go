// aicore-eval executes synthetic WP-15 golden scenarios using explicit
// recorded observations or sanitized turn_trace.v1 payloads. It does not
// read application databases or invoke live models by default.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/eval"
)

func main() { os.Exit(run()) }
func run() int {
	golden := flag.String("golden", "", "directory of golden.v1 YAML cases")
	observed := flag.String("observations", "", "directory of eval_observations.v1 JSON files")
	traces := flag.String("traces", "", "directory of synthetic eval_trace_fixture.v1 JSON files")
	baseline := flag.String("baseline", "", "optional previous eval_report.v1 JSON")
	out := flag.String("out", "", "optional JSON report path")
	flag.Parse()
	if *golden == "" || (*observed == "") == (*traces == "") {
		fmt.Fprintln(os.Stderr, "usage: aicore-eval -golden DIR (-observations DIR | -traces DIR) [-baseline FILE] [-out FILE]")
		return 2
	}
	paths, err := eval.Files(*golden)
	if err != nil || len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "evaluation: missing or invalid fixture directory")
		return 2
	}
	results := []eval.ScenarioReport{}
	full := eval.Report{Schema: "eval_report.v1", Method: "synthetic_recorded", SyntheticOnly: true, Passed: true, Metrics: map[string]eval.Metric{}}
	for _, path := range paths {
		s, e := eval.LoadFile(path)
		if e != nil {
			fmt.Fprintln(os.Stderr, "evaluation: invalid synthetic golden fixture")
			return 2
		}
		var executor eval.Executor
		file := filepath.Join(*observed, s.ID+".json")
		if *traces != "" {
			full.Method = "synthetic_trace_replay"
			file = filepath.Join(*traces, s.ID+".json")
			t, e := eval.LoadTraceFixture(file)
			if e != nil {
				fmt.Fprintln(os.Stderr, "evaluation: invalid synthetic trace fixture")
				return 2
			}
			executor = eval.TraceExecutor{Fixture: t}
		} else {
			o, e := eval.LoadRecorded(file)
			if e != nil {
				fmt.Fprintln(os.Stderr, "evaluation: invalid synthetic observation fixture")
				return 2
			}
			executor = eval.NewRecordedExecutor(o)
		}
		report := eval.Runner{Executor: executor}.Run(context.Background(), []eval.Scenario{s})
		full.Passed = full.Passed && report.Passed
		full.Totals.Scenarios += report.Totals.Scenarios
		full.Totals.PassedScenarios += report.Totals.PassedScenarios
		full.Totals.FailedScenarios += report.Totals.FailedScenarios
		full.Totals.Turns += report.Totals.Turns
		full.Totals.PassedTurns += report.Totals.PassedTurns
		full.Totals.Assertions += report.Totals.Assertions
		full.Totals.PassedAssertions += report.Totals.PassedAssertions
		full.Totals.Failures += report.Totals.Failures
		full.Totals.Blockers += report.Totals.Blockers
		for key, m := range report.Metrics {
			v := full.Metrics[key]
			v.Passed += m.Passed
			v.Total += m.Total
			full.Metrics[key] = v
		}
		results = append(results, report.Scenarios...)
		full.ReviewQueue = append(full.ReviewQueue, report.ReviewQueue...)
	}
	full.Scenarios = results
	for key, m := range full.Metrics {
		if m.Total > 0 {
			m.Rate = float64(m.Passed) / float64(m.Total)
		}
		full.Metrics[key] = m
	}
	if *baseline != "" {
		b, e := eval.LoadReport(*baseline)
		if e != nil {
			fmt.Fprintln(os.Stderr, "evaluation: invalid baseline")
			return 2
		}
		full.Regressions = eval.CompareReports(b, full, eval.RegressionPolicy{})
		if len(full.Regressions) > 0 {
			full.Passed = false
		}
	}
	data, e := eval.EncodeReport(full)
	if e != nil {
		fmt.Fprintln(os.Stderr, "evaluation: report failure")
		return 2
	}
	if *out != "" {
		f, openErr := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if openErr != nil {
			fmt.Fprintln(os.Stderr, "evaluation: report write failed")
			return 2
		}
		_ = f.Chmod(0600)
		_, e = f.Write(append(data, '\n'))
		closeErr := f.Close()
		if e == nil {
			e = closeErr
		}
		if e != nil {
			fmt.Fprintln(os.Stderr, "evaluation: report write failed")
			return 2
		}
	} else {
		fmt.Println(string(data))
	}
	if !full.Passed {
		return 1
	}
	return 0
}
