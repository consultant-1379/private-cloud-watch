Myriad
======

Myriad takes a configuration file (*myriadfile*) consisting of a set of *jobs* and
executes those jobs within a *job enviornment*. It can record standard output and error
from the job and copy files in / out of the environment. Myriad cleans up resources and
returns an exit code related to the exit code of some jobs.

### Requirements

docker  Linux  - Tested on Docker version 17.06.0-ce, amd64
	MacOSX - Tested on Docker version 17.03.1-ce

A docker image (e.g. "ubuntu") on your system for myriad to run.  
If you haven't already, grab like so::


```
$ docker run -it ubuntu
 Unable to find image 'ubuntu:latest' locally
 latest: Pulling from library/ubuntu
 e0a742c2abfd: Pull complete 
 486cb8339a27: Pull complete 
 dc6f0d824617: Pull complete 
 4f7a5649a30e: Pull complete 
 672363445ad2: Pull complete 
 Digest: sha256:84c334414e2bfdcae99509a6add166bbb4fa4041dc3fa6af08046a66fed3005f
 Status: Downloaded newer image for ubuntu:latest
 root@b0a60d005ef8:/# ls
 bin  boot  dev  etc  home  lib  lib64  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var
 root@b0a60d005ef8:/# exit
 exit
$
```


A myriad configuration file in your current working directory.

```
$ cat .myriad.yaml 
---
timeout: 30s
out: "./output"
docker.image: ubuntu
driver: docker
```


```
Usage:
  myriad [flags]

Examples:
myriad -d -t 20s myriad-file.mf

Flags:
      --config string    config file (default is $(PWD)/.myriad.yaml)
  -d, --debug            enable debug logging
  -h, --help             help for myriad
  -o, --out string       output container stdio to this path (must exist!)
  -t, --timeout string   quit execution if it has not complete after this period of time (default "0s")
  -v, --verbose          print informational logging
      --version          Print version info and exit.
  -w, --wait             continue running all jobs until an interrupt is sent to myriad.
```

By default, the output (STDERR and STDOUT) of each process are discarded when
out: is not specified in the .myriad.yaml file and when -o is not specified on the command line.
Use one or the other out, not both. If you specify an output directory myriad will write
files there `${process-name}.stdio` for each process. 

The `--timeout` flag sets a limit on the overall time that myriad will run before
exiting. For example `-t 1h` will set it to one hour. By default there is no limit.


The `--debug` flag will increase the verbosity of logging.

## myriadfile

Myriad takes a single myriadfile (extension .mf) as an argument. This is a text
file written in [HCL format](https://github.com/hashicorp/hcl). This defines the
set of jobs that need to be run and how each job-environment should be configured.
This must start with a version string (current version is `v0.2`) followed by any
number of job definitions:



```
version = "v0.2"

job "hello" {
   command = <<EOF
sh -c 'echo "Hello myriad!" > /test.txt; echo "Wassup myriad on stdout!"'
EOF
   output "/test.txt" {
      dst = "hello_output"
   }
}

job "goodbye" {
   command = <<END
sh -c 'echo "Goodbye myriad!" > /test2.txt; echo "Adios myriad from stdout!"'
END
   output "/test2.txt" {
      dst = "goodbye_output"
   }
   wait = true
}

```

Locally this `dst` path is *relative* to our `--out` path.

This myriadfile produces the following output when `--out` path is "./output" as specified
in the above `myriad.yaml` example:

```
:output$ find .
./goodbye-stdio             Adios myriad from stdout!
./goodbye_output            directory created
./goodbye_output/test2.txt  file containing: Goodbye myriad!
./hello-stdio               Wassup myriad on stdout!
./hello_output              directory created
./hello_output/test.txt     file containing: Hello myriad!
```


The command setting is required and is a shell command to run with and including 
`sh -c` 

Additional supported parameters in the job definition:

    wait   : boolean    Wait on this job in addition to the last job defined.
    input  : object     Add a file or directory to the job environment.
    output : object     Copy a file or directory from the environment after a job is done.

### Inputs and Outputs

The myriad file supports adding files to the job environment before
the command is run.  Multiple files or directories may be added.
The syntax is:


```
version = "v0.2"
job "howdy" {
   command = <<EOF
sh -c 'echo "Howdy $(cat /name.txt)"'
EOF
   input "/name.txt" {
        src = "/Users/cwvhogue/my-name.txt"
   }
}

```

So '/name.txt' must be an absolute path in the job environment. And
the source, '/Users/cwvhogue//my-name.txt' is either an absolute path
or a path relative to the current working directory when myriad is
invoked. 


*Note that simple commands not requiring shell syntax interpretation can be run without `sh -c`:*

```
version = "v0.2"

job "uname" {
    command = <<EOF
uname -a
EOF
}
```

Produces a file at `./output/uname-stdio` containing the line
```
Linux uname 4.9.27-moby #1 SMP Thu May 11 04:01:18 UTC 2017 x86_64 x86_64 x86_64 GNU/Linux
```


*Note: if the output path is not set no data is copied back from the job environment.*
*Also: myriad requires you to `mkdir` the command line or `.myriad.yaml` specified output directory,
but it will create the subdirectory specified in `dst` in the `.mf` file*

Output can also be a directory, such that all the files therein will be copied.

```
version = "v0.2"

job "directory_out" {
    command = <<EOF
sh -c "mkdir -p junk; uname -a > junk/test.txt; cat junk/test.txt; ls -alt > junk/ls.txt"
EOF
    output "/junk" {
        dst = "oot"
    }
}
```

This example gives us:

```
./directory_out-stdio    
./oot
./oot/junk
./oot/junk/ls.txt
./oot/junk/test.txt

with contents:
$ cat directory_out-stdio 
Linux directory_out 4.9.27-moby #1 SMP Thu May 11 04:01:18 UTC 2017 x86_64 x86_64 x86_64 GNU/Linux
$ cat oot/junk/ls.txt
total 76
drwxr-xr-x   1 root root 4096 Jul 22 00:12 .
drwxr-xr-x   1 root root 4096 Jul 22 00:12 ..
drwxr-xr-x   2 root root 4096 Jul 22 00:12 junk
drwxr-xr-x   5 root root  340 Jul 22 00:12 dev
dr-xr-xr-x 134 root root    0 Jul 22 00:12 proc
-rwxr-xr-x   1 root root    0 Jul 22 00:12 .dockerenv
drwxr-xr-x   1 root root 4096 Jul 22 00:12 etc
dr-xr-xr-x  13 root root    0 Jul 21 22:56 sys
drwxr-xr-x   1 root root 4096 Jul 20 17:15 run
drwxr-xr-x   1 root root 4096 Jul 20 17:15 var
drwxr-xr-x   1 root root 4096 Jul 20 17:15 sbin
drwxr-xr-x   1 root root 4096 Jul 20 17:15 usr
drwxrwxrwt   2 root root 4096 Jul 10 18:57 tmp
drwxr-xr-x   2 root root 4096 Jul 10 18:57 bin
drwx------   2 root root 4096 Jul 10 18:56 root
drwxr-xr-x   2 root root 4096 Jul 10 18:56 lib64
drwxr-xr-x   2 root root 4096 Jul 10 18:56 media
drwxr-xr-x   2 root root 4096 Jul 10 18:56 mnt
drwxr-xr-x   2 root root 4096 Jul 10 18:56 opt
drwxr-xr-x   2 root root 4096 Jul 10 18:56 srv
drwxr-xr-x   2 root root 4096 Apr 12  2016 boot
drwxr-xr-x   2 root root 4096 Apr 12  2016 home
drwxr-xr-x   8 root root 4096 Sep 13  2015 lib
$ cat oot/junk/test.txt
Linux directory_out 4.9.27-moby #1 SMP Thu May 11 04:01:18 UTC 2017 x86_64 x86_64 x86_64 GNU/Linux

```

## Configuring myriad

The command line takes a `--config` path to a config file or reads
a `.myriad` file in the current working directory. Note that this
file must have an appropriate extension: `.json` or `.yaml`. Here is the
contents of an example config file:

```
{
    "timeout": "10s",
    "out": "./output",
    "docker.image" : "erixzone/myriad-test",
    "driver": "docker"

}
```

Note that myriad will prioritize settings in this file over settings in the
command line. So this is an effective way of overriding behavior.

## Drivers

Myriad is designed so that different drivers can be used to perform the tests.
Currently we only support the `docker` driver.

### Docker

The Docker driver will start a separate container for each process such that all
containers are on a shared network. The image used is `erixzone/myriad-test` however
this can be overridden via the config file.

```
"docker.image" : Image to use for each process. Default: 'erixzone/myriad-test'
"docker.network.name"   Name of network to use. Default: 'bridge'
"docker.machine.name"   Name of the docker-machine to use. Default: 'default'
"docker.no_machine"     Do not use docker-machine even if it is present.
"docker.force_use_machine" Use docker-machine always.
```

### AWS

The AWS driver will start an Amazon EC2 instance for each job such that all instances
are on a shared network. In order for this to work you must specify the following information
in your configuration file:

- Your region.
- A valid AMI image for your region.
- An instance type.
- An SSH username, keyfile and it's registered name in AWS.
- A security group with at least SSH (port 22) access.

Note that since instance provisions can take a while you may want to set `timeout` to a higher number.
I find that the 't2.micro' instances start up in approximately 2 minutes so I set `"timeout": "5m"`.
```
{
    "driver": "aws",
    "aws.region": "us-east-1",
    "aws.image.ami": "ami-fa82739a",
    "aws.instance.type": "t2.micro",
    "aws.ssh.user": "ubuntu",
    "aws.ssh.keyFile": "~/.ssh/id_rsa",
    "aws.ssh.keyName": "ec2_key_name",
    "aws.securityGroup.ID": "sg-abcdefgh"
}
```
