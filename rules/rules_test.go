package rules

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"path"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/willthames/otre/spans"
)

type testTrace struct {
	spans    []spans.Span
	expected int
}

func newTestTrace(traceFile string, expected int) *testTrace {
	t := new(testTrace)
	byteTraces, err := ioutil.ReadFile(traceFile)
	t.spans = *new([]spans.Span)
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

	testTraces := [5]testTrace{
		*newTestTrace(path.Join(parentdir, "rules", "trace_5xx.json"), 100),
		*newTestTrace(path.Join(parentdir, "rules", "trace_ping.json"), 0),
		*newTestTrace(path.Join(parentdir, "rules", "trace_ping_5xx.json"), 0),
		*newTestTrace(path.Join(parentdir, "rules", "trace_api_newservice.json"), 100),
		*newTestTrace(path.Join(parentdir, "rules", "trace_normal.json"), 25),
	}

	rules, err := ioutil.ReadFile("policy.rego")

	if err != nil {
		panic(err)
	}
	rulesengine := NewRulesEngine(string(rules))
	for _, trace := range testTraces {
		sampleResult := rulesengine.sampleSpans(trace.spans)
		if sampleResult.SampleRate != trace.expected {
			t.Errorf("Result of acceptSpans for trace %v not as expected (%v), reason: %v", trace.spans, trace.expected, sampleResult.Reason)
		}
	}
	// Seed 1 has 25, the critical edge case, in the first 8 values
	rand.Seed(1)
	for i := 0; i < 8; i++ {
		decision, _ := rulesengine.AcceptSpans(testTraces[4].spans)
		if i == 5 && !decision {
			t.Errorf("Result of AcceptTrace for random trace #%d not as expected (%v)", i, true)

		} else if i != 5 && decision {
			t.Errorf("Result of AcceptTrace for random trace #%d not as expected (%v)", i, false)
		}
	}
}
