package traces

import (
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	"testing"
)

func TestCompleteSpan(t *testing.T) {
	trace := new(Trace)
	trace.spans = make(map[SpanID]types.Span)
	trace.spans["root"] = types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "root"}}
	trace.spans["child"] = types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "child", ParentID: "root"}}
	trace.spans["grandchild1"] = types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}}
	trace.spans["grandchild2"] = types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild2", ParentID: "child"}}

	missingSpans := trace.MissingSpans()
	if len(missingSpans) != 0 {
		t.Errorf("Complete span should not have missing spans %v", missingSpans)
	}
	if !trace.IsComplete() {
		t.Errorf("Complete span should return true for IsComplete")
	}
}

func TestIncompleteSpan(t *testing.T) {
	trace := new(Trace)
	trace.spans = make(map[SpanID]types.Span)
	trace.spans["root"] = types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "root"}}
	trace.spans["grandchild1"] = types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}}
	trace.spans["grandchild2"] = types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild2", ParentID: "child"}}

	missingSpans := trace.MissingSpans()
	if len(missingSpans) != 1 && missingSpans[0] != "child" {
		t.Errorf("Inomplete span should return one and only one missing span (not %v)", missingSpans)
	}
	if trace.IsComplete() {
		t.Errorf("Incomplete span should return false for IsComplete")
	}
}
