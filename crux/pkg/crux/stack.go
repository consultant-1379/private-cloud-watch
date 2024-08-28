package crux

import (
	"fmt"

	"github.com/go-stack/stack"
)

// CallStack returns a nice compressed call stack.
func CallStack() string {
	myfn := fmt.Sprintf // get around go vet enthusiasm
	ret := ""
	stack := stack.Trace()
	stack = stack[1:]
	// trim the annoying goexit at top
	n := len(stack)
	if myfn("%n", stack[n-1]) == "goexit" {
		stack = stack[:n-1]
	}
	for _, e := range stack {
		ret += myfn("<-%+n(%s:%d)", e, e, e)
	}
	return ret[2:]
}
