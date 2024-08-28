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
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	//	"github.com/erixzone/crux/pkg/begat/common"
)

var begatCmd = &cobra.Command{
	Use:   "begat",
	Short: "generate begat test programs from begat test-specs",
	Long:  `nada`,
	Run: func(cmd *cobra.Command, args []string) {
		unit := viper.GetBool("unit")
		unit = true // figure this out; shouldn't need this. TBD
		first := true
		var any bool
		var err error
		for _, f := range args {
			rd, err := os.Open(f)
			if err != nil {
				log.Fatal(err)
			}
			tests, err := parseSpec(rd)
			if err != nil {
				log.Fatal(err)
			}
			for _, t := range tests {
				if first {
					generate(nil, nil, true, os.Stdout)
					first = false
				}
				err = generate(t, tests, unit, os.Stdout)
				if err != nil {
					log.Fatal(err)
				}
				any = true
			}
		}
		if any {
			err = generate(nil, nil, false, os.Stdout)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	//begatCmd.Flags().StringP("output", "o", "", "put output into this file")
	begatCmd.Flags().BoolP("unit", "u", true, "generate a unit test")
	RootCmd.AddCommand(begatCmd)
}
