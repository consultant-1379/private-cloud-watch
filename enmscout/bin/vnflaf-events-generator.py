#!/usr/bin/python3.6

"""
VNF-LAF events generator by loren jan wilson, 2019-10.

Mission: Take in a list of filenames, and output a list of HA workflow events.

Note: Why not take logs from stdin? Because the logs don't include tenant
names, so we have to infer them from the filename.

Purpose: We need to know what the HA system did so we know how customers were
impacted. We also have a visualization tool for this, but sometimes it helps to
just have a text file of events for further processing or reporting, and you
can't easily extract events from the Prometheus time series database.

Future improvements:
- It's possible that we actually need more than just HA events, and more event
  types could be added in the future.
- Some HA workflows encounter errors, and it would be possible to collect and
  tally them if we knew what they all looked like.

Input is:
- a list of vnflaf log filenames as output by the vnflaf-log-slurper utility.

Output is:
- one line for each HA event, including:
    - workflow start time and completion time (although there's no good way to know when the HA workflow ended because it isn't logged...you have to assume it ended after the HA ID stops showing up in the log lines)
    - list of affected VMs
    - also possibly a field showing number of retries of this HA workflow? or would that be per VM?
- alternately, one line for each recovered VM, including:
    - time an HA workflow was fired for it
    - HA ID of workflow
    - number of retries to get it going again
    - time it was seen to recover

What to do:
- for each file, get deployment name from filename
- open files and load relevant lines into memory, grouped per deployment:
        - 2019-09-24 17:16:07,609 INFO  [com.ericsson.oss.services.wfs.jse.impl.WorkflowInstanceServiceImpl] (http-0.0.0.0:8080-1) Successfully started workflow with id enmdeploymentworkflows.--.1.73.7.--.HighAvailabilityWorkflow__top, variables {WFSContext={}, vms={"staging01-sso-1":"10.10.0.171","staging01-mspmip-1":"10.10.0.122","staging01-dpmediation-1":"10.10.0.192","staging01-mskpirt-1":"10.10.0.220","staging01-httpd-1":"10.10.0.90","staging01-lvsrouter-1":"10.10.0.103","staging01-msap-1":"10.10.0.108","staging01-msnetlog-1":"10.10.0.118","staging01-msapgfm-1":"10.10.0.185","staging01-msfm-1":"10.10.0.116","staging01-mscmce-1":"10.10.0.112","staging01-mscmapg-1":"10.10.0.196","staging01-mscm-1":"10.10.0.110","staging01-mssnmpcm-1":"10.10.0.124","staging01-mssnmpfm-1":"10.10.0.126","staging01-dchistory-1":"10.10.0.66","staging01-comecimpolicy-1":"10.10.0.64","staging01-mscmip-1":"10.10.0.114","staging01-mspm-1":"10.10.0.120","staging01-medrouter-1":"10.10.0.105","staging01-sso-0":"10.10.0.170"}} and business key HA_staging01-sso-1,staging01-mspmip-1,staging01-dpmediation-1,staging01-mskpirt-1,staging01-httpd-1,staging01-lvsrouter-1,staging01-msap-1,staging01-msnetlog-1,staging01-msapgfm-1,staging01-msfm-1,staging01-mscmce-1,staging01-mscmapg-1,st..._1569345367380
        - Successfully started workflow with id enmdeploymentworkflows.--.1.73.7.--.HighAvailabilityWorkflow__top, variables {WFSContext={}, vms={"staging01-httpd-0":"10.10.0.89"}} and business key HA_staging01-httpd-0_1569835510069
        - HA_1569345367380 stack_status of inner stack for VM staging01-sso-1 is DELETE_COMPLETE
        - Marking resource unhealthy for staging01-sso-1, resource
        - Marking inner stack unhealthy for staging01-httpd-0
        - VM ( staging01-sso-1 ) successfully restored (NO HA ID ON THIS LINE)
        - HA_1569835510069 Successfully cleared FM alarm for VM: staging01-httpd-0
        - HA_1569345367380 FM alarm for staging01-msap-1 not raised, bypassing attempt to clear
- sort lines by time
- walk through them:
    - parse VM names and HA id and start time from "Successfully started workflow with id.*HighAvailabilityWorkflow" lines
    - calculate retries for each VM by looking at "DELETE_COMPLETE" lines that occur later, but before successfully restored
    - calculate finished time by looking at "successfully restored" line
        - or instead using "Successfully cleared FM alarm OR bypassing attempt to clear"
    - end up with a line of output, including deployment name, vm name, HA workflow ID, HA workflow start time, VM recovered time, number of retries
- sort all output lines for all deployments by time, then print

"""

import os
import sys
import argparse
import gzip
import re
import json
import enmscout

my_name = 'vnflaf-events-generator'

# Pass a list of logfiles and an output format at the command line.
parser = argparse.ArgumentParser()
parser.add_argument('--csv', action="store_true", help='CSV output format (default is json)')
parser.add_argument('--debug', action="store_true", help='debug logging')
parser.add_argument('logfiles', nargs='+', help='vnflaf input log files')
args = parser.parse_args()

# Set up general logging.
log = enmscout.configure_logging(my_name, debug=args.debug)

# Define the patterns we care about.
ha_workflow_start_pattern = 'Successfully started workflow with id.*HighAvailabilityWorkflow.*vms={(.*?)}.*_(\d+)$'
vm_mark_resource_pattern = 'Marking resource unhealthy for (.*?), resource'
vm_mark_innerstack_pattern = 'Marking inner stack unhealthy for (.*?)$'
vm_restored_pattern = 'VM \( (.*?) \) successfully restored'

ha_workflow_start_re = re.compile(ha_workflow_start_pattern)
vm_mark_resource_re = re.compile(vm_mark_resource_pattern)
vm_mark_innerstack_re = re.compile(vm_mark_innerstack_pattern)
vm_restored_re = re.compile(vm_restored_pattern)

# Lines to keep when filtering the logs.
relevant_re = [ ha_workflow_start_re, vm_mark_resource_re, vm_mark_innerstack_re, vm_restored_re ]

class Deployment:
    """ A Deployment is an ENMaaS deployment, and it has a list of log lines
    and also a log processing routine.
    """
    def __init__(self, name):
        self.name = name
        self.logs = []

    def generate_ha_events(self):
        """ Generate and return a list of HAEvents.
        """
        # Sort log lines by time.
        self.logs.sort()
        # As HAEvents resolve, append them to this list.
        ha_events = []
        # Walk through the logs and create HAEvents.
        open_events = {}
        for line in self.logs:
            # Format the timestamp to something more standard.
            # By default it's '%Y-%m-%d %H:%M:%S,%f'
            # We want it more like '%Y-%m-%dT%H:%M:%S.%f'
            this_time = "T".join(line.split()[0:2]).replace(',', '.')

            # This line is one of the following things:
            # 1. A workflow start for some number of VMs.
            # 2. A heat operation ("attempt") for a VM.
            # 3. A "VM successfully restored" message (VM fault cleared).

            # If it's a workflow start, parse out the VMs and open events.
            match = ha_workflow_start_re.search(line)
            if match:
                # Open a separate event for each VM.
                pairs = match.group(1).split(',')
                vms = [ pair.split(':')[0].strip('"') for pair in pairs ]
                ha_id = match.group(2)
                for vm in vms:
                    if vm not in open_events:
                        open_events[vm] = HAEvent(self.name, vm, ha_id, this_time)
                    else:
                        log.debug(f"Ignoring a duplicate HA workflow for VM {vm} at {this_time}")
                continue

            # If it's a heat operation, increment the attempts counter.
            match = vm_mark_resource_re.search(line) or vm_mark_innerstack_re.search(line)
            if match:
                # Increment attempts for this VM.
                vm = match.group(1)
                if vm in open_events:
                    open_events[vm].increment_attempts()
                else:
                    log.debug(f"Ignoring a heat operation with no HA workflow for VM {vm} at {this_time}")
                continue

            # If it's successfully restored, close this event and append it.
            match = vm_restored_re.search(line)
            if match:
                # Close out this VM.
                vm = match.group(1)
                if vm in open_events:
                    open_events[vm].recovered_time = this_time
                    ha_events.append(open_events.pop(vm))
                else:
                    log.debug(f"Ignoring recovery with no HA workflow for VM {vm} at {this_time}")
                continue

        # We've gone through all the lines, so if any events are still open,
        # close them with no recovery.
        for vm, event in open_events.items():
            log.debug(f"Closing unrecovered HA event for VM {vm} at {event.start_time}")
            ha_events.append(event)

        return ha_events


class HAEvent:
    """ An HAEvent is an HA event, which corresponds to an attempt by the
    VNF-LAF to recover a single VM.
    Includes deployment name, VM name, HA workflow ID, HA workflow start time,
    VM recovered time, number of attempts.
    """
    def __init__(self, deployment, vm, ha_id, start_time):
        self.deployment = deployment
        self.vm = vm
        self.ha_id = ha_id
        self.start_time = start_time
        self.recovered_time = None
        self.attempts = 0

    def __str__(self):
        if args.csv:
            return f""""{self.deployment}","{self.vm}","{self.ha_id}","{self.start_time}","{self.recovered_time}",{self.attempts}"""
        else:
            return str(self.__dict__)

    def increment_attempts(self):
        self.attempts += 1

def process_logfiles(logfiles):
    """ Take a list of log files and return a list of Deployments.
    """
    deployments = {}
    for logfile in logfiles:
        # Get deployment name from filename.
        deployment_name = os.path.basename(logfile).split('_')[0]
        # Combine duplicates.
        if deployment_name not in deployments:
            deployments[deployment_name] = Deployment(deployment_name)
        # Append all relevant lines to the logs for this Deployment.
        with gzip.open(logfile, 'rt') as lf:
            for line in lf:
                for r in relevant_re:
                    if r.search(line):
                        deployments[deployment_name].logs.append(line)
                        break

    return deployments.values()

def main():
    # Turn the logfiles into a dict of deployment name to log lines.
    deployments = process_logfiles(args.logfiles)

    # Generate a total list of HA events from all deployments.
    ha_events = []
    for deployment in deployments:
        ha_events.extend(deployment.generate_ha_events())

    # Print a CSV header if needed.
    if args.csv:
        print("Deployment,VM,HA Workflow ID,Start Time (UTC),Recovered Time (UTC),Attempts")

    # Print HA events sorted by time.
    ha_events.sort(key=lambda x: x.start_time)
    if args.csv:
        for event in ha_events:
            print(str(event))
    else:
        dict_events = []
        for event in ha_events:
            dict_events.append(event.__dict__)
        print(json.dumps(dict_events, indent=4))

if __name__ == "__main__":
    main()
