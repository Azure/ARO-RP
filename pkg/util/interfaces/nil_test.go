package interfaces

import (
	"testing"
)

type xInterface interface{}
type yStruct struct{}

func wrongIsNil(x xInterface) bool {
	//go-staticheck does not report anything
	return x == nil
}

func TestIsNil(t *testing.T) {
	var x xInterface
	var y *yStruct

	x = y

	if y != nil {
		t.Fatal("Wrong test setup")
	}

	if x == nil { //nolint to be able to demonstrate the static check
		//go-staticheck reports: this comparison is never true; the lhs of the comparison has been assigned a concretely typed value (SA4023)
		t.Fatal("If we reach this the whole package is obsolete")
	}

	if wrongIsNil(x) {
		t.Fatal("If we reach this the whole package is obsolete")
	}

	if !IsNil(x) {
		t.Fatal("Should be true")
	}
}
