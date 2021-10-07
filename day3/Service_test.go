package day3

import (
	"fmt"
	"testing"
)


type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// it's not a exported Method
func (f Foo) sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: "+msg, v...))
	}
}
func TestNewObject(t *testing.T) {
	var foo Foo
	s := NewObject(foo)
	_assert(len(s.methods) == 1, "wrong service Method, expect 1, but got %d", len(s.methods))
	mType := s.methods["Sum"]
	_assert(mType != nil, "wrong Method, Sum shouldn't nil")
}
