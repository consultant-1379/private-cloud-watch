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
	version "github.com/erixzone/crux"
	"github.com/erixzone/crux/pkg/utils/cli"
	"github.com/spf13/cobra"
)

var format string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of fulcrum code",
	Long: `Even though the version refers to fulcrum, it specifies exactly
which release of the crux code was used to generate this binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		cli.PrintFormatData(format, version.Version())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
	versionCmd.Flags().StringVarP(&format, "format", "f", "", "Go text/template string")
}
