package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/casbah"
)

var staticTestFile string
var sysLog bool
var directDial bool
var knownHosts string
var helpFlag bool
var startingPort int
var proxySocket string
var user string

func init() {
	flag.BoolVar(&directDial, "direct", false, "dial nodes directly")
	flag.BoolVar(&sysLog, "syslog", false, "syslogd will add timestamps")
	flag.StringVar(&knownHosts, "knownhosts", "~/.ssh/known_hosts", "known_hosts file")
	flag.BoolVar(&helpFlag, "h", false, "show usage and exit")
	flag.IntVar(&startingPort, "p", 8000, "starting service port")
	flag.StringVar(&proxySocket, "proxy", "eusecgw", "proxy to dial through")
	flag.StringVar(&staticTestFile, "test", "", "get data from static file")
	flag.StringVar(&user, "user", "cloud-user", "username for node login")
}

func usage(rc int) {
	fmt.Fprintf(os.Stderr, "usage: %s [options] tenant-list [tenant-dir]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(rc)
}

type ConsulData []map[string]interface{}

func consulJson(lafNode string) []byte {
	if staticTestFile != "" {
		jbytes, err := ioutil.ReadFile(staticTestFile)
		if err != nil {
			log.Print(err)
			return nil
		}
		return jbytes
	}
	session, err := casbah.NewProxySession(proxySocket, user, lafNode, knownHosts)
	if err != nil {
		if kerr, ok := err.(*casbah.HostKeyError); ok {
			log.Printf("new host key: %s", kerr.Line)
		}
		log.Printf("Session failed: %s", err)
		return nil
	}
	defer session.Close()
	data, err := session.Output("curl http://127.0.0.1:8500/v1/agent/members")
	if err != nil {
		log.Printf("session.Output failed: %s", err)
		return nil
	}
	return data
}

type Tenant struct {
	name  string
	addr  string
	nodes []string
	prometheus.Registry
	gConsul prometheus.Gauge
	gStatus *prometheus.GaugeVec
}

func (tenant *Tenant) initNodes(dir string) error {
	if dir == "" {
		return nil
	}
	fp, err := os.Open(filepath.Join(dir, tenant.name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer fp.Close()
	r := bufio.NewReader(fp)
	for {
		l, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		tenant.nodes = append(tenant.nodes, l[:len(l)-1])
	}
	return nil
}

func (tenant *Tenant) initGather(port int) {
	tenant.Registry = *prometheus.NewRegistry()

	tenant.gConsul = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "consul",
		Help: "Consul member count",
	})
	tenant.gStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "status",
		Help: "Consul member status",
	}, []string{"name"})
	tenant.MustRegister(tenant.gConsul)
	tenant.MustRegister(tenant.gStatus)

	h := promhttp.HandlerFor(tenant, promhttp.HandlerOpts{})
	mux := http.NewServeMux()
	mux.Handle("/metrics", h)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%d", port), mux))
}

func (tenant *Tenant) Gather() ([]*dto.MetricFamily, error) {
	var nodes ConsulData
	log.Printf("Gather requested for %s on %s", tenant.name, tenant.addr)
	jbytes := consulJson(tenant.addr)
	if jbytes != nil {
		err := json.Unmarshal(jbytes, &nodes)
		if err != nil {
			log.Printf("json.Unmarshal failed on %s: %s", tenant.name, err)
		}
	}
	tenant.gConsul.Set(float64(len(nodes)))
	missing := make(map[string]struct{})
	for _, name := range tenant.nodes {
		missing[name] = struct{}{}
	}
	for _, node := range nodes {
		name := node["Name"].(string)
		delete(missing, name)
		status := node["Status"].(float64)
		tenant.gStatus.WithLabelValues(name).Set(status)
	}
	for name := range missing {
		tenant.gStatus.WithLabelValues(name).Set(0.0)
	}
	log.Printf("loaded %d+%d nodes from %s", len(nodes), len(missing), tenant.name)

	return tenant.Registry.Gather()
}

func readConfig(args []string) ([]*Tenant, error) {
	fname := args[0]
	var nodesDir string
	if len(args) > 1 {
		nodesDir = args[1]
	}
	fp, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	r := bufio.NewReader(fp)
	var l string
	var tenants []*Tenant
	splitter := regexp.MustCompile("[ \t]+")
	for {
		l, err = r.ReadString('\n')
		if err != nil {
			break
		}
		l = l[:len(l)-1]
		if len(l) == 0 || l[0] == '#' {
			continue
		}
		s := splitter.Split(l, 2)
		if len(s) != 2 {
			err = fmt.Errorf("can't parse config: %s", l)
			break
		}
		tenant := &Tenant{name: s[0], addr: s[1]}
		err = tenant.initNodes(nodesDir)
		if err != nil {
			break
		}
		tenants = append(tenants, tenant)
	}
	if err != io.EOF {
		return nil, err
	}
	return tenants, nil
}

func main() {
	flag.Parse()
	if helpFlag {
		usage(0)
	}
	if directDial {
		proxySocket = ""
	}
	if flag.NArg() < 1 {
		usage(127)
	}
	if sysLog {
		log.SetFlags(0)
	}
	tenants, err := readConfig(flag.Args())
	if err != nil {
		panic(err)
	}
	port := startingPort
	for _, tenant := range tenants[:len(tenants)-1] {
		go tenant.initGather(port)
		port++
	}
	tenants[len(tenants)-1].initGather(port)
}
