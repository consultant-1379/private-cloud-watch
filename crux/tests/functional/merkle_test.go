package ftests

import (
	"testing"

	"github.com/erixzone/crux/pkg/begat/merkle"
)

func TestHash1(t *testing.T) {
	merkle.MyTestScenario1(t)
	merkle.MyTestScenario2(t)
	merkle.MyTestScenario3(t)
	merkle.MyTestScenario4(t)
	merkle.MyTestScenario5(t)
	merkle.MyTestScenario6(t)
	merkle.RmTree("mx")
}
