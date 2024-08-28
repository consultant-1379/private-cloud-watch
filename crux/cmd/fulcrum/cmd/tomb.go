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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/spf13/cobra"

	"github.com/erixzone/crux/pkg/clog"
	"github.com/erixzone/crux/pkg/crux"
)

var mydebug bool

var tombCmd = &cobra.Command{
	Use:   "tomb",
	Short: "sync/src metric data",
	Long:  `nada`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := crux.GetLoggerW(os.Stdout)
		logger.SetDebug()
		vip := parseCmd(cmd)
		crux.PromInit(vip.GetInt("fire"), vip.GetBool("wait"))
		mydebug = vip.GetBool("debug")

		listen := vip.GetString("listen")
		if listen == "" {
			fmt.Fprintln(os.Stderr, "Listening port must be specified using the -listen flag.")
			os.Exit(1)
		}
		if listen[0:1] != ":" {
			listen = ":" + listen
		}

		ofilename := vip.GetString("ofile")
		if ofilename == "" {
			fmt.Fprintln(os.Stderr, "Output file name must be specified using the --ofile flag.")
			os.Exit(1)
		}
		ofile, err := os.OpenFile(ofilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		crux.FatalIfErr(logger, crux.ErrE(err))
		ifilename := vip.GetString("ifile")
		if ifilename == "" {
			fmt.Fprintln(os.Stderr, "Input file name must be specified using the --ifile flag.")
			os.Exit(1)
		}
		ifile, err := os.OpenFile(ifilename, os.O_RDONLY, 0644)
		crux.FatalIfErr(logger, crux.ErrE(err))

		fmt.Printf("listening on '%s'; writing to file '%s' and reading from '%s'\n", listen, ofilename, ifilename)
		// now actually start the work
		runHTTP(listen, logger, ofile, ifile)

		time.Sleep(400 * time.Second)
		fmt.Printf("done!!\n")
	},
}

func init() {
	// tombCmd.Flags().Int("port", 23123, "use this port to listen for flocking traffic")
	tombCmd.Flags().String("key", "", "specify secondary key") // not used really
	// tombCmd.Flags().String("name", "", "specify node name")
	// tombCmd.Flags().String("ip", "", "specify node ip")
	tombCmd.Flags().String("beacon", "lodestar.org", "external coordination point") // not used really
	// tombCmd.Flags().String("horde", "nohordename", "horde name") // deliberately set the default horde name to something unusual
	tombCmd.Flags().String("ifile", "", "disk file to use as a reader")
	tombCmd.Flags().String("ofile", "", "disk file to use as a writer")

	tombCmd.Flags().Int("fire", 0, "prometheus port (0 means none)")
	tombCmd.Flags().Bool("wait", false, "wait for quit message on prometheus port")
	// tombCmd.Flags().String("prefix", "", "Prefix for metric names. If omitted, no prefix is added.")
	// tombCmd.Flags().Int("proxy-port", 2878, "Proxy port.")
	tombCmd.Flags().String("listen", "", "Port/address to listen to on the format '[address:]port'. If no address is specified, the adapter listens to all interfaces.")
	tombCmd.Flags().Bool("debug", false, "Print detailed debug messages.")
	addCmd(tombCmd)
}

func runHTTP(listen string, log clog.Logger, ofile, ifile *os.File) {
	http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
		if mydebug {
			log.Log(nil, "/receive starting")
		}
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if mydebug {
			log.Log(nil, "Got request")
		}

		var req prompb.WriteRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		j, err := json.Marshal(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		j = append(j, '\n')
		if _, err = ofile.Write(j); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if mydebug {
			log.Log(nil, "/receive ending")
		}
	})

	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if mydebug {
			log.Log(nil, "/send starting")
		}
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.ReadRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// we ignore the query; return it all!

		// get data from file into 'data'
		// cheap hack; limit ourself to just 10 million bytes
		data := make([]byte, 50000000)
		n, err := ifile.ReadAt(data, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = data[0:n]
		if mydebug {
			log.Log(nil, "/send sent %d bytes", n)
		}

		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Header().Set("Content-Encoding", "snappy")

		compressed = snappy.Encode(nil, data)
		if _, err := w.Write(compressed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if mydebug {
			log.Log(nil, "/send ending")
		}
	})

	log.Fatal(http.ListenAndServe(listen, nil))
}
