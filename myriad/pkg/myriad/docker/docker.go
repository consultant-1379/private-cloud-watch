// Copyright 2016 Ericsson AB All Rights Reserved.

package docker

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	"github.com/erixzone/myriad/pkg/myriad"
	"github.com/erixzone/myriad/pkg/myriadca"
	"github.com/erixzone/myriad/pkg/myriadfile"
	"github.com/erixzone/myriad/pkg/util/cidr"
	"github.com/erixzone/myriad/pkg/util/log"
	"github.com/erixzone/myriad/pkg/util/shell"
	"github.com/erixzone/myriad/pkg/util/shutdown"
)

//const wantedDockerClientVersion = "1.21"
const wantedDockerClientVersion = "1.25"

type driver struct {
	client         *docker.Client
	waitContainers *myriad.JobMux
}

type factory struct{}

func init() {
	viper.SetDefault("docker.image", "erixzone/myriad-target")
	viper.SetDefault("docker.uid", 999)
	viper.SetDefault("docker.gid", 999)
	viper.SetDefault("docker.network.name", "myriad.erixzone.net")
	viper.SetDefault("docker.network.persist", false)
	viper.SetDefault("docker.network.acceptexisting", false)
	viper.SetDefault("docker.network.subnet", "/24")
	viper.SetDefault("docker.machine.name", "default")

	// Want ternary logic here; it is invalid if both of these options are set to true.
	viper.SetDefault("docker.no_machine", false)
	viper.SetDefault("docker.force_use_machine", false)

	err := myriad.RegisterDriver("docker", &factory{})
	if err != nil {
		log.Error("Docker driver failed to register")
		panic(err)
	}
}

func (f factory) New() (myriad.Driver, error) {
	if useDockerMachine() {
		log.Info("Setting up env for docker-machine.")
		if err := setupDockerMachine(); err != nil {
			log.WithError(err).Error("Failed to setup docker machine.")
			missingDockerMachineInfo()
			return nil, err
		}
	}

	client, err := docker.NewVersionedClientFromEnv(wantedDockerClientVersion)
	if err != nil {
		return nil, err
	}
	driver := &driver{
		client:         client,
		waitContainers: myriad.NewJobMux(),
	}
	return driver, nil
}

// Run runs the myriad jobs
func (d *driver) Run(ctx context.Context, jobs []myriadfile.Job, ca *myriadca.CertificateAuthority) (err error) {

	if err = d.verifyDockerVersion(); err != nil {
		return err
	}

	// Network setup
	log.Info("Setting up network.")
	ctx, network, err := d.setupNetwork(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to setup network.")
		return
	}
	defer network.CleanUp()

	// Check that the image exist
	if err = d.verifyImageExists(); err != nil {
		log.WithError(err).Error("Docker image not found")
		return
	}

	containers := shutdown.ParallelCleaner{}
	defer containers.CleanUp()
	// Run all jobs that are not the final one
	for _, job := range jobs {
		id, container, err := d.runJob(ctx, ca, job)
		if err != nil {
			return err
		}
		containers.Add(container)
		if job.WaitOn {
			log.Debugf("Waiting on %s", job.Name)
			d.waitContainers.AddJob(job.Name, d.containerTask(job, id))
		}
	}
	d.configPrometheus()
	return d.waitContainers.Wait(ctx)
}

// containerTask creates a task for the myriad.JobMux that polls the docker daemon
// until the containers return. Note that while this task will continue to execute
// in the event of a ctx cancellation, the JobMux will record an error for the task.
// Of course there is a bit of a race there as the cleanup task will also try to kill
// the container. So the task may nonetheless complete (with a bad code).
func (d *driver) containerTask(j myriadfile.Job, id string) func() error {
	return func() error {
		for {
			log.Debugf("Polling container: %s", id)
			running, exitCode, err := d.dockerContainerStatus(id)
			if err != nil {
				return err
			}
			if !running {
				if exitCode != 0 {
					return fmt.Errorf("Job %s exited with code %d", j.Name, exitCode)
				}
				return nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// docker-machine related code

// dockerMachineInspect are the values returned from running `docker-machine
// inspect <machine-name>` that we need to replicate the env variables
// required to talk to Docker using docker-machine. Inspect provides
// a lot more info but these are the only bits we need.
type dockerMachineInspect struct {
	Driver struct {
		IPAddress string
	}
	HostOptions struct {
		EngineOptions struct {
			TlsVerify bool
		}
		AuthOptions struct {
			StorePath     string
			ServerKeyPath string
		}
	}
	Name string
}

func setupDockerMachine() (err error) {
	inspect := &dockerMachineInspect{}
	machineName := viper.GetString("docker.machine.name")
	inspectCmd := exec.Command("docker-machine", "inspect", machineName)
	if err = shell.UnmarshalJSON(inspectCmd, inspect); err != nil {
		return err
	}
	if err = setEnvFromInspect(inspect); err != nil {
		return err
	}
	return nil
}

// useDockerMachine determines if we need to use docker-machine
// The rule is check the ternary status of no_machine and
// force_use_machine. If neither of these is set, check if
// docker-machine is in the path.
func useDockerMachine() bool {
	noMachine := viper.GetBool("docker.no_machine")
	forceMachine := viper.GetBool("docker.force_use_machine")
	if noMachine && forceMachine {
		panic("Configuration error: cannot set both docker.no_machine and docker.force_use_machine to true!")
	}
	if noMachine {
		return false
	}
	if forceMachine {
		return true
	}

	client, err := docker.NewVersionedClientFromEnv(wantedDockerClientVersion)
	if err == nil {
		if _, err = client.Info(); err == nil {
			log.Info("Found working docker client; not using docker-machine")
			return false
		}
	}

	if _, err := exec.LookPath("docker-machine"); err != nil {
		log.Info("Did not find docker-machine in PATH")
		return false
	}
	return true
}

// set current process's environment to be able to call docker
func setEnvFromInspect(i *dockerMachineInspect) (err error) {
	if i.HostOptions.EngineOptions.TlsVerify {
		err = os.Setenv("DOCKER_TLS_VERIFY", "1")
	}
	if err == nil && i.Driver.IPAddress != "" {
		host := fmt.Sprintf("tcp://%s:2376", i.Driver.IPAddress)
		err = os.Setenv("DOCKER_HOST", host)
	}
	// TODO: Figure out what value is corret to use for DOCKER_CERT_PATH See:
	// TODO: ^ github issue https://github.com/docker/machine/issues/3226
	if err == nil && i.HostOptions.AuthOptions.ServerKeyPath != "" {
		certPath, _ := path.Split(i.HostOptions.AuthOptions.ServerKeyPath)
		err = os.Setenv("DOCKER_CERT_PATH", certPath)
	}
	if err == nil && i.Name != "" {
		err = os.Setenv("DOCKER_MACHINE_NAME", i.Name)
	}
	return err
}

const missingDockerMachine string = `
Myriad jobs failed to start; no docker-machine running...

It looks like you are trying to run myriad with docker-machine but
don't have a working docker-machine named "%s" set up. Myriad doesn't
do this for you. If you think you should have a working machine,
it may be shutdown. You can check:

    $ docker-machine ls

If no machine exists you can run the following command to start one:

    $ docker-machine create --driver=virtualbox "%s"

If you do not want to use docker-machine, set the following in your config:

    'docker.no_machine' = true

`

// returns some user-actionable text on error
func missingDockerMachineInfo() {
	machineName := viper.GetString("docker.machine.name")
	fmt.Fprintf(os.Stderr, missingDockerMachine, machineName, machineName)
}

// docker related code

func IPAMSubnet(IPAM docker.IPAMOptions) *cidr.IPsubnet {
	for _, ipc := range IPAM.Config {
		if ipc.Subnet != "" {
			result, _ := cidr.Subnet(ipc.Subnet)
			return result
		}
	}
	return nil
}

func (d *driver) createNetwork(name string, subnet *cidr.IPsubnet, needinfo bool) (*docker.Network, error) {
	opts := docker.CreateNetworkOptions{Name: name}
	opts.IPAM = docker.IPAMOptions{Driver: "default"}
	if subnet.IP != nil {
		ipc := docker.IPAMConfig{Subnet: subnet.String()}
		opts.IPAM.Config = []docker.IPAMConfig{ipc}
	}
	net, err := d.client.CreateNetwork(opts)
	if err != nil || !needinfo {
		return net, err
	}
	return d.client.NetworkInfo(name)
}

func (d *driver) setupNetwork(ctx context.Context) (context.Context, shutdown.Cleaner, error) {
	var err error
	name := viper.GetString("docker.network.name")
	subnet, err := cidr.Subnet(viper.GetString("docker.network.subnet"))
	if err != nil {
		return nil, nil, err
	}
	ctxContainers := shutdown.WithWaitLayer(ctx)
	persist := name == "bridge" // don't nuke default network
	persist = persist || viper.GetBool("docker.network.persist")
	fn := func() error {
		shutdown.Wait(ctxContainers)
		if persist {
			log.Infof("Skipping shutdown of network '%s'", name)
			return nil
		}
		log.Infof("Shutting down network '%s'", name)
		if err = d.client.RemoveNetwork(name); err != nil {
			msg := "Failed to shutdown network '%s'"
			log.WithError(err).Warnf(msg, name)
		}
		return err
	}
	net, err := d.client.NetworkInfo(name)
	if err == nil { // named network exists...
		if viper.GetBool("docker.network.acceptexisting") {
			return ctxContainers, shutdown.WithCleaner(ctx, fn), nil
		}
		cur := IPAMSubnet(net.IPAM)
		if subnet.Match(cur) { // and subnet matches specs, all done
			return ctxContainers, shutdown.WithCleaner(ctx, fn), nil
		}
		// use existing address, if we need one and if possible
		if subnet.IP == nil && subnet.Size >= cur.Size {
			subnet.IP = cur.IP
		}
		d.client.RemoveNetwork(name)
	}
	net, err = d.createNetwork(name, subnet, true)
	if err != nil {
		msg := "Failed to create docker network %s: %s"
		return nil, nil, fmt.Errorf(msg, name, err)
	}
	cur := IPAMSubnet(net.IPAM)
	if subnet.Match(cur) { // subnet matches specs, all done
		return ctxContainers, shutdown.WithCleaner(ctx, fn), nil
	}
	if subnet.Size < cur.Size {
		msg := "Cowardly refusing to expand CIDR size of docker network %s to /%d"
		return nil, nil, fmt.Errorf(msg, cur, subnet.Size)
	}
	// use the address, but get rid of the network
	subnet.IP = cur.IP
	d.client.RemoveNetwork(name)
	// this time fer sure!
	net, err = d.createNetwork(name, subnet, false)
	if err != nil {
		msg := "Failed to create docker network %s: %s"
		return nil, nil, fmt.Errorf(msg, name, err)
	}
	return ctxContainers, shutdown.WithCleaner(ctx, fn), nil
}

// runJob starts a single docker container for the myriad jobs that
// are not the final job in the myriad-file. The function returns a shutdown
// or an error if there was an issue starting the container.
func (d *driver) runJob(ctx context.Context, ca *myriadca.CertificateAuthority, job myriadfile.Job) (
	id string, s shutdown.Cleaner, err error) {

	l := log.WithField("job", job.Name)
	config := &docker.Config{
		Hostname: job.Name,
		Cmd:      job.Command,
		Image:    viper.GetString("docker.image"),
	}

	// Bind read-only all files / directories listed in Input.Src
	var binds []string
	for dst, input := range job.Input {
		if _, err = os.Stat(input.Src); os.IsNotExist(err) {
			l.WithError(err).Errorf("Input file %s not found!", dst)
			return
		}
		abs, err := filepath.Abs(input.Src)
		if err == nil {
			input.Src = abs
		}
		bind := fmt.Sprintf("%s:%s:ro", input.Src, dst)
		binds = append(binds, bind)
	}

	hostConfig := &docker.HostConfig{
		Binds:     binds,
		DNSSearch: []string{viper.GetString("docker.network.name")},
		//LogConfig:   docker.LogConfig{Type: "json-file"},
		NetworkMode: viper.GetString("docker.network.name"),
	}
	opts := docker.CreateContainerOptions{
		Name:       job.Name,
		Config:     config,
		HostConfig: hostConfig,
	}
	optsJson, err := toJSON(opts)
	if err != nil {
		l.WithError(err).Warn("Error")
	}
	l.WithField("opts", optsJson).Debug("CreateContainer")
	ctn, err := d.client.CreateContainer(opts)
	var tarBuf *bytes.Buffer
	if err == nil && ca != nil {
		tarBuf, err = ca.LeafTarStream(job.Name)
	}
	if err == nil && tarBuf != nil {
		opts := docker.UploadToContainerOptions{
			InputStream: tarBuf,
			Path:        "/",
			Context:     ctx,
		}
		err = d.client.UploadToContainer(ctn.ID, opts)
	}
	if err == nil {
		l = l.WithField("container", ctn.ID)
		hc, err := toJSON(hostConfig)
		if err != nil {
			l.WithError(err).Warn("Error")
		}
		l.WithField("hostConfig", hc).Debug("StartContainer")
		err = d.client.StartContainer(ctn.ID, hostConfig)
	}
	if err != nil {
		l = l.WithError(err).WithField("command", job.Command)
		l = l.WithField("output", id)
		l.Error("Failed to setup docker container for job")
		file, ferr := getJobLogFile(job)
		if ferr == nil {
			fmt.Fprintf(file, "Error starting container: %s", err)
			file.Close()
		}
		return
	}
	id = ctn.ID

	l.WithField("state", ctn.State.String()).Info("Started container")
	fn := func() error { return d.cleanupContainer(string(id), job) }
	s = shutdown.WithCleaner(ctx, fn)

	if viper.GetString("out") != "" {
		d.captureLogs(job, id)
	}
	return id, s, err

}

// getJobLogFile opens a logFile for writing. You must close it.
func getJobLogFile(job myriadfile.Job) (*os.File, error) {
	filename := fmt.Sprintf("%s-%s", job.Name, "stdio")
	p := path.Join(viper.GetString("out"), filename)
	return os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
}

// captureLogs captures stderr and stdout from the container.
// The Logs() function will exit after the container is stopped.
func (d *driver) captureLogs(job myriadfile.Job, id string) {
	file, err := getJobLogFile(job)
	if err != nil {
		filename := fmt.Sprintf("%s-%s", job.Name, "stdio")
		log.WithError(err).Warnf("Failed to open '%s'", filename)
		return
	}
	opts := docker.LogsOptions{
		Container:    id,
		OutputStream: file,
		ErrorStream:  file,
		Follow:       true,
		Stdout:       true,
		Stderr:       true,
	}
	go func() {
		l := log.WithField("container", id)
		l.Infof("Capturing container logs")
		err = d.client.Logs(opts)
		if err != nil {
			l.WithError(err).Warnf("Error getting container logs")
		}
		err = file.Close()
		if err != nil {
			l.WithError(err).Warnf("Error closing file %s", file.Name())
		}
		l.Info("Done capturing logs for container")

	}()
}

func (d *driver) dockerContainerStatus(id string) (running bool, exitCode int, err error) {
	inspect, err := d.client.InspectContainer(id)
	if err != nil {
		log.Printf("Failed to get container... got %+v", inspect)
		return
	}
	return inspect.State.Running, inspect.State.ExitCode, nil
}

func (d *driver) cleanupContainer(id string, job myriadfile.Job) (err error) {
	running, _, err := d.dockerContainerStatus(id)
	l := log.WithField("container", id).WithField("job", job.Name)
	if err != nil {
		l.WithError(err).Error("Failed to get container status")
		return err
	}
	if running {
		l.Info("Stopping container")
		// TODO: config option for how long to wait before killing container?
		// Docker command defaults to 10s, but we may want to be faster?
		d.client.StopContainer(id, 1)
	}
	outdir := viper.GetString("out")

	if outdir != "" {
		for path, output := range job.Output {
			dstPath := filepath.Join(outdir, output.Dst)
			l.Infof("Downloading %s to %s", path, dstPath)

			preader, pwriter := io.Pipe()
			opts := docker.DownloadFromContainerOptions{
				Path:         path,
				OutputStream: pwriter,
				// FIXME: no way to cancel docker.DownloadFromContainer request
				// So here we set an arbitrary timeout (30 seconds) and hope for the best.
				// See: https://github.com/fsouza/go-dockerclient/issues/516
				// CWVH  updated - go-dockerclient now supports context, but not yet applied here
				InactivityTimeout: time.Second * 30,
			}
			go func() {
				err := extractTar(preader, dstPath)
				if err != nil {
					l.WithError(err).Warn("Failed in extractTar on download stream")
				}
			}()
			if err = d.client.DownloadFromContainer(id, opts); err != nil {
				l.WithError(err).Error("Failed to download")
			}
		}
	} else if len(job.Output) > 0 {
		l.Infof("Ignoring output files since -out is not set.")
	}

	l.Info("Removing container")
	return d.client.RemoveContainer(docker.RemoveContainerOptions{ID: id})
}

type dockerVersion struct {
	Server string `json:"server"`
	Client string `json:"client"`
}

func (d *driver) verifyDockerVersion() error {
	dockerVersionTmpl := `{"server": "{{.Server.Version}}", "client": "{{.Client.Version}}"}`
	versionJSON, err := shell.StdoutLine("docker", "version", "-f", dockerVersionTmpl)
	if err != nil {
		return err
	}
	var v dockerVersion
	if err = json.Unmarshal([]byte(versionJSON), &v); err != nil {
		return err
	}

	// CWVH removed hard-coded version number compatibility check.
	return nil
}

func (d *driver) verifyImageExists() error {
	image := viper.GetString("docker.image")
	_, err := d.client.InspectImage(image)
	return err
}

func extractTar(r io.Reader, mpath string) error {
	tarReader := tar.NewReader(r)

	if _, err := os.Stat(mpath); err != nil {
		log.WithField("path", mpath).Debug("Making output directory...")
		err := os.MkdirAll(mpath, 0755)
		if err != nil {
			return err
		}
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		p := path.Join(mpath, header.Name)
		log.WithField("file", p).Debug("Target file...")

		switch header.Typeflag {
		case tar.TypeDir:
			log.WithField("directory", p).Debug("Making directory...")
			if _, err := os.Stat(p); err != nil {
				err := os.MkdirAll(p, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
			}
		case tar.TypeReg:
			log.WithField("file", p).Debug("Creating file...")
			perm := os.FileMode(header.Mode).Perm()
			f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, perm)
			if err != nil {
				return err
			}
			defer f.Close()

			log.WithField("file", p).Debug("Copying from tar stream reader...")
			if _, err := io.Copy(f, tarReader); err != nil {
				return err
			}

		default:
			msg := "Ignoring non-regular file %s when getting output (type: %d)"
			log.Warnf(msg, p, int(header.Typeflag))
		}
	}
	return nil
}

func toJSON(i interface{}) (string, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(b), err
}

var yamlGlobalPreamble = `# generated by myriad
global:
  scrape_interval: 4s
  scrape_timeout: 2s
  external_labels:
    config_date: "%s"
    config_sec: "%d"

scrape_configs:
- job_name: "%s"
`

var yamlTLSParms = `  scheme: https
  tls_config:
    ca_file: %s
    ca_file_is_tar: true
`

var yamlStaticPreamble = `  static_configs:
`

var yamlTarget = `  - targets: [ "%s:%d" ]
    labels:
      IPv4: "%s"
`

func (d *driver) configPrometheus() error {
	port := viper.GetInt("prometheus.port")
	if port <= 0 {
		return nil
	}
	netname := viper.GetString("docker.network.name")
	net, err := d.client.NetworkInfo(netname)
	if err != nil { // "can't happen"
		return nil
	}
	var yaml strings.Builder
	now := time.Now()
	fmt.Fprintf(&yaml, yamlGlobalPreamble, now.Format(time.UnixDate), now.Unix(),
		viper.GetString("prometheus.job"))
	if ca_file := viper.GetString("admincert"); ca_file != "" {
		fmt.Fprintf(&yaml, yamlTLSParms, ca_file)
	}
	fmt.Fprintf(&yaml, yamlStaticPreamble)
	for _, e := range net.Containers {
		if e.Name == "prometheus" {
			continue
		}
		fmt.Fprintf(&yaml, yamlTarget,
			e.Name, port, strings.Split(e.IPv4Address, "/")[0])
	}
	// now add in yaml epilog
	if epilog := viper.GetString("prometheus.yaml_epilog"); epilog != "" {
		stuff, err := ioutil.ReadFile(epilog)
		if err != nil {
			return err
		}
		fmt.Fprintf(&yaml, "%s", stuff)
	}
	err = myriad.WritePrometheusConfig(yaml.String())
	if err != nil {
		return err
	}
	if viper.GetBool("prometheus.docker.run") {
		d.runPrometheus()
	} else if viper.GetBool("prometheus.run") {
		myriad.RunPrometheusCmd()
	}
	return nil
}

func (d *driver) runPrometheus() {
	var err error
	l := log.WithField("job", "prometheus")
	dir := viper.GetString("prometheus.dir")
	if dir == "" || dir == "." {
		dir, err = os.Getwd()
		if err != nil {
			l.WithError(err).Errorf("os.Getwd failed")
			return
		}
	}
	cidname := path.Join(dir, viper.GetString("prometheus.cid"))
	// see if we can prod a running prometheus...
	cid, err := ioutil.ReadFile(cidname)
	if err == nil {
		opts := docker.KillContainerOptions{
			ID:     string(cid),
			Signal: docker.SIGHUP,
		}
		err = d.client.KillContainer(opts)
	}
	if err == nil {
		log.Infof("Sent SIGHUP to prometheus container %s", string(cid))
		return
	}
	cmd := myriad.AppendPrometheusFlags([]string{viper.GetString("prometheus.log")})
	addr := strings.Split(viper.GetString("prometheus.flag.web.listen-address"), ":")
	port := addr[len(addr)-1]
	dport := docker.Port(port + "/tcp")
	pexps := make(map[docker.Port]struct{})
	pexps[dport] = struct{}{}
	config := &docker.Config{
		Hostname:     "prometheus",
		Cmd:          cmd,
		Image:        viper.GetString("prometheus.docker.image"),
		ExposedPorts: pexps,
	}
	pbind := docker.PortBinding{HostIP: "0.0.0.0", HostPort: port}
	pbinds := make(map[docker.Port][]docker.PortBinding)
	pbinds[dport] = []docker.PortBinding{pbind}
	hostConfig := &docker.HostConfig{
		RestartPolicy: docker.NeverRestart(),
		AutoRemove:    true, // needs client version 1.25 or later
		Binds:         []string{fmt.Sprintf("%s:/prometheus", dir)},
		PortBindings:  pbinds,
		DNSSearch:     []string{viper.GetString("docker.network.name")},
		NetworkMode:   viper.GetString("docker.network.name"),
	}
	opts := docker.CreateContainerOptions{
		Name:       "prometheus",
		Config:     config,
		HostConfig: hostConfig,
	}
	ctn, err := d.client.CreateContainer(opts)
	if err != nil {
		l.WithError(err).Error("Failed to create container for prometheus")
		return
	}
	err = d.client.StartContainer(ctn.ID, hostConfig)
	if err != nil {
		l.WithError(err).Error("Failed to start container for prometheus")
		return
	}
	l.WithField("state", ctn.State.String()).Info("Started prometheus container")
	err = ioutil.WriteFile(cidname, []byte(ctn.ID), 0600)
	if err != nil {
		log.WithError(err).Error("Create prometheus cid file failed")
		return
	}
	log.Infof("Created prometheus cid file '%s'", cidname)
}
