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
	"net"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/flock"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch the results of flocking",
	Long:  `nada`,
	Run: func(cmd *cobra.Command, args []string) {
		clog.Log = crux.GetLoggerW(os.Stdout)
		clog.Log.SetDebug()
		vip := parseCmd(cmd)
		crux.PromInit(vip.GetInt("fire"), vip.GetBool("wait"))
		beacon := vip.GetString("beacon")
		beaconip, port1x, err1 := net.SplitHostPort(beacon)
		crux.FatalIfErr(nil, crux.ErrE(err1))
		port, _ := strconv.Atoi(port1x)
		key, err := flock.String2Key(vip.GetString("key"))
		crux.FatalIfErr(nil, err)
		fmt.Printf("key = %s  %s\n", vip.GetString("key"), key.String())
		un, err := flock.NewUDPX(port, beaconip, key, true)
		crux.FatalIfErr(nil, err)

		// wait for something to happen
		overall := time.Tick(1000 * time.Second)
		n := vip.GetInt("n")
		fmt.Printf("waiting for n=%d\n", n)
		// start off status analyser
		quit := make(chan bool)
		monq := make(chan crux.MonInfo)
		statq := make(chan flock.Status)
		go flock.StatusAnalyser(quit, monq, statq, 3, 15*time.Second)

	loop:
		for {
			select {
			case <-overall:
				break loop
			case m := <-un.Inbound:
				mi := un.Recv(m)
				//fmt.Printf("------recv %+v\n", mi)
				if mi == nil {
					continue
				}
				monq <- *mi
			case fs := <-statq:
				if fs.N <= 0 {
					fmt.Printf("flocks: %+v  --- breaking loop\n", fs)
					break loop
				}
				fmt.Printf("flocks: %+v\n", fs)
			}
		}
		// done
		quit <- true
		fmt.Printf("done!!\n")
	},
}

func init() {
	watchCmd.Flags().Int("n", 0, "exit with this count and stable")
	watchCmd.Flags().String("key", "", "specify secondary key")
	watchCmd.Flags().String("beacon", "127.0.0.1:28351", "specify node ip/port")
	watchCmd.Flags().Int("fire", 0, "prometheus port (0 means none)")
	watchCmd.Flags().Bool("wait", false, "wait for quit message on prometheus port")
	addCmd(watchCmd)
}
