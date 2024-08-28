package main

import (
	"fmt"
	"time"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/pastiche"
)

func main() {
	fmt.Printf("Not intended to run this func.")

}

// PastichePluginTest - use to verify plugin binary is available and compiled correctly.
func PastichePluginTest() {
	fmt.Printf("====== Function PastichePluginTest Called ====\n")
}

// PastichePlugin  - To be called as a plugin function, usually in a go-routine.
func PastichePlugin(quit chan bool, svcs chan *crux.Err, period time.Duration, ch chan []crux.Fservice) {
	// The plugin function will not return until it is sent "true" on the quit/done channel.
	pastiche.PluginFunc(quit, svcs, period, ch)
	return

}
