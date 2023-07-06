package nilinterface

import (
	"errors"
	"fmt"
)

type BB interface {
	BB() int
}

type Me struct {
	field int
}

func (a *Me) BB() int {
	return a.field // this is key to get a later panic
}

func main() {
	var a BB
	a, err := NewMe()
	if err != nil {
		fmt.Println("Error expected. Good")
		a = nil
	}

	if a != nil { // want "nilinterfacecheck comparing nil against an interface is usually a mistake"
		fmt.Println("Go told us it is safe to use 'a' ... and now you will see a panic. Awesome :( ")
		fmt.Println(a.BB())
	}
}

func NewMe() (BB, error) {
	return nested()
}

func nested() (*Me, error) {
	return nil, errors.New("failed")
}

func testNilComparison() bool {
	var a *Me = nil
	if a == nil {
		return false
	}

	return true
}

func testIdentNotNill() bool {
	var a string = "b"
	var b string = "c"
	if a == b {
		return false
	}
	return true
}

func iRetNil() BB {
	return nil
}

func iRetNoth() BB {
	var a *Me
	return a
}

func iRetAnotherNil() *Me {
	return nil
}

func iRetAnotherNoth() *Me {
	var a *Me
	return a
}

func testCompareFunc() {

	if iRetNil() == nil { // want "nilinterfacecheck comparing nil against an interface is usually a mistake"
	}
}

func testCompareFuncNoth() {
	if iRetNoth() == nil { // want "nilinterfacecheck comparing nil against an interface is usually a mistake"
	}
}

func testNilPointerShouldntTrigger() {
	if iRetAnotherNil() == nil {

	}
}

func testEmptyNilPointerShouldntTrigger() {
	if nil == iRetAnotherNoth() {

	}
}
