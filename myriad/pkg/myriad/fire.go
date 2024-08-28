package myriad

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/erixzone/myriad/pkg/util/log"

	"github.com/spf13/viper"
)

func WritePrometheusConfig(cfg string) error {
	var err error
	dir := viper.GetString("prometheus.dir")
	if dir != "" {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.WithError(err).Error("Create prometheus dir failed")
			return err
		}
	}
	cfgname := path.Join(dir, viper.GetString("prometheus.flag.config.file"))
	err = ioutil.WriteFile(cfgname, []byte(cfg), 0644)
	if err != nil {
		log.WithError(err).Error("Write prometheus config file failed")
		return err
	}
	log.Infof("Wrote prometheus config file '%s'", cfgname)
	return nil
}

func AppendPrometheusFlags(flags []string) []string {
	const pfx = "prometheus.flag."
	for key, ivalue := range viper.AllSettings() {
		if strings.HasPrefix(key, pfx) {
			if value, ok := ivalue.(string); ok {
				flags = append(flags, "--"+key[len(pfx):], value)
			} else {
				log.Warnf("%s: expected string value, got %T\n", key, ivalue)
			}
		}
	}
	return flags
}

func RunPrometheusCmd() {
	promexe, err := exec.LookPath(viper.GetString("prometheus.cmd"))
	if err != nil {
		log.WithError(err).Error("No prometheus executable")
		return
	}
	// see if we can prod a running prometheus...
	dir := viper.GetString("prometheus.dir")
	pidname := path.Join(dir, viper.GetString("prometheus.pid"))
	pidbytes, err := ioutil.ReadFile(pidname)
	pidstr := ""
	var pid int
	if err == nil {
		pidstr = strings.TrimSpace(string(pidbytes))
		pid, err = strconv.Atoi(pidstr)
	}
	if err == nil {
		exepath, _ := os.Readlink(path.Join("/proc", pidstr, "exe"))
		if exepath == promexe {
			err = syscall.Kill(pid, syscall.SIGHUP)
			if err == nil {
				log.Infof("Sent SIGHUP to prometheus process %d ", pid)
				return
			}
		}
	}
	// get a handle on the prometheus log file
	logname := path.Join(dir, viper.GetString("prometheus.log"))
	lf, err := os.OpenFile(logname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.WithError(err).Error("Open/Create prometheus log file failed")
		return
	}
	log.Infof("Prometheus log file is '%s'", logname)
	defer lf.Close() // don't care about error

	// start prometheus...
	promcmd := AppendPrometheusFlags([]string{"prometheus"})
	pid, err = daemonProcess(promexe, promcmd, lf, dir)
	if err != nil {
		log.WithError(err).Error("Run prometheus failed")
		return
	}
	log.Infof("Running prometheus; pid = %d", pid)
	pidstr = fmt.Sprintf("%d\n", pid)
	err = ioutil.WriteFile(pidname, []byte(pidstr), 0600)
	if err != nil {
		log.WithError(err).Error("Create prometheus pid file failed")
		return
	}
	log.Infof("Created prometheus pid file '%s'", pidname)
}

func daemonProcess(exepath string, argv []string, lf *os.File, dir string) (int, error) {
	attr := new(os.ProcAttr)
	attr.Dir = dir
	attr.Files = append(attr.Files, nil, lf, lf)
	attr.Sys = new(syscall.SysProcAttr)
	attr.Sys.Setsid = true

	proc, err := os.StartProcess(exepath, argv, attr)
	if err != nil {
		return -1, err
	}
	pid := proc.Pid
	proc.Release()
	return pid, nil
}
