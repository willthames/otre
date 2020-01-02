package rules

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"path"
	"runtime"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

type testTrace struct {
	spans    []types.Span
	expected bool
}

func newTestTrace(traceFile string, expected bool) *testTrace {
	t := new(testTrace)
	byteTraces, err := ioutil.ReadFile(traceFile)
	t.spans = *new([]types.Span)
	err = json.Unmarshal(byteTraces, &t.spans)
	if err != nil {
		panic(err)
	}
	t.expected = expected
	for _, span := range t.spans {
		out, err := json.Marshal(span)
		if err != nil {
			logrus.WithField("out", out).Warn("newTestTrace")
		}
	}
	return t
}

func TestAcceptTrace(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	parentdir := path.Join(path.Dir(filename), "..")

	testTraces := [3]testTrace{
		*newTestTrace(path.Join(parentdir, "rules", "trace_5xx.json"), true),
		*newTestTrace(path.Join(parentdir, "rules", "trace_ping.json"), false),
		*newTestTrace(path.Join(parentdir, "rules", "trace_api_newservice.json"), true),
	}

	randomTraces := [8]testTrace{
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), false),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), false),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), false),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), false),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), false),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), true),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), false),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), false),
	}

	rules, err := ioutil.ReadFile("policy.rego")

	if err != nil {
		panic(err)
	}
	rulesengine := NewRulesEngine(string(rules))
	for _, trace := range testTraces {
		if rulesengine.acceptSpans(trace.spans) != trace.expected {
			t.Errorf("Result of acceptSpans for trace %v not as expected (%v)", trace.spans, trace.expected)
		}
	}
	// Seed 1 has 25, the critical edge case, in the first 8 values
	rand.Seed(1)
	for i, trace := range randomTraces {
		if rulesengine.acceptSpans(trace.spans) != trace.expected {
			t.Errorf("Result of acceptSpans for random trace #%d not as expected (%v)", i, trace.expected)
		}
	}
}
