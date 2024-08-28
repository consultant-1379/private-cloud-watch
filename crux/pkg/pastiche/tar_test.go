package pastiche

import (
	"fmt"
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

type TarTest struct {
	targetDir string
}

func init() {
	tt := TarTest{}
	Suite(&tt)
}

// TestTarFuncs - verifes that untar and tar functions work.
func (tt *TarTest) TestTarFuncs(c *C) {
	cwd, _ := os.Getwd()
	c.Logf("Test running in dir: %s \n", cwd)
	os.Chdir("testdata")

	// Tar from directory to stream and/or file
	// then show untar works and data is the same
	newTarFile := "NewTar.tgz"
	fWriter, err := os.Create(newTarFile)
	c.Assert(nil, Equals, err)
	defer os.Remove(newTarFile)

	strm, err := NewTarStreamer(fWriter)
	c.Assert(nil, Equals, err)

	sourceDir := "OrigDir"
	err = strm.SendDir(sourceDir)
	c.Assert(nil, Equals, err)

	// Untar test from the created tar file just created
	sourcePath := newTarFile

	rdr, err := os.Open(sourcePath)
	c.Assert(nil, Equals, err)

	destDir, err := ioutil.TempDir("/tmp", "untar-outputdir")
	if err != nil {
		fmt.Printf("Error creating temp dir:%v\n", err)
		c.Fail()
	}

	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		err = os.Mkdir(destDir, 0777)
		c.Assert(err, Equals, nil)
	}

	bytes, topDir, err := Untar(rdr, destDir, true)
	c.Assert(nil, Equals, err)
	c.Logf("Wrote %d bytes from tar to disk. Tar's top dir is: %s\n", bytes, topDir)

}
