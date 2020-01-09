package traces

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

// SpanID is an ID for a span
type SpanID string

// TraceID is an ID for a trace
type TraceID string

// Trace is a struct containing spans, a mapping of spanIDs to spans
type Trace struct {
	spans map[SpanID]types.Span
}

// TraceBuffer is a mapping of TraceIDs to Traces
type TraceBuffer = map[TraceID]Trace

// NewTrace creates a Trace object from a list of Spans
func NewTrace(spans []types.Span) *Trace {
	trace := new(Trace)
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
	for _, span := range t.spans {
		jsonSpan, err := json.Marshal(span)
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
	for _, span := range t.spans {
		parentID = SpanID(span.CoreSpanMetadata.ParentID)
		if parentID != "" {
			_, ok := t.spans[parentID]
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
	result := []SpanID{}
	for _, span := range t.spans {
		parentID = SpanID(span.CoreSpanMetadata.ParentID)
		if parentID != "" {
			_, ok := t.spans[parentID]
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
	maximum := time.Unix(0, 0)
	for _, span := range t.spans {
		timestamp := span.Timestamp
		duration := span.DurationMs
		finish := timestamp.Add(time.Duration(int64(duration * 1E6)))
		if maximum.Before(finish) {
			maximum = finish
		}
	}
	return maximum.Before(abstime)
}

// OlderThanRelative checks whether the most recently completed span
// is older than a duration in milliseconds ago from now
func (t *Trace) OlderThanRelative(durationMs int64, now time.Time) bool {
	abstime := now.Add(time.Duration(int64(-durationMs * 1E6)))
	return t.olderThanAbsolute(abstime)
}
