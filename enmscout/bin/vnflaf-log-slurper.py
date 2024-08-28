#!/usr/bin/python3.6

"""
VNF jboss log slurper by loren jan wilson, 2019-10.

Mission: Go to the deployments, get the jboss logs, convert to UTC, output one
file per day.

Purpose: Provide data for scripts which analyze these logs and provide useful
info so that people don't have to sort through the logs by hand.

Input is:
- a config file with one deployment per line, including:
    - tenant name
    - ssh key location relative to keys dir
    - ip address of VNF services node (only one for now but someday we'll need
      more per deployment due to VNF HA)
- a directory containing the keys
- a working directory (e.g. /var/tmp/vnflaf)
- an output directory (e.g. /var/log/vnflaf)

Output is:
- per deployment, a log file per day, with timezones translated to UTC for each line
    - name by first date-time seen in the file, e.g.:
      /var/log/vnflaf/staging/2019-09-30T11:24:37.log.gz

What to do:
- for each deployment,
    - figure out the local timezone by asking the services node
    - rsync the current raw logs to a temporary location (one dir for each deployment)
    - run the log conversion routine for each deployment

Optimizations:
    - added --recent option to fast-forward to a recent date before processing
      logs, which will substantially speed up log processing.
        - throw out anything that might start late due to timezone shift...
          this means sampling one day more than we plan on keeping, to make
          sure that when we do the shift, we have everything we're supposed to
          have for the start day and later.

"""

import argparse
import os
import datetime
import pytz
import gzip
import subprocess
import traceback
import enmscout

my_name = 'vnflaf-log-slurper'

# Load some config items from the enmscout config.
conf = enmscout.load_config(my_name)
jboss_log_location = conf['jboss_log_location']
time_format = conf['time_format']
# How long we wait for a spawned process to return.
process_timeout = int(conf['process_timeout'])
# How recent is --recent? We sample recent_days+1 days' worth of logs, do the
# timezone conversion, and then round to recent_days days to cut off any
# incomplete days that would occur due to negative or positive timezone shift.
recent_days = int(conf['recent_days'])

# Pass a tenant info file and a keys location at the command line.
parser = argparse.ArgumentParser()
parser.add_argument('deployments_cfg', help='deployments.cfg')
parser.add_argument('keys_dir', help='keys directory')
parser.add_argument('work_dir', help='working directory')
parser.add_argument('out_dir', help='output directory')
parser.add_argument('--recent', action='store_true',
        help=f'only process logs from the last {recent_days} days')
parser.add_argument('--debug', action="store_true", help='debug logging')
args = parser.parse_args()

# If we're only processing recent logs, let's make sure we know what day to
# start collecting, as well as what day to round our logs off.
if args.recent:
    today = datetime.datetime.utcnow().date()
    rounding_date = today - datetime.timedelta(days=recent_days)
    collection_date = today - datetime.timedelta(days=(recent_days + 1))

# Set up logging.
log = enmscout.configure_logging(my_name, debug=args.debug)

class Deployment:
    """ A Deployment is an ENMaaS deployment.
    """
    def __init__(self, name, key, address):
        self.name = name
        self.key = key
        self.address = address
        self.timezone = None
        self.keypath = os.path.join(args.keys_dir, self.key)
        self.work_dir = os.path.join(args.work_dir, self.name)
        self.out_dir = os.path.join(args.out_dir, self.name)
        self.work_file = os.path.join(self.work_dir, "server.log")
        self.logs_transferred = False

    def request_timezone(self):
        """ Run the ssh command to request the timezone in a non-blocking fashion.
        Such as:
        ssh -i keys/staging01_cu_key.pem cloud-user@10.2.10.10 'timedatectl' | grep "Time zone" | awk '{print $3}'
        Not sure anymore whether this would be useful, so I'm removing it for now.
                "-o", "UserKnownHostsFile=/dev/null",
        """
        tz_cmd = ["ssh", "-i", self.keypath, "-o", "StrictHostKeyChecking=no",
                "-l", "cloud-user", self.address, "timedatectl"]

        self.tz_process = subprocess.Popen(tz_cmd, stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL, universal_newlines=True)

    def receive_timezone(self):
        """ Try to parse the result of request_timezone(). If we can't do it,
        timezone remains None.
        """
        try:
            stdout_data, stderr_data = self.tz_process.communicate(timeout=process_timeout)
            for line in stdout_data.splitlines():
                fields = line.split()
                if fields[1] == "zone:":
                    self.timezone = fields[2]
        except:
            log.traceback("Couldn't receive timezone")

    def request_jboss_logs(self):
        """ Run the rsync command to re-sync the raw jboss log for this deployment.
        Such as:
            rsync -e "ssh -i keys/staging01_cu_key.pem" cloud-user@10.2.10.10:/ericsson/3pp/jboss/standalone/log/server.log ./server.log
            Old line:
                "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i " + self.keypath,
        """
        # Make the parent directory for the file we're about to sync.
        enmscout.make_parent_dir(self.work_file)
        jbl_cmd = ["rsync", "-e",
                "ssh -o StrictHostKeyChecking=no -i " + self.keypath,
                "cloud-user@" + self.address + ":" + jboss_log_location,
                self.work_file]
        self.jbl_process = subprocess.Popen(jbl_cmd)

    def wait_for_jboss_logs(self):
        """ Wait for request_jboss_logs() to finish, and make sure it succeeded.
        """
        self.jbl_process.wait(timeout=process_timeout)
        if self.jbl_process.returncode == 0:
            self.logs_transferred = True
            log.info(f"{self.name} logs transferred")
        else:
            self.logs_transferred = False
            log.warning(f"{self.name} logs were not successfully transferred")

    def convert_logs(self):
        """ Convert the raw logs to UTC, and output to one gzip file per date.
        """
        # Make sure jboss logs have arrived first.
        if not self.logs_transferred:
            log.warning(f"Logs not transferred for {self.name}, so not converting logs")
            return
        # Make sure we have a timezone.
        if self.timezone is None:
            log.warning(f"Can't convert logs for {self.name} without timezone")
            return

        # Entries are generated using the first line we see for each date.
        # Use this to decide the output filename.
        date_to_filename = {}
        date_to_log = {}
        last_date = ""

        # Open the log file and parse each line.
        with open(self.work_file, 'r') as work_file:
            for line in work_file:
                # Parse the time and make sure it's legit.
                try:
                    fields = line.split(' ', 2)
                    year, month, day = fields[0].split('-')
                    # Skip anything that doesn't start with a 4-digit year.
                    if not (year.isnumeric and len(year) == 4 and year[0] == "2"):
                        continue
                except:
                    # If the time doesn't parse properly, write it to the last date
                    # seen, but don't worry if we don't have anywhere to write it.
                    # This should take care of multi-line entries after the first
                    # date shows up.
                    if last_date in date_to_log:
                        date_to_log[last_date].append(line.strip())
                    log.debug(f"Skipping timezone conversion for log line: {line}")
                    continue
                # If we're only processing recent logs, let's start collecting
                # from collection_date to make sure we'll have a full week's
                # worth after our timezone conversion. We'll need to truncate
                # logs at a strict recent_days day boundary after the timezone
                # conversion.
                if args.recent:
                    try:
                        this_date = datetime.date(int(year), int(month), int(day))
                        if this_date < collection_date:
                            continue
                    except:
                        log.warning(f"Time math for recent flag didn't work out: {line}")
                        continue
                # Convert the time to UTC.
                try:
                    date = fields[0]
                    # Yeah, the log contains multiple time formats. Nice.
                    # This converts "2019-08-28 04:17:21.446-0400" or "+0000" to the other format.
                    time = fields[1].replace('.', ',').replace('+', '-').split('-')[0]
                    date_time = date + " " + time
                    converted_datetime = pytz.timezone(self.timezone).localize(
                            datetime.datetime.strptime(date_time, time_format)
                            ).astimezone(pytz.utc)
                except:
                    log.warning(f"Couldn't do timezone conversion on line: {line}")
                    # Skip this line.
                    continue
                # If we're only using recent lines, skip this one unless it's
                # recent_days days ago or newer. The intent is for this to
                # begin at midnight, which is why we strip the time components
                # with date().
                if args.recent and converted_datetime.date() < rounding_date:
                    continue
                # Figure out where to write this line, and append it to the lines to be written.
                try:
                    line_to_write = converted_datetime.strftime(time_format)[:-3] + " " + fields[2].strip()
                    date_string = converted_datetime.strftime("%Y-%m-%d")
                    if date_string not in date_to_filename:
                        date_to_filename[date_string] = self.name + '_' + \
                                converted_datetime.strftime("%Y-%m-%dT%H:%M:%S.log.gz")
                    if date_string not in date_to_log:
                        date_to_log[date_string] = []
                    date_to_log[date_string].append(line_to_write)
                except:
                    log.critical(f"Couldn't figure out where to write line: {line_to_write}")
                    raise

        # For each date, write out its log file.
        for date in date_to_log:
            # Write the log lines for this date to a gzipped file with the decided-upon name.
            # Make a subdirectory per month.
            year, month, day = date.split('-')
            this_dir = os.path.join(self.out_dir, year, month)
            out_file = os.path.join(this_dir, date_to_filename[date])
            enmscout.make_parent_dir(out_file)
            with gzip.open(out_file, 'wt') as f:
                f.write('\n'.join(date_to_log[date]))

        log.info(f"{self.name} logs converted")

def make_deployments():
    # Parse tenant config file and make deployment objects.
    deployments = []
    with open(args.deployments_cfg, 'r') as deployments_cfg:
        for line in deployments_cfg:
            fields = line.split()
            deployments.append(Deployment(fields[0], fields[1], fields[2]))
    return deployments

def main():
    """ Overall strategy:
    1. Rsync all tenant logs in parallel.
    2. Get timezones from all tenants in parallel.
    3. For each tenant, run the log conversion routine.
    """

    # Parse the config file to create deployments.
    deployments = make_deployments()

    # Get timezones from all deployments in parallel.
    log.info("Requesting timezones...")
    for depl in deployments:
        depl.request_timezone()

    log.info("Loaded timezones")
    for depl in deployments:
        depl.receive_timezone()
        print(depl.name, depl.timezone)

    # Rsync all deployment logs in parallel.
    log.info("Requesting jboss logs...")
    for depl in deployments:
        depl.request_jboss_logs()

    log.info("Waiting for jboss logs to complete...")
    for depl in deployments:
        depl.wait_for_jboss_logs()

    # For each deployment, run the log conversion routine.
    log.info("Converting logs to UTC...")
    for depl in deployments:
        depl.convert_logs()

if __name__ == "__main__":
    main()
