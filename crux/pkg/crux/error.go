package crux

import (
	"fmt"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
)

// Err is the crux error type.
type Err struct {
	Err   string
	Stack string
}

// ErrE returns our error
func ErrE(err error) *Err {
	if err == nil {
		return nil
	}
	e := Err{Err: err.Error(), Stack: CallStack()}
	return &e
}

// ErrS returns an Err from a string
func ErrS(str string) *Err {
	if str == "" {
		return nil
	}
	e := Err{Err: str, Stack: CallStack()}
	return &e
}

// ErrF returns an Err from a fmt+args
func ErrF(f string, a ...interface{}) *Err {
	e := Err{Err: fmt.Sprintf(f, a...), Stack: CallStack()}
	return &e
}

func (e *Err) String() string {
	if e == nil {
		return ""
	}
	return e.Err
}

// Error implements the error interface
func (e *Err) Error() string {
	return e.String()
}

// Assert implements library sanity assertions
func Assert(val bool) {
	if !val {
		panic(nil)
	}
}

// FatalIfErr does a fatal and then exits
func FatalIfErr(log clog.Logger, err *Err) {
	if err == nil {
		return
	}
	if log != nil {
		log.Log("level", "fatal", "Fatal error: %s (stack %s)", err.Err, err.Stack)
	} else {
		fmt.Printf("Fatal error: %s (stack %s)\n", err.Err, err.Stack)
	}
	Exit(1)
}

// Err2Proto populates a protobuf Err field (see protos/err.proto)
func Err2Proto(e *Err) *pb.Err {
	if e == nil {
		return &pb.Err{Err: "", Stack: ""}
	}
	return &pb.Err{Err: e.Err, Stack: e.Stack}
}

// Proto2Err returns an Err from a protobuf Err (see protos/err.proto)
func Proto2Err(err *pb.Err) *Err {
	if err == nil || err.Err == "" {
		return nil
	}
	return &Err{Err: err.Err, Stack: err.Stack}
}
