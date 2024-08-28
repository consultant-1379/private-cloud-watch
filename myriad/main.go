// Copyright 2016 Ericsson AB All Rights Reserved.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	"github.com/erixzone/myriad/pkg/myriad"
	"github.com/erixzone/myriad/pkg/myriadca"
	"github.com/erixzone/myriad/pkg/myriadfile"
	"github.com/erixzone/myriad/pkg/util/log"
	"github.com/erixzone/myriad/pkg/util/shutdown"

	// Allow supported drivers to register themselves:
	_ "github.com/erixzone/myriad/pkg/myriad/aws"    // 'aws' driver
	_ "github.com/erixzone/myriad/pkg/myriad/docker" // 'docker' driver
)

// myriadVersion is the actual myriad release version.
// For major releases this should be incremented.
const myriadVersion = "v0.2"

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var cfgFile string
var showVersion bool

func init() {
	log.SetLevel(logrus.WarnLevel)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $(PWD)/.myriad.yaml)")
	rootCmd.Flags().StringP("timeout", "t", "0s", "quit execution if it has not complete after this period of time")
	rootCmd.Flags().StringP("out", "o", "", "output container stdio to this path (must exist!)")
	rootCmd.Flags().BoolP("debug", "d", false, "enable debug logging")
	rootCmd.Flags().BoolP("wait", "w", false, "continue running all jobs until an interrupt is sent to myriad.")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Print version info and exit.")
	rootCmd.Flags().BoolP("verbose", "v", false, "print informational logging")

	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("myriad")
	viper.AutomaticEnv()

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	viper.SetConfigName(".myriad")
	viper.AddConfigPath(".")

	viper.BindPFlags(rootCmd.Flags())

	viper.SetDefault("driver", "docker")
	viper.BindEnv("driver")

	viper.SetDefault("make_certs", true)
	viper.SetDefault("cert_OU", "crux")
	viper.SetDefault("certdir", "/crux/crt")
	viper.SetDefault("admincert", "./admincert.tar")
	viper.SetDefault("ca.url", "")
	viper.SetDefault("ca.userid", "")
	viper.SetDefault("ca.passwd", "")
	viper.SetDefault("prometheus.run", false)
	viper.SetDefault("prometheus.port", 0)
	viper.SetDefault("prometheus.cmd", "prometheus")
	viper.SetDefault("prometheus.dir", "")
	viper.SetDefault("prometheus.job", "myriad")
	viper.SetDefault("prometheus.flag.config.file", "prometheus.yaml")
	viper.SetDefault("prometheus.flag.web.listen-address", "0.0.0.0:9090")
	viper.SetDefault("prometheus.log", "prometheus.log")
	viper.SetDefault("prometheus.yaml_epilog", "")
	viper.SetDefault("prometheus.pid", "prometheus.pid")
	viper.SetDefault("prometheus.docker.run", false)
	viper.SetDefault("prometheus.docker.image", "erixzone/prom-run")
	viper.SetDefault("prometheus.cid", "prometheus.cid")

	err := viper.ReadInConfig()

	// ReadInConfig will return an error if no config is provided.
	// We only want to raise an error if a file was provided.
	if viper.ConfigFileUsed() != "" && err != nil {
		fmt.Fprintf(os.Stderr, "Error reading myriad configuration: %s\n", err)
		os.Exit(2)
	}

	log.Infof("Using config file: %s", viper.ConfigFileUsed())
}

var rootCmd = &cobra.Command{
	Use:       "myriad",
	Short:     "Execute a set of networked processes for integration testing",
	Long:      ``,
	Example:   "myriad -d -t 20s myriad-file.txt",
	ValidArgs: []string{"myriad-file"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetBool("debug") {
			log.SetLevel(logrus.DebugLevel)
			log.SetVerbosity(2)
		} else if viper.GetBool("verbose") {
			log.SetLevel(logrus.InfoLevel)
			log.SetVerbosity(0)
		}

		if showVersion {
			printVersion()
			os.Exit(0)
		}

		if len(args) != 1 {
			return cmd.Usage()
		}
		cmd.SilenceUsage = true

		var ca *myriadca.CertificateAuthority
		if viper.GetBool("make_certs") {
			var err error
			if viper.GetString("ca.url") == "" {
				ca, err = myriadca.MakeCertificateAuthority()
			} else {
				ca, err = myriadca.FetchCertificateAuthority()
			}
			if err != nil {
				log.WithError(err).Error("Make/Fetch CertificateAuthority failed")
				return err
			}
		}
		admincert := viper.GetString("admincert")
		if admincert != "" {
			tarBuf, err := ca.LeafTarStream("admin")
			if err != nil {
				log.WithError(err).Error("Make admin cert failed")
				return err
			}
			err = ioutil.WriteFile(admincert, tarBuf.Bytes(), 0600)
			if err != nil {
				log.WithError(err).Error("Write admin cert file failed")
				return err
			}
		}

		// setup context for myriad: We start with one that can be
		// cancelled (this is required since driver.Run() may return
		// and then we need to cancel any in-flight goroutines.
		// We also initialize the top-level wait-layer on this context.
		ctx := context.Background()
		ctx = shutdown.WithWaitLayer(ctx)
		ctx, cancel := context.WithCancel(ctx)

		// If a timeout is defined that is also initialized
		// on the context. This overrides cancel which is perfectly ok.
		// Note that if --timeout and --wait are both set wait takes precedent.
		timeout := viper.GetString("timeout")
		wait := viper.GetBool("wait")
		if timeout != "" && !wait {
			dt, err := time.ParseDuration(timeout)
			if err != nil {
				return err
			}
			if dt != 0 {
				ctx, cancel = context.WithTimeout(ctx, dt)
			}
		} else if timeout != "" && wait {
			msg := "Note: Setting timeout (%s) with --wait is useless. Wait will cause jobs to ignore the timeout"
			log.Infof(msg, timeout)
		}

		d, err := myriad.GetDriver(viper.GetString("driver"))
		if err != nil {
			return err
		}

		file, err := os.Open(args[0])
		if err != nil {
			log.WithError(err).Warnf("Failed to open %s", args[0])
			return err
		}
		defer file.Close()

		jobs, err := myriadfile.Parse(file)
		if err != nil {
			log.WithError(err).Warnf("Failed to parse myriadfile '%s': %s", file.Name(), err)
			return err
		}

		if len(jobs) > 0 {
			jobs[len(jobs)-1].WaitOn = true
		}

		// handleSignals will cause d.Run() to return an error if CTRL+C is
		// pressed or if the myriad run hits its timeout. If d.Run() completes
		// cancel() is called here. In both cases we then block on <-done from
		// handleSignals. This is neccesary as there may be goroutines started
		// by d.Run() that still need to complete.
		go handleSignals(ctx, cancel, os.Interrupt, os.Kill)
		err = d.Run(ctx, jobs, ca)
		cancel()
		shutdown.Wait(ctx)
		return err
	},
}

func handleSignals(ctx context.Context, cancel context.CancelFunc, signals ...os.Signal) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, signals...)

	select {
	case <-ctx.Done():
	case s := <-signalChan:
		fmt.Printf("Got %v, shutting down...\n", s)
		cancel()
	}
}

type myriadfileConfig struct {
	FormatsSupported []string
}

type versionConfig struct {
	Version        string
	BuildDatetime  string
	CommitDatetime string
	CommitID       string
	ShortCommitID  string
	BranchName     string
	GitTreeIsClean bool
	IsOfficial     bool
	Myriadfile     myriadfileConfig
}

func printVersion() {
	gitTreeIsClean := (myriadGitTreeIsClean == "true")
	c := &versionConfig{
		Version:        gitTag(),
		BuildDatetime:  myriadBuildDatetime,
		CommitDatetime: myriadCommitDatetime,
		CommitID:       myriadCommitID,
		ShortCommitID:  myriadShortCommitID,
		BranchName:     myriadBranchName,
		GitTreeIsClean: gitTreeIsClean,
		IsOfficial:     isOfficial(),
		Myriadfile:     myriadfileConfig{FormatsSupported: myriadfile.Versions()},
	}

	b, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error gathering version info: %s\n", err)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stdout, string(b))
}
