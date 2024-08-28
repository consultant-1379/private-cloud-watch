package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"text/template"
)

// PrintFormatData accepts a text/template string and
// an interface. If the string is non-empty, it uses it.
// If the string is empty, it pretty-prints JSON.
func PrintFormatData(format string, data interface{}) {
	if format != "" {
		t, err := template.New("format").Parse(format)
		ExitIfError(err)
		err = t.Execute(os.Stdout, &data)
		ExitIfError(err)
		fmt.Println("")
	} else {
		b, err := json.MarshalIndent(&data, "", "  ")
		ExitIfError(err)
		fmt.Println(string(b))
	}
}

// ExitIfError exits if the error passed in is not nil.
func ExitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

// HandleSignals registers signals to cancel a context
func HandleSignals(ctx context.Context, cancel context.CancelFunc, signals ...os.Signal) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, signals...)

	select {
	case <-ctx.Done():
	case s := <-signalChan:
		fmt.Printf("Got %v, shutting down...\n", s)
		cancel()
	}
}
