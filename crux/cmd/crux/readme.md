# `crux`  
Template/Demo command executable.
 
### `crux` Key Components:
- Template/Demo command executable
- Busybox style executable housing app subcommands
- Version information tied to git repo state information
- Viper/Cobra/pflag
  - command line parameter handling
  - help system
  - yaml config files


### `crux` Command Line Help
````
	$ crux -h
	Nothing more to say.
	
	Usage:
	  crux [command]
	
	Available Commands:
	  help        Help about any command
	  version     Print the version of crux code
	
	Flags:
	      --H string        horde name
	      --X string        configuration family for crux services
	      --config string   config file (default is $HOME/.cryaml)
	  -h, --help            help for crux
	      --ip string       IP address (either name or octets) to reach this node
	      --join string     list of nodes for finding Consul
	      --node string     node name

	Use "crux [command] --help" for more information about a command.

	$ crux version
	{
	  "Version": "",
	  "BuildDatetime": "UNKNOWN",
	  "CommitDatetime": "$Format:%cI$",
	  "CommitID": "$Format:%H$",
	  "ShortCommitID": "$Format:%h$",
	  "BranchName": "UNKNOWN",
	  "GitTreeIsClean": false,
	  "IsOfficial": false
	}
````
