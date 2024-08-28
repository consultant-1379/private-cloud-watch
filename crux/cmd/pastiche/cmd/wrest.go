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
	"io"
	"log"
	"os"
	"path/filepath"

	//	version "github.com/erixzone/crux"
	"github.com/spf13/cobra"
)

var wrestCmd = &cobra.Command{
	Use:   "wrest key dest-path",
	Short: "fetch a file by its key",
	Long:  `get the bytes associated with key and put as a file with the given pathname`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatal("Usage: wrest key destpath")
		}
		if err := copyFile(filepath.Join(pasticheRoot, args[0]), args[1]); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(wrestCmd)
}

func copyFile(src, dest string) error {
	rd, err := os.Open(src)
	if err != nil {
		return err
	}
	wr, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, rd)
	return err
}
