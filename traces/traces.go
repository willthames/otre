package traces

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SpanID is an ID for a span
type SpanID string

// TraceID is an ID for a trace
type TraceID string

// Trace is a struct containing spans, a mapping of spanIDs to spans
type Trace struct {
	traceID TraceID
	spans   map[SpanID]types.Span
	sync.RWMutex
	version string
}

var (
	tracesInBuffer = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "otre_traces_in_buffer",
		Help: "The number of traces currently in the buffer",
	})
	spansInBuffer = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "otre_spans_in_buffer",
		Help: "The number of spans currently in the buffer",
	})
)

func (t *Trace) addSpan(span types.Span) {
	spanID := SpanID(span.ID)
	logrus.WithField("SpanID", spanID).WithField("TraceID", t.traceID).Debug("Locking trace")
	t.Lock()
	t.spans[spanID] = span
	spansInBuffer.Inc()
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
func (tb *TraceBuffer) AddSpan(span types.Span) {
	traceID := TraceID(span.TraceID)
	logrus.WithField("TraceID", traceID).Debug("RLocking TraceBuffer")
	tb.RLock()
	trace, ok := tb.Traces[traceID]
	logrus.WithField("TraceID", traceID).Debug("RUnlocking TraceBuffer")
	tb.RUnlock()
	if !ok {
		logrus.WithField("TraceID", traceID).Debug("Locking TraceBuffer")
		tb.Lock()
		tb.Traces[traceID] = NewTrace(traceID, []types.Span{span})
		tracesInBuffer.Inc()
		spansInBuffer.Inc()
		logrus.WithField("TraceID", traceID).Debug("Unlocking TraceBuffer")
		tb.Unlock()
	} else {
		trace.addSpan(span)
		spansInBuffer.Inc()
	}
}

// DeleteTrace deletes a trace from the trace buffer
func (tb *TraceBuffer) DeleteTrace(traceID TraceID) {
	tb.RLock()
	trace := tb.Traces[traceID]
	tb.RUnlock()
	spans := trace.Spans()
	for _, span := range spans {
		spanID := SpanID(span.ID)
		trace.Lock()
		delete(trace.spans, spanID)
		trace.Unlock()
		spansInBuffer.Dec()
	}
	tb.Lock()
	delete(tb.Traces, traceID)
	tb.Unlock()
	tracesInBuffer.Dec()
}

// NewTrace creates a Trace object from a list of Spans
func NewTrace(traceID TraceID, spans []types.Span) *Trace {
	trace := new(Trace)
	trace.traceID = traceID
	trace.spans = make(map[SpanID]types.Span)
	for _, span := range spans {
		trace.spans[SpanID(span.CoreSpanMetadata.ID)] = span
	}
	return trace
}

// MarshalJSON converts a Trace to a JSON string
func (t *Trace) MarshalJSON() ([]byte, error) {
	v := make([]string, len(t.spans))
	idx := 0
	t.RLock()
	defer t.RUnlock()
	var jsonSpan []byte
	var err error
	for _, span := range t.spans {
		jsonSpan, err = json.Marshal(span)
		if err != nil {
			return nil, err
		}
		v[idx] = string(jsonSpan[:])
		idx++
	}
	return []byte("[" + strings.Join(v, ",") + "]"), nil
}

// Spans converts a Trace to a list of spans
func (t *Trace) Spans() []types.Span {
	v := make([]types.Span, len(t.spans))
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
		parentID = SpanID(span.CoreSpanMetadata.ParentID)
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
		parentID = SpanID(span.CoreSpanMetadata.ParentID)
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
	var duration float64
	maximum := time.Unix(0, 0)
	t.RLock()
	defer t.RUnlock()
	for _, span := range t.spans {
		timestamp = span.Timestamp
		duration = span.DurationMs
		finish = timestamp.Add(time.Duration(int64(duration * 1E6)))
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
		parentID = SpanID(span.CoreSpanMetadata.ParentID)
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
