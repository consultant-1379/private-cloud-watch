package merkle

import (
	"testing"
)

func TestHash1(t *testing.T) {
	MyTestScenario1(t)
	/* defer these to functional tests
	MyTestScenario2(t)
	MyTestScenario3(t)
	MyTestScenario4(t)
	MyTestScenario5(t)
	MyTestScenario6(t)
	*/
	RmTree("mx")
}
