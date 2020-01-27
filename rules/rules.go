package rules

import (
	"context"
	"encoding/json"
	"math/rand"

	"github.com/open-policy-agent/opa/rego"
	"github.com/sirupsen/logrus"
	"github.com/willthames/otre/spans"
)

// RulesEngine is used to test traces against a policy
type RulesEngine struct {
	query rego.PreparedEvalQuery
	ctx   context.Context
}

// SampleResult expresses the result from the policy of
// what rule matched and the appropriate sample rate
type SampleResult struct {
	SampleRate int    `json:"sampleRate"`
	Reason     string `json:"reason"`
}

// NewRulesEngine creates a rules engine with a policy
// defined by the policy argument
func NewRulesEngine(policy string) *RulesEngine {
	var err error
	r := new(RulesEngine)
	r.ctx = context.Background()
	r.query, err = rego.New(
		rego.Query("response = data.otre.response"),
		rego.Module("accept.rego", policy),
	).PrepareForEval(r.ctx)
	if err != nil {
		logrus.WithError(err).Error("Error creating rules engine")
	}
	return r
}

func (r *RulesEngine) sampleSpans(spans []spans.Span) *SampleResult {
	results, err := r.query.Eval(r.ctx, rego.EvalInput(spans))
	defaultResult := &SampleResult{SampleRate: 100, Reason: "Unexpected response, default to accept"}

	if err != nil {
		logrus.WithError(err).WithField("spans", spans)
		return defaultResult
	} else if len(results) == 0 {
		logrus.WithField("spans", spans).Warn("No results returned")
		return defaultResult
	}
	response, ok := results[0].Bindings["response"].(map[string]interface{})
	if !ok {
		logrus.WithField("spans", spans).WithField("results", results).Warn("Unexpected result returned")
		return defaultResult
	}
	sampleRate, err := response["sampleRate"].(json.Number).Int64()
	if err != nil {
		logrus.WithError(err).WithField("spans", spans).WithField("results", results).Warn("Unexpected result returned")
		return defaultResult
	}
	var reason string
	reason, ok = response["reason"].(string)
	if !ok {
		logrus.WithField("spans", spans).WithField("results", results).Warn("Unexpected result returned")
		return defaultResult
	}
	return &SampleResult{SampleRate: int(sampleRate), Reason: reason}
}

// AcceptSpans checks whether a set of spans is accepted by the rules
// engine or not
func (r *RulesEngine) AcceptSpans(spans []spans.Span) (decision bool, sample *SampleResult) {
	sample = r.sampleSpans(spans)
	decision = (rand.Intn(100) < sample.SampleRate)
	return
}
