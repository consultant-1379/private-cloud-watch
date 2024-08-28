# CM Sync KPI

A pair of monitoring scripts that are used to raise a customized alarm in ENM when CM sync KPI is breached.
The scripts untilise the ENM scripting module that allows the user to logon to an ENM deployment, execute valid ENM CLI commands and receive and process the response.

Commands used:

``` sh
cmedit get * CmFunction.syncStatus==SYNCHRONIZED
```

``` sh
cmedit get * CmFunction.syncStatus==UNSYNCHRONIZED
```

## cm_sync_threshold.py

This script calculates the precentage of Unsynchronized CM nodes and raises and alarm using the `fm-alarm-lib` if the KPI threshold is breached.

Scripting Support via Client workstation using url, username, and password by calling function open(url, username, password):

``` python
   import enmscripting
   session = enmscripting.open('<ENM Launcher URL>','<user name>','<password>')
```

### Options

* -t, --threshold [arg]
    + KPI Unsynchronized Threshold as a percentage.

* -l, --launcher [arg]
    + Tenancy httpd_fqdn.

* -u, --user [arg]
    + Tenancy GUI Admin username.

* -p, --passwd [arg]
    + Tenancy GUI Admin password.


## cm_sync_threshold_percentage.py

This script simply returns the percentage of unsynchronized CM nodes.
This script is currently being executed on the ENM Scripting VM with the help of Ansible tower. The ansible playbook is used to raise the alarm with the percentage output from the script.

The ansible playbook is included in the `tasks` directory.

Scripting Support via General Scripting VM:

``` python
   import enmscripting
   session = enmscripting.open()
```

## Notes

Documentation for the ENM CLI can be found on ENM, under the help section.
For example: <ENM Launcher URL>/#help/app/cliapp/topic/scriptingsupport/programmersguide

The ENM Scripting library is availible on the ENM Scripting VM.