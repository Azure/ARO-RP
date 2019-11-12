package orderedmap

import (
	"encoding/json"
	"reflect"
	"testing"
)

type KeyValue struct {
	Key   string
	Value int
}

type KeyValues []KeyValue

func (xs *KeyValues) UnmarshalJSON(b []byte) error {
	return UnmarshalJSON(b, xs)
}

func (xs KeyValues) MarshalJSON() ([]byte, error) {
	return MarshalJSON(xs)
}

func TestExample(t *testing.T) {
	in := []byte(`{"a":1,"b":2}`)
	out := KeyValues{
		{
			Key:   "a",
			Value: 1,
		},
		{
			Key:   "b",
			Value: 2,
		},
	}

	var m KeyValues
	err := json.Unmarshal(in, &m)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m, out) {
		t.Errorf("got m %#v", m)
	}

	b, err := json.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(b, in) {
		t.Errorf("got b %s", string(b))
	}
}
