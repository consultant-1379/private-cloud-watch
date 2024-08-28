// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/sha3"

	"github.com/erixzone/crux/pkg/begat/common"
)

var retainCmd = &cobra.Command{
	Use:   "retain",
	Short: "Store bytes in the given file with the key of their sha-3 checksum",
	Long:  `nada`,
	Run: func(cmd *cobra.Command, args []string) {
		work := pasticheRoot
		if viper.GetBool("clear") {
			os.RemoveAll(work)
		}
		os.Mkdir(work, 0777) // just make it; we don't care if it fails because its there.
		s := common.NewStein()
		for _, f := range args {
			chk, err := cpFile(f, work)
			if err != nil {
				log.Fatal(err)
			}
			s.AddO(f, chk)
		}
		if ofile := viper.GetString("output"); ofile == "" {
			for _, p := range s.Ofiles {
				fmt.Printf("%s %s\n", p.Key, p.Value)
			}
		} else {
			if err := s.AddToFile(ofile); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	retainCmd.Flags().StringP("output", "o", "", "put output checksums into this file")
	retainCmd.Flags().BoolP("clear", "c", false, "clear pastiche storage")
	RootCmd.AddCommand(retainCmd)
}

func cpFile(path string, dest string) (o string, e error) {
	// get ready to read the input file
	h := sha3.New512()
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// create the output file
	tmpfile, err := ioutil.TempFile(dest, "xx")
	if err != nil {
		return "", err
	}
	// now copy it
	buf := make([]byte, 4096) // the number doesn't matter much
	for {
		n, err := file.Read(buf)
		if err != nil {
			break
		}
		h.Write(buf[0:n])
		tmpfile.Write(buf[0:n])
	}
	// make sure it went well
	if (err != nil) && (err != io.EOF) {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}
	// finally done!
	var rh common.RawHash
	h.Sum(rh[:0])
	hash := common.GetHash(rh).String()
	//		fmt.Printf("%s: %s %d\n", path, o.Hash, o.Hash)
	destf := filepath.Join(dest, hash)
	err = os.Rename(tmpfile.Name(), destf)
	return hash, err
}
