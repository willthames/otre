package spans

import (
	"testing"
	"time"
)

func TestJSONUnmarshal(t *testing.T) {
	b := []byte(`{"id":"hello","name":"bowser","timestamp":1480979203000000,"binary_annotations":[{"key":"hello","value":"world"}],"duration":1000}`)
	span := new(Span)
	err := span.UnmarshalJSON(b)
	if err != nil {
		t.Errorf("Failed to unmarshal span from json: %v", err)
	}
	if span.Duration != 1000*time.Microsecond {
		t.Errorf("span duration was incorrectly parsed")
	}
	value, ok := span.BinaryAnnotations["hello"]
	if !ok || value != "world" {
		t.Errorf("binary annotations incorrectly parsed")
	}
}

func TestThriftMarshall(t *testing.T) {
	span := Span{ID: "hello", TraceID: "world", Timestamp: time.Now()}
	span.BinaryAnnotations = make(map[string]string)
	span.BinaryAnnotations["this"] = "that"
	b, err := span.MarshalThrift()
	if err != nil {
		t.Errorf("Failed to marshal span to thrift")
	}
	span2 := new(Span)
	err = span2.UnmarshalThrift(b)
	if err != nil {
		t.Errorf("Failed to unmarshal span from thrift")
	}
	if span.ID != span2.ID || span.TraceID != span2.TraceID || span.BinaryAnnotations["this"] != span2.BinaryAnnotations["this"] {
		t.Errorf("span %v doesn't equal span2 %v", span, span2)
	}
}
