package rules

import (
	"context"
	"math/rand"

	"github.com/Sirupsen/logrus"
	honey "github.com/honeycombio/honeycomb-opentracing-proxy/types"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"
	"github.com/willthames/otre/traces"
)

// RulesEngine is used to test traces against a policy
type RulesEngine struct {
	query rego.PreparedEvalQuery
	ctx   context.Context
}

// NewRulesEngine creates a rules engine with a policy
// defined by the policy argument
func NewRulesEngine(policy string) *RulesEngine {
	var err error
	r := new(RulesEngine)
	r.ctx = context.Background()
	r.query, err = rego.New(
		rego.Query("accept = data.otre.accept"),
		rego.Module("accept.rego", policy),
		rego.Function1(
			&rego.Function{
				Name: "percentChance",
				Decl: types.NewFunction(types.Args(types.N), types.B),
			},
			func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
				var result bool
				if chance, ok := a.Value.(ast.Number); ok {
					if ast.Compare(chance, ast.IntNumberTerm(0)) == 0 {
						result = false
					} else {
						roll := ast.IntNumberTerm(rand.Intn(100) + 1)
						// 1% is roll == 0, chance == 1
						// 2% is roll in [0,1], chance == 2
						result = (ast.Compare(roll, chance) < 0)
					}
					return ast.BooleanTerm(result), nil
				}
				return nil, nil
			}),
	).PrepareForEval(r.ctx)
	if err != nil {
		logrus.WithError(err).Error("Error creating rules engine")
	}
	return r
}

func (r *RulesEngine) acceptSpans(spans []honey.Span) bool {
	results, err := r.query.Eval(r.ctx, rego.EvalInput(spans))

	if err != nil {
		logrus.WithError(err).WithField("spans", spans)
	} else if len(results) == 0 {
		logrus.WithField("spans", spans).Warn("No results returned")
		// Handle undefined result.
	} else if result, ok := results[0].Bindings["accept"].(bool); !ok {
		logrus.WithField("spans", spans).WithField("results", results).Warn("Unexpected result returned")
	} else {
		return result
	}
	// default to accept
	return true
}

// AcceptTrace checks whether trace is accepted by the rules
// engine or not
func (r *RulesEngine) AcceptTrace(trace traces.Trace) bool {
	spans := trace.Spans()
	return r.acceptSpans(spans)
}
