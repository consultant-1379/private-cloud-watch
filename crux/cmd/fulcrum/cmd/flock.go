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
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/ruck"
)

var flockCmd = &cobra.Command{
	Use:   "flock",
	Short: "Run the flocking command with",
	Long:  `nada`,
	Run: func(cmd *cobra.Command, args []string) {
		clog.Log = crux.GetLoggerW(os.Stdout)
		clog.Log.SetDebug()
		vip := parseCmd(cmd)

		// prometheus monitoring
		crux.PromInit(vip.GetInt("fire"), vip.GetBool("wait"))
		// now actually start the work
		ruck.Bootstrap(vip.GetInt("port"), vip.GetString("key"), vip.GetString("name"), vip.GetString("ip"), vip.GetString("horde"),
			vip.GetString("beacon"), vip.GetString("networks"), vip.GetString("certdir"), vip.GetBool("visitor"))

		time.Sleep(40 * time.Second)
		fmt.Printf("done!!\n")
	},
}

func init() {
	flockCmd.Flags().Int("port", 23123, "use this port to listen for flocking traffic")
	flockCmd.Flags().String("key", "", "specify secondary key")
	flockCmd.Flags().String("name", "", "specify node name")
	flockCmd.Flags().String("ip", "", "specify node ip")
	flockCmd.Flags().String("beacon", "lodestar.org", "external coordination point")
	flockCmd.Flags().String("horde", "nohordename", "horde name") // deliberately set the default horde name to something unusual
	// If no networks are specified, we will probe the local network for the given ip.
	flockCmd.Flags().String("networks", "", "comma-separated list of CIDR networks to probe")
	flockCmd.Flags().String("certdir", "/crux/crt", "certificate directory")
	flockCmd.Flags().Bool("visitor", false, "visitor node")
	flockCmd.Flags().Int("fire", 0, "prometheus port (0 means none)")
	flockCmd.Flags().Bool("wait", false, "wait for quit message on prometheus port")
	addCmd(flockCmd)
}
