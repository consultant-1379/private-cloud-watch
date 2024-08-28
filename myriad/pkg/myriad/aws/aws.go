package aws

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/apcera/libretto/ssh"
	"github.com/apcera/libretto/virtualmachine/aws"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	"github.com/erixzone/myriad/pkg/myriad"
	"github.com/erixzone/myriad/pkg/myriadca"
	"github.com/erixzone/myriad/pkg/myriadfile"
	"github.com/erixzone/myriad/pkg/util/log"
	"github.com/erixzone/myriad/pkg/util/shutdown"
)

type driver struct {
	runID          string
	keyFileData    []byte
	machines       map[string]*aws.VM
	machineCleanup *shutdown.ParallelCleaner
	waitProvision  *myriad.JobMux
	waitJobs       *myriad.JobMux
}
type factory struct{}

const timeFormat = "20060102150405"

func init() {
	viper.SetDefault("aws.image.ami", "")
	viper.SetDefault("aws.instance.nameTemplate", "myriad{{.RunID}}_job_{{.JobName}}")
	viper.SetDefault("aws.instance.type", "t2.micro")
	viper.SetDefault("aws.ssh.user", "")
	viper.SetDefault("aws.ssh.keyFile", "")
	viper.SetDefault("aws.ssh.keyName", "")
	viper.SetDefault("aws.region", "us-west-2")
	viper.SetDefault("aws.securityGroup.ID", "")

	err := myriad.RegisterDriver("aws", &factory{})
	if err != nil {
		log.WithError(err).Error("Failed to register aws driver")
	}
}

func (f factory) New() (myriad.Driver, error) {
	raw, err := ioutil.ReadFile(viper.GetString("aws.ssh.keyFile"))
	if err != nil {
		log.WithError(err).Warnf("Error opening file for 'aws.ssh.keyFile'")
		return nil, err
	}

	return &driver{
		runID:          time.Now().Format(timeFormat),
		keyFileData:    raw,
		machines:       make(map[string]*aws.VM),
		machineCleanup: &shutdown.ParallelCleaner{},
		waitProvision:  myriad.NewJobMux(),
		waitJobs:       myriad.NewJobMux(),
	}, nil
}

func (d *driver) getVMName(job string) (string, error) {
	// nameTemplate is a golang text/template string that takes RunID and JobName strings.
	tmpl, err := template.New("vm_tmpl").Parse(viper.GetString("aws.instance.nameTemplate"))
	if err != nil {
		return "", err
	}

	buff := bytes.NewBuffer(nil)
	data := &struct {
		RunID   string
		JobName string
	}{
		RunID:   d.runID,
		JobName: job,
	}
	err = tmpl.Execute(buff, data)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}

func (d *driver) Run(ctx context.Context, jobs []myriadfile.Job, ca *myriadca.CertificateAuthority) (err error) {
	// TODO: validate aws options at the beginning
	// TODO: make certificates if ca != nil
	defer d.machineCleanup.CleanUp()

	for _, job := range jobs {
		log.Debugf("Provsioning %s", job.Name)
		vm, cleaner, err := d.provision(ctx, job)
		if err != nil {
			log.WithError(err).Warn("Failed to provision VM")
			return err
		}
		d.machineCleanup.Add(cleaner)
		d.machines[job.Name] = vm
	}
	err = d.waitProvision.Wait(ctx)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		vm := d.machines[job.Name]
		err = d.startCommand(ctx, job, vm)
		if err != nil {
			return
		}
	}

	return d.waitJobs.Wait(ctx)
}

func (d *driver) provision(ctx context.Context, job myriadfile.Job) (vm *aws.VM, c shutdown.Cleaner, err error) {
	name, err := d.getVMName(job.Name)
	if err != nil {
		return
	}

	vm = &aws.VM{
		Name:         name,
		AMI:          viper.GetString("aws.image.ami"),
		InstanceType: viper.GetString("aws.instance.type"),
		Region:       viper.GetString("aws.region"),

		DeviceName:                   "/dev/sda1",
		KeepRootVolumeOnDestroy:      false,
		DeleteNonRootVolumeOnDestroy: true,

		KeyPair: viper.GetString("aws.ssh.keyName"),
		SSHCreds: ssh.Credentials{
			SSHUser:       viper.GetString("aws.ssh.user"),
			SSHPrivateKey: string(d.keyFileData),
		},
		SecurityGroup: viper.GetString("aws.securityGroup.ID"),
	}
	task := func() error {
		err := vm.Provision()
		if err != nil {
			return err
		}
		if err = vm.Start(); err != nil {
			return err
		}
		ips, err := vm.GetIPs()
		if err != nil {
			log.WithError(err).Warn("Error getting IPs")
			return err
		}

		client := &ssh.SSHClient{
			Creds:   &vm.SSHCreds,
			IP:      ips[aws.PublicIP],
			Options: ssh.Options{},
			Port:    22,
		}
		log.Debugf("Waiting on SSH: %s %s", job.Name, vm.InstanceID)
		return waitForSSH(ctx, client)
	}
	c = shutdown.WithCleaner(ctx, func() error { return vm.Destroy() })
	return vm, c, d.waitProvision.AddJob(job.Name, task)
}

func (d *driver) startCommand(ctx context.Context, job myriadfile.Job, vm *aws.VM) (err error) {
	stdout := ioutil.Discard
	stderr := ioutil.Discard

	if job.Logfile != "" {
		file, err := os.OpenFile(job.Logfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.WithError(err).Warnf("Failed to open '%s'", job.Logfile)
		} else {
			stdout = file
			stderr = file
		}
	}

	task := func() error {
		client, err := vm.GetSSH(ssh.Options{})
		if err != nil {
			return err
		}
		if err = client.Connect(); err != nil {
			return err
		}
		defer client.Disconnect()

		cmd := strings.Join(job.Command, " ")
		log.Debugf("Starting command (%s, %s) %s", job.Name, vm.InstanceID, cmd)

		err = client.Run(cmd, stdout, stderr)
		l := log.WithField("cmd", cmd).WithField("id", vm.InstanceID)
		if err != nil {
			l.WithError(err)
		}
		l.Debugf("Return from command")
		return err
	}

	if job.WaitOn {
		log.Debugf("Waiting on %s", job.Name)
		return d.waitJobs.AddJob(job.Name, task)
	}
	go task()
	return

}

// waitForSSH waits until ssh comes up. We could use the libretto provided
// vm.GetSSH but that uses a fixed timeout and we need to use a context.Context.
func waitForSSH(ctx context.Context, c *ssh.SSHClient) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err := c.Connect()
		if err == nil {
			return nil
		}
		log.Debug("SSH Connect failed, retrying...")
		time.Sleep(5 * time.Second)
	}
}
