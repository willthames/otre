package traces

import (
	"testing"
	"time"

	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

func TestCompleteSpan(t *testing.T) {
	trace := new(Trace)
	trace.spans = make(map[SpanID]types.Span)
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "root"}})
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "child", ParentID: "root"}})
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}})
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild2", ParentID: "child"}})

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
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "root"}})
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}})
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild2", ParentID: "child"}})

	missingSpans := trace.MissingSpans()
	if len(missingSpans) != 1 && missingSpans[0] != "child" {
		t.Errorf("Inomplete span should return one and only one missing span (not %v)", missingSpans)
	}
	if trace.IsComplete() {
		t.Errorf("Incomplete span should return false for IsComplete")
	}
}

func TestIsOlderThanAbsolute(t *testing.T) {
	starttime := time.Date(2020, time.January, 8, 9, 0, 0, 0, time.UTC)
	endtime := time.Date(2020, time.January, 8, 9, 0, 1, 0, time.UTC)
	trace := new(Trace)
	trace.spans = make(map[SpanID]types.Span)
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", DurationMs: 1000, ID: "a"}, Timestamp: starttime})
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", DurationMs: 800, ID: "b"}, Timestamp: starttime})
	if trace.olderThanAbsolute(starttime) {
		t.Errorf("trace should not be older than the start time")
	}
	if trace.olderThanAbsolute(endtime) {
		t.Errorf("trace should not be older than the end time")
	}
	if !trace.olderThanAbsolute(endtime.Add(time.Duration(1E6))) {
		t.Errorf("trace should be older than the end time plus a millisecond")
	}
}

func TestParentSpanID(t *testing.T) {
	trace := new(Trace)
	trace.spans = make(map[SpanID]types.Span)
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}})
	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild2", ParentID: "child"}})

	rootSpanID, err := trace.rootSpanID()
	if err == nil || rootSpanID != "" {
		t.Errorf("Trace missing root span should return error for rootSpanID()")
	}

	trace.addSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "root"}})
	rootSpanID, err = trace.rootSpanID()
	if err != nil || rootSpanID == "" {
		t.Errorf("rootSpanID() should return a root span ID and no error")
	}
}

func TestTraceBuffer(t *testing.T) {
	tbm := *new(TraceBufferMetrics)
	traceBuffer := NewTraceBuffer()
	tbm = traceBuffer.AddSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}})
	if tbm.TraceDelta != 1 || tbm.SpanDelta != 1 {
		t.Errorf("Adding new trace with single span should cause span and trace deltas of 1")
	}
	tbm = traceBuffer.AddSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "root"}})
	if tbm.TraceDelta != 0 || tbm.SpanDelta != 1 {
		t.Errorf("Adding single span to existing trace should cause span delta of 1")
	}
	traceBuffer.AddSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "child", ParentID: "root"}})
	traceBuffer.AddSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild2", ParentID: "child"}})
	tbm = traceBuffer.DeleteTrace(TraceID("trace"))
	if tbm.TraceDelta != -1 || tbm.SpanDelta != -4 {
		t.Errorf("Deleting trace with four spans should cause span delta -4 and trace delta of -1")
	}
}

func TestDuplicateSpan(t *testing.T) {
	tbm := *new(TraceBufferMetrics)
	traceBuffer := NewTraceBuffer()
	tbm = traceBuffer.AddSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}})
	if tbm.TraceDelta != 1 || tbm.SpanDelta != 1 {
		t.Errorf("Adding new trace with single span should cause span and trace deltas of 1")
	}
	tbm = traceBuffer.AddSpan(types.Span{CoreSpanMetadata: types.CoreSpanMetadata{TraceID: "trace", ID: "grandchild1", ParentID: "child"}})
	if tbm.TraceDelta != 0 || tbm.SpanDelta != 0 {
		t.Errorf("Adding duplicate span should cause span and trace deltas of 0")
	}
}
