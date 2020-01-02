package traces

import (
	"encoding/json"
	"strings"

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
		_, ok := t.spans[parentID]
		if !ok {
			return false
		}
	}
	return true
}
