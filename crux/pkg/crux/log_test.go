package crux

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"
)

func TestClog(t *testing.T) { TestingT(t) }

type clogSuite struct {
	x int // not used
}

func init() {
	Suite(&clogSuite{})
}

func (k *clogSuite) SetUpSuite(c *C) {
}

func (k *clogSuite) TearDownSuite(c *C) {
}

func (k *clogSuite) TestClog1(c *C) {

	fmt.Printf(" =========== Demo GoKit style API ================\n")

	cfgLog := GetLogger()
	// Basic logging test
	cfgLog.Log("shinyInteger", 5, "We-Feel", "really Super")

	// Severity level test:  Yes, severity (aka level) is being set through  a tag
	cfgLog.Log("SEV", "DEBUG", "shinyInteger", 7, "Action", "SHOULDN'T PRINT, DEBUG NOT SET.")
	fmt.Printf("\nDEMO: Setting level to debug so DEBUG statements show\n")
	fmt.Printf("DEMO: (debug,info,warn,error,fatal,panic)\n")
	cfgLog.SetDebug()
	cfgLog.Log("SEV", "DEBUG", "shinyInteger", 7, "Action", "Will print")

	// TAG'd statements default to INFO level like any other
	// But the output threshold is separate from the logger's main threshold,
	// and defaults to defaultTagThreshold
	commonTag := "MEMORY"
	// Trigger output since default statement level is INFO.
	//cfgLog.SetTagLevel(commonTag, walrus.InfoLevel)
	cfgLog.Log("TAG", []string{commonTag, "COMPILER"}, "shinyInteger", 99, "TestType", "tagging")
	cfgLog.Log("TAG", []string{"COMPILER"}, "shinyInteger", 99, "TestType", "tagging - SHOULDN'T DISPLAY.  Default tag threshold too low.")
	cfgLog.Log("AnotherInteger", 101, "This is the last param, and odd, therefore should get converted to a message by the magic Log() call")

	// Set output level to info
	// do mix of info and debug logs.  debug should NOT output

	// Use With() to create a new logger w/ preset values

	//NOTE: Log.With() returns a Entry, which points back to the original
	//logger This means that debug level changes to this "logger" will
	//affect both it and the original.  Thankfully, It's not a common
	//pattern to have multiple loggers outputting at different levels
	dbgLog := cfgLog.Log("SEV", "INFO", "WITH-Int", 777, "With-Str", "with works!")
	fmt.Printf("\nDEMO: Setting level to info\n")
	dbgLog.SetInfo()
	//dbgLog := cfgLog.With("WITH-Int", 777, "With-Str", "with works!")
	dbgLog.Log("Just a message, since everything else set earlier use With()")
	dbgLog.Log("Another message.  Should log with same preset fields.")

	dbgLog.Log("SEV", "DEBUG", "shinyInteger", 7, "Action", "Shouldn't print if logger level is INFO.  Also shows override of settings for earlier With()")
}
