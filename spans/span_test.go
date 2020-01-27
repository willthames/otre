package spans

import (
	"encoding/json"
	"testing"
	"time"
)

func testJSONUnmarshal(t *testing.T) {
	b := []byte(`{"id":1,"name":"bowser","timestamp":1480979203000000,"binaryAnnotations":[{"key":"hello","value":"world"}],"duration":1000}`)
	span := new(Span)
	json.Unmarshal(b, &span)
	if span.Duration != 1000*time.Microsecond {
		t.Errorf("span duration was incorrectly parsed")
	}
	value, ok := span.BinaryAnnotations["hello"]
	if !ok || value != "world" {
		t.Errorf("binary annotations incorrectly parsed")
	}
}
