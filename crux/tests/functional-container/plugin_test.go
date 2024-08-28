package ftests

import (
	"fmt"
	"os/exec"
	"plugin"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/ruck"
)

func TestPlugin(t *testing.T) { TestingT(t) }

type pluginSuite struct {
}

func init() {
	Suite(&pluginSuite{})
}

func (k *pluginSuite) SetUpSuite(c *C) {
}

func (k *pluginSuite) TearDownSuite(c *C) {
}

func (k *pluginSuite) TestPlugin(c *C) {
	var nhb int
	fs := crux.Fservice{FuncName: "ExamplePlugin", FileName: "/crux/bin/example", Quit: make(chan bool, 2)}
	fsr := crux.FserviceReturn{}
	alive := make(chan []crux.Fservice, 5)
	fs.Alive = chan<- []crux.Fservice(alive)
	err := Start1(fs, &fsr)
	c.Assert(err, IsNil)
	done := make(chan bool)
	go func(ch chan []crux.Fservice, d chan bool) {
		nhb = 0
		for {
			slice := <-ch
			fmt.Printf("received %+v\n", slice)
			if slice == nil {
				d <- true
				return
			}
			nhb++
		}
	}(alive, done)
	// wait for a few heartbeats
	time.Sleep(10 * time.Second)
	fmt.Printf("quitting\n")
	// tell it to quit
	fs.Quit <- true
	fmt.Printf("sent quit\n")
	// wait til done
	<-done
	fmt.Printf("done received; nhb = %d\n", nhb)
	c.Assert(nhb >= 9, Equals, true)
}

func Start1(fs crux.Fservice, fsr *crux.FserviceReturn) error {
	image := fs.FileName
	if image == "" {
		image = ruck.DefaultExecutable
	}
	cmd := exec.Command("ls", "-l", "/crux", "/crux/bin")
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%+v %s\n", err, stdoutStderr)
	p, err := plugin.Open(image)
	if err != nil {
		return err
	}
	f, err := p.Lookup(fs.FuncName)
	if err != nil {
		fmt.Printf("looking for %s in %s\n", fs.FuncName, image)
		return err
	}
	go f.(func(<-chan bool, chan<- []crux.Fservice, **crux.Confab))(fs.Quit, fs.Alive, nil) // start it
	return nil
}
