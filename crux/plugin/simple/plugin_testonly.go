package main

import "fmt"

//  For Test purposes only.

func main() {
	fmt.Printf("Not intended to run this func.")

}

// PluginTest - doesn't require any crux packages to avoid triggering go plugin errors.
func PluginTest() {
	fmt.Printf("====== Function PluginTest Called ====\n")
}
