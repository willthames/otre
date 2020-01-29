package spans

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
	thrift "github.com/thrift-iterator/go"
)

// Span represents the Zipkin V1 Span object. See
// https://github.com/openzipkin/zipkin-api/blob/master/zipkin-api.yaml
type Span struct {
	TraceID           string
	Name              string
	ID                string
	ParentID          string
	Annotations       []*annotation
	BinaryAnnotations map[string]string
	Debug             bool
	Timestamp         time.Time
	Duration          time.Duration
	TraceIDHigh       *int64
}

// v1Span is used as an intermediate step between encoding and the Span tyhpe
type v1Span struct {
	TraceID           string             `thrift:"trace_id,1" json:"traceId"`
	Name              string             `thrift:"name,3" json:"name"`
	ID                string             `thrift:"id,4" json:"id"`
	ParentID          string             `thrift:"parent_id,5" json:"parentId,omitempty"`
	Annotations       []*annotation      `thrift:"annotations,6" json:"annotations"`
	Debug             bool               `thrift:"debug,9" json:"debug,omitempty"`
	TraceIDHigh       *int64             `thrift:"trace_id_high,12" json:"trace_id_high,omitempty"`
	BinaryAnnotations []BinaryAnnotation `thrift:"binary_annotations,8" json:"binary_annotations"`
	Timestamp         int64              `thrift:"timestamp,10" json:"timestamp,omitempty"`
	Duration          int64              `thrift:"duration,11" json:"duration,omitempty"`
}

type annotation struct {
	timestamp int64     `json:"timestamp"`
	value     string    `json:"value"`
	endpoint  *endpoint `json:"endpoint,omitempty"`
}

type BinaryAnnotation struct {
	Key      string    `thrift:"key,13" json:"key"`
	Value    string    `thrift:"value,14" json:"value"`
	endpoint *endpoint `json:"endpoint,omitempty"`
}

type endpoint struct {
	ipv4        string `json:"ipv4"`
	port        int    `json:"port"`
	serviceName string `json:"serviceName"`
}

// Span converts a JSONSpan into Span after Unmarshalling
func (v1span v1Span) Span() Span {
	span := Span{
		TraceID:     v1span.TraceID,
		Name:        v1span.Name,
		ID:          v1span.ID,
		ParentID:    v1span.ParentID,
		Annotations: v1span.Annotations,
		Debug:       v1span.Debug,
		TraceIDHigh: v1span.TraceIDHigh,
	}
	span.Duration = time.Duration(v1span.Duration * 1E3)
	span.Timestamp = time.Unix(v1span.Timestamp/1E6, (v1span.Timestamp%1E6)*1E3)
	span.BinaryAnnotations = make(map[string]string, len(v1span.BinaryAnnotations))
	for _, annotation := range v1span.BinaryAnnotations {
		span.BinaryAnnotations[annotation.Key] = annotation.Value
	}
	return span
}

// NewJSONSpan converts a Span into JSONSpan suitable for Marshalling
func newV1Span(span Span) v1Span {
	binaryAnnotations := convertAnnotations(span.BinaryAnnotations)
	timestamp := span.Timestamp.UnixNano() / 1E3
	duration := span.Duration.Microseconds()
	v1span := v1Span{
		TraceID:           span.TraceID,
		Name:              span.Name,
		ID:                span.ID,
		ParentID:          span.ParentID,
		Annotations:       span.Annotations,
		Debug:             span.Debug,
		TraceIDHigh:       span.TraceIDHigh,
		BinaryAnnotations: binaryAnnotations,
		Timestamp:         timestamp,
		Duration:          duration,
	}
	return v1span
}

func convertAnnotations(annotations map[string]string) []BinaryAnnotation {
	result := make([]BinaryAnnotation, len(annotations))
	index := 0
	for key, value := range annotations {
		result[index] = BinaryAnnotation{Key: key, Value: value}
		index++
	}
	return result
}

func (s Span) MarshalJSON() ([]byte, error) {
	return json.Marshal(newV1Span(s))
}

func (s *Span) UnmarshalJSON(data []byte) error {
	var v1span v1Span
	if err := json.Unmarshal(data, &v1span); err != nil {
		return err
	}
	logrus.WithField("span", v1span).Trace("Unmarshalled span from json")
	*s = v1span.Span()
	return nil
}

func (s Span) MarshalThrift() ([]byte, error) {
	v1span := newV1Span(s)
	return thrift.Marshal(v1span)
}

func (s *Span) UnmarshalThrift(data []byte) error {
	var v1span v1Span
	if err := thrift.Unmarshal(data, &v1span); err != nil {
		return err
	}
	logrus.WithField("span", v1span).Trace("Unmarshalled span from thrift")
	*s = v1span.Span()
	return nil
}
