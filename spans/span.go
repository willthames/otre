package spans

import (
	"encoding/json"
	"time"
)

// Span represents the Zipkin V1 Span object. See
// https://github.com/openzipkin/zipkin-api/blob/master/zipkin-api.yaml
type Span struct {
	TraceID           string                 `thrift:"trace_id,1" json:"traceId"`
	Name              string                 `thrift:"name,3" json:"name"`
	ID                string                 `thrift:"id,4" json:"id"`
	ParentID          string                 `thrift:"parent_id,5" json:"parentId,omitempty"`
	Annotations       []*annotation          `thrift:"annotations,6" json:"annotations"`
	BinaryAnnotations map[string]interface{} `thrift:"-" json:"-"`
	Debug             bool                   `thrift:"debug,9" json:"debug,omitempty"`
	Timestamp         time.Time              `thrift:"-" json:"-"`
	Duration          time.Duration          `thrift:"-" json:"-"`
	TraceIDHigh       *int64                 `thrift:"trace_id_high,12" json:"trace_id_high,omitempty"`
}

// JSONSpan is used as an intermediate step between json encoding and the Span tyhpe
type JSONSpan struct {
	spanAlias
	BinaryAnnotations []BinaryAnnotation `thrift:"binary_annotations,8" json:"binaryAnnotations"`
	Timestamp         int64              `thrift:"timestamp,10" json:"timestamp,omitempty"`
	Duration          int64              `thrift:"duration,11" json:"duration,omitempty"`
}

type spanAlias Span

type annotation struct {
	timestamp int64     `json:"timestamp"`
	value     string    `json:"value"`
	endpoint  *endpoint `json:"endpoint,omitempty"`
}

// Span converts a JSONSpan into Span after Unmarshalling
func (jsonspan JSONSpan) Span() Span {
	span := Span(jsonspan.spanAlias)
	span.Duration = time.Duration(jsonspan.Duration * 1E3)
	span.Timestamp = time.Unix(jsonspan.Timestamp/1E6, (jsonspan.Timestamp%1E6)*1E3)
	span.BinaryAnnotations = make(map[string]interface{}, len(jsonspan.BinaryAnnotations))
	for _, annotation := range jsonspan.BinaryAnnotations {
		span.BinaryAnnotations[annotation.key] = annotation.value
	}
	return span
}

// NewJSONSpan converts a Span into JSONSpan suitable for Marshalling
func NewJSONSpan(span Span) JSONSpan {
	return JSONSpan{
		spanAlias(span),
		convertAnnotations(span.BinaryAnnotations),
		span.Timestamp.UnixNano() / 1E3,
		span.Duration.Microseconds(),
	}
}

func convertAnnotations(annotations map[string]interface{}) []BinaryAnnotation {
	result := make([]BinaryAnnotation, len(annotations))
	for key, value := range annotations {
		result = append(result, BinaryAnnotation{key: key, value: value})
	}
	return result
}

type BinaryAnnotation struct {
	key      string      `json:"key"`
	value    interface{} `json:"value"`
	endpoint *endpoint   `json:"endpoint,omitempty"`
}

type endpoint struct {
	ipv4        string `json:"ipv4"`
	port        int    `json:"port"`
	serviceName string `json:"serviceName"`
}

func (s Span) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONSpan(s))
}

func (s *Span) UnmarshalJSON(data []byte) error {
	var js JSONSpan
	if err := json.Unmarshal(data, &js); err != nil {
		return err
	}
	*s = js.Span()
	return nil
}
