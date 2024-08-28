# PM KPI Script

## Introduction

This script reads data from the 2 pmserv folders in the DDC file share, and gets the value of the last 15 minute ROP file collection success and failures. 

## Collection and Calculation Steps

The DDC files are located in the `/var/ericsson/ddc_data` file share which is accessible from all VMs in the ENM install. We use EMP VM for our collection.

>In `ddc_data`, the folder names are `<deployment-name>-<service-group>-<instance-number>_TOR`. So for example, if we are looking for the folder with pmserv data for instance 1, the folder would be `frstagingenm01-pmserv-1_TOR` .

Inside the pmserv folder, a new folder is created for each day. So for the date `02-12-2019`, the folder would be called `021219`.

The data we need is in the file called `instr.txt`. So the final path to the data file for pmserv for deployment frstagingenm01 on date 02-12-2019 would be `/var/ericsson/ddc_data/frstagingenm01-pmserv-1_TOR/021219/instr.txt` .

### instr.txt File structure

The instr.txt file has the heading for each type of metric collected at the start of the file. In our case , we are interested in the `FileCollectionStatistics` metric for pmserv file collection. 

```bash
> grep FileCollectionStatistics instr.txt | head -n 1
02-12-19 00:00:05.942 CFG-frstagingenm01-pmserv-1-com.ericsson.oss.services.pm.collection.instrumentation.pm-service:type=FileCollectionStatistics combinedNumberOfFilesCollected combinedNumberOfFilesFailed combinedNumberOfStoredBytes combinedNumberOfTransferredBytes fifteenMinutesRopNumberOfFilesCollected fifteenMinutesRopNumberOfFilesFailed fifteenMinutesRopNumberOfStoredBytes fifteenMinutesRopNumberOfTransferredBytes oneMinuteRopNumberOfFilesCollected oneMinuteRopNumberOfFilesFailed oneMinuteRopNumberOfStoredBytes oneMinuteRopNumberOfTransferredBytes twentyFourRopNumberOfFilesCollected twentyFourRopNumberOfFilesFailed twentyFourRopNumberOfStoredBytes twentyFourRopNumberOfTransferredBytes
```
We are interested in fifteenMinutesRopNumberOfFilesCollected and fifteenMinutesRopNumberOfFilesFailed which are index number 8 and 9 respectively.

### Script Logic

1. Collects the last line of FileCollectionStatistics data in both instances of pmserv deployment.
2. For each instance, check if the timestamp in data collected is same as that of the previous collection. This is achieved with a txt file in the same directory as the script which contains the timestamps. If they are same, ignore the data and mark collection as failure. 
3. If the times are different, add the success and failure from both instances to get total.
4. Calculate the percentage with `(file_success / (file_success+file_fail)) * 100`

> If no nodes are subscribed for PM file collection, the data will be 0. Return a 100% so that no alarms are raised.

> If collection from both instances was marked as failed, exit with 0% to raise an alarm.

## Ansible Playbook

The ansible playbook here is pretty basic, it runs the script on the VM , reads the values and raises an alarm using the FM Lib. It excepts the Fm Lib to be already installed on the VM.

### Improvement

1. Make a workflow job with different modules so they could be reused with other scripts
2. Have the threshold be ansible variables rather than hard-coded