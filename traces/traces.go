package traces

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/willthames/otre/rules"
	"github.com/willthames/otre/spans"
)

// SpanID is an ID for a span
type SpanID string

// TraceID is an ID for a trace
type TraceID string

// Trace is a struct containing spans, a mapping of spanIDs to spans
type Trace struct {
	traceID TraceID
	spans   map[SpanID]spans.Span
	sync.RWMutex
	version        string
	SampleResult   *rules.SampleResult
	SampleDecision bool
}

// TraceBufferMetrics returns the net change in spans and traces in a TraceBuffer
type TraceBufferMetrics struct {
	SpanDelta  int
	TraceDelta int
}

func (t *Trace) addSpan(span spans.Span) {
	spanID := SpanID(span.ID)
	logrus.WithField("SpanID", spanID).WithField("TraceID", t.traceID).Debug("Locking trace")
	t.Lock()
	t.spans[spanID] = span
	logrus.WithField("SpanID", spanID).WithField("TraceID", t.traceID).Debug("Unlocking trace")
	t.Unlock()
}

// TraceBuffer is a mapping of TraceIDs to Traces
type TraceBuffer struct {
	Traces map[TraceID]*Trace
	sync.RWMutex
}

// NewTraceBuffer creates a new TraceBuffer
func NewTraceBuffer() *TraceBuffer {
	traceBuffer := new(TraceBuffer)
	traceBuffer.Traces = make(map[TraceID]*Trace)
	return traceBuffer
}

// AddSpan adds a span to a TraceBuffer, creating
// a new trace if the trace isn't yet in the TraceBuffer
func (tb *TraceBuffer) AddSpan(span spans.Span) TraceBufferMetrics {
	traceID := TraceID(span.TraceID)
	tbm := *new(TraceBufferMetrics)
	logrus.WithField("TraceID", traceID).Debug("RLocking TraceBuffer")
	tb.RLock()
	trace, ok := tb.Traces[traceID]
	logrus.WithField("TraceID", traceID).Debug("RUnlocking TraceBuffer")
	tb.RUnlock()
	if !ok {
		logrus.WithField("TraceID", traceID).Debug("Locking TraceBuffer")
		tb.Lock()
		tb.Traces[traceID] = NewTrace(traceID, []spans.Span{span})
		tbm.SpanDelta = 1
		tbm.TraceDelta = 1
		logrus.WithField("TraceID", traceID).Debug("Unlocking TraceBuffer")
		tb.Unlock()
	} else {
		trace.RLock()
		_, ok = trace.spans[SpanID(span.ID)]
		trace.RUnlock()
		trace.addSpan(span)
		if !ok {
			tbm.SpanDelta = 1
		}
	}
	return tbm
}

// DeleteTrace deletes a trace from the trace buffer
func (tb *TraceBuffer) DeleteTrace(traceID TraceID) TraceBufferMetrics {
	tbm := *new(TraceBufferMetrics)
	tb.RLock()
	trace := tb.Traces[traceID]
	tb.RUnlock()
	spans := trace.Spans()
	for _, span := range spans {
		spanID := SpanID(span.ID)
		trace.Lock()
		delete(trace.spans, spanID)
		trace.Unlock()
		tbm.SpanDelta--
	}
	tb.Lock()
	delete(tb.Traces, traceID)
	tb.Unlock()
	tbm.TraceDelta = -1
	return tbm
}

// NewTrace creates a Trace object from a list of Spans
func NewTrace(traceID TraceID, spanlist []spans.Span) *Trace {
	trace := new(Trace)
	trace.traceID = traceID
	trace.spans = make(map[SpanID]spans.Span)
	for _, span := range spanlist {
		trace.spans[SpanID(span.ID)] = span
	}
	return trace
}

// MarshalJSON converts a Trace to a JSON string
func (t *Trace) MarshalJSON() ([]byte, error) {
	t.RLock()
	defer t.RUnlock()
	return json.Marshal(t.spans)
}

// Spans converts a Trace to a list of spans
func (t *Trace) Spans() []spans.Span {
	v := make([]spans.Span, len(t.spans))
	idx := 0
	t.RLock()
	defer t.RUnlock()
	for _, span := range t.spans {
		v[idx] = span
		idx++
	}
	return v
}

// IsComplete checks if all spans in a trace have
// parents (leaves can potentially be missing but that is impossible
// to detect)
func (t *Trace) IsComplete() bool {
	var parentID SpanID
	t.RLock()
	defer t.RUnlock()
	for _, span := range t.spans {
		parentID = SpanID(span.ParentID)
		var ok bool
		if parentID != "" {
			_, ok = t.spans[parentID]
			if !ok {
				return false
			}
		}
	}
	return true
}

// MissingSpans returns the SpanIDs of all spans that are
// a parent ID of a child span but not present in the trace
func (t *Trace) MissingSpans() []SpanID {
	var parentID SpanID
	var ok bool
	result := []SpanID{}
	t.RLock()
	defer t.RUnlock()
	for _, span := range t.spans {
		parentID = SpanID(span.ParentID)
		if parentID != "" {
			_, ok = t.spans[parentID]
			if !ok {
				result = append(result, parentID)
			}
		}
	}
	return result
}

// olderThanAbsolute checks whether the most recently completed span
// is older than an absolute timestamp
func (t *Trace) olderThanAbsolute(abstime time.Time) bool {
	var timestamp, finish time.Time
	var duration time.Duration
	maximum := time.Unix(0, 0)
	t.RLock()
	defer t.RUnlock()
	for _, span := range t.spans {
		timestamp = span.Timestamp
		duration = span.Duration
		finish = timestamp.Add(duration)
		if maximum.Before(finish) {
			maximum = finish
		}
	}
	return maximum.Before(abstime)
}

// OlderThanRelative checks whether the most recently completed span
// is older than a time.Duration ago from now
func (t *Trace) OlderThanRelative(duration time.Duration, now time.Time) bool {
	abstime := now.Add(-duration)
	return t.olderThanAbsolute(abstime)
}

func (t *Trace) rootSpanID() (SpanID, error) {
	var parentID SpanID
	t.RLock()
	defer t.RUnlock()
	for spanID, span := range t.spans {
		parentID = SpanID(span.ParentID)
		if parentID == "" {
			return spanID, nil
		}
	}
	return "", fmt.Errorf("Couldn't find root span")
}

// AddStringTag adds a key-value binary annotation to a trace
func (t *Trace) AddStringTag(key string, value string) error {
	rootSpanID, err := t.rootSpanID()
	if err != nil {
		return err
	}
	t.spans[rootSpanID].BinaryAnnotations[key] = value
	return nil
}

// AddIntTag adds a key-value binary annotation to a trace
func (t *Trace) AddIntTag(key string, value int) error {
	rootSpanID, err := t.rootSpanID()
	if err != nil {
		return err
	}
	t.spans[rootSpanID].BinaryAnnotations[key] = value
	return nil
}
