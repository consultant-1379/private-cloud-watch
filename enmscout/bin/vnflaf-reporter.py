#!/usr/bin/python3.6

"""
VNF-LAF report runner by Loren Jan Wilson, 2019-10.
Collect logs, then run all necessary reports.
This is meant to be run from cron periodically...every 5 or 10 minutes.

Only one can run at a time. It tries to acquire a lock before running.
"""

import os
import sys
import fcntl
import subprocess
import glob
import datetime
import enmscout

class Reporter:
    def __init__(self, name):
        self.name = name
        self.commands = []

        self.bin_dir = None
        self.deployments = None
        self.keys = None
        self.raw_logs_dir = None
        self.log_output_dir = None
        self.reports_dir = None
        self.read_config()

        self.log = enmscout.configure_logging(self.name)

    def read_config(self):
        """ Import the config file.
        """
        conf = enmscout.load_config(self.name)

        self.bin_dir = conf['bin_dir']
        self.deployments_file = conf['deployments_file']
        self.keys_dir = conf['keys_dir']
        self.raw_logs_dir = conf['raw_logs_dir']
        self.log_output_dir = conf['log_output_dir']
        self.reports_dir = conf['reports_dir']

    def make_report_dates(self, recent_days, timespan="month"):
        """ Return a list of recent dates.

        This is an optimization that enables us to run reports on recent dates
        instead of absolutely everything. Otherwise the execution time will
        increase in a linear fashion.

        Currently this returns the first day of each month that's recent
        enough. If you wanted to run one report per week, you could change
        this function.
        """
        if timespan != "month":
            raise NotImplementedError()

        recent_days_ago = datetime.datetime.utcnow().date() - datetime.timedelta(days=recent_days)
        start_date = datetime.date(recent_days_ago.year, recent_days_ago.month, 1)

        # This is based on how much data we actually have on disk for recent
        # months, which we can't assume.
        deployments = os.listdir(self.log_output_dir)
        recent_dates = set()
        # Each deployment directory contains year subdirectories and then
        # nested month subdirectories:
        # self.log_output_dir/tbaytel/2019/10
        for depl in deployments:
            depl_dir = os.path.join(self.log_output_dir, depl)
            years = os.listdir(depl_dir)
            for year in years:
                year_dir = os.path.join(depl_dir, year)
                months = os.listdir(year_dir)
                for month in months:
                    # Do some time math to make sure this is recent enough.
                    first_day_of_month = datetime.date(int(year), int(month), 1)
                    if start_date < first_day_of_month:
                        recent_dates.add(first_day_of_month)

        # Return objects that include a start date and a timespan.
        report_dates = []
        for rd in recent_dates:
            report_dates.append(ReportDate(rd, "month"))
        return report_dates

    def add_command(self, command):
        self.commands.append(command)

    def run(self):
        """ Run all commands in order.
        """
        with enmscout.interprocess_lock(self.name):
            for command in self.commands:
                command.execute(self.log)

    def add_vnflaf_log_slurper(self):
        """ Add a ReporterCommand which runs the VNF-LAF log slurper to import
        and UTC-timeshift the logs from all deployments.
        """
        command_args = [ os.path.join(self.bin_dir, 'vnflaf-log-slurper.py'),
            '--recent', self.deployments_file, self.keys_dir, self.raw_logs_dir,
            self.log_output_dir ]
        self.add_command(ReporterCommand(command_args))

    def add_vnflaf_events_generator(self, rdate):
        """ Add a ReporterCommand for the given ReportDate which runs the
        VNF-LAF events generator.
        """
        bin_loc = os.path.join(self.bin_dir, 'vnflaf-events-generator.py')
        year = rdate.year()
        month = rdate.month()
        input_files = glob.glob(os.path.join(self.log_output_dir, '*', year, month, '*'))
        out_file = os.path.join(self.reports_dir, 'ha-events', f'ha-events-{year}-{month}')
        enmscout.make_parent_dir(out_file)

        # One command for the CSV report.
        csv_command = [ bin_loc, '--csv' ] + input_files
        csv_out_file = f"{out_file}.csv"
        self.add_command(ReporterCommand(csv_command, out_file=csv_out_file,
            allow_failure=True))

        # And one command for the JSON report.
        json_command = [ bin_loc ] + input_files
        json_out_file = f"{out_file}.json"
        self.add_command(ReporterCommand(json_command, out_file=json_out_file,
            allow_failure=True))

    def add_ha_state_report(self, rdate):
        """ Add a ReporterCommand for the given ReportDate which runs the
        ha-state reporter.
        """
        bin_loc = os.path.join(self.bin_dir, 'ha-state.py')
        year = rdate.year()
        month = rdate.month()
        input_files = glob.glob(os.path.join(self.log_output_dir, '*', year, month, '*'))
        out_file = os.path.join(self.reports_dir, 'ha-state', f'ha-state-{year}-{month}.txt')

        command_args = [ bin_loc ] + input_files
        self.add_command(ReporterCommand(command_args, out_file=out_file,
            allow_failure=True))

class ReporterCommand:
    def __init__(self, command_args, out_file=None, allow_failure=False):
        self.command_args = command_args
        self.out_file = out_file
        self.allow_failure = allow_failure

    def execute(self, log):
        """ Run a given command, optionally writing output to a given file.
        Note: this blocks until the command is done, then writes output.
        """
        log.debug("running command: " + str(self.command_args))
        process = subprocess.run(self.command_args, stdout=subprocess.PIPE,
                stderr=subprocess.PIPE, universal_newlines=True)
        # Check to make sure the command succeeded.
        if process.returncode != 0:
            msg = f"Command run failed: {process}"
            if self.allow_failure:
                log.warning(msg)
            else:
                raise Exception(msg)
        # Send stderr to the log.
        for line in process.stderr.splitlines():
            log.warning(line)
        # Send stdout to a file, if requested.
        if self.out_file:
            enmscout.make_parent_dir(self.out_file)
            with open(self.out_file, 'w') as f:
                for line in process.stdout.splitlines():
                    f.write(line)
                    f.write('\n')
        return process

class ReportDate:
    def __init__(self, start_date, timespan):
        self.start_date = start_date
        self.timespan = timespan

    def year(self):
        return str(self.start_date.year)

    def month(self):
        # Two digit zero-padded month.
        return str(self.start_date.month).zfill(2)

def main():
    reporter = Reporter('vnflaf-reporter')

    try:
        reporter.add_vnflaf_log_slurper()
        report_dates = reporter.make_report_dates(60)
        for rdate in report_dates:
            reporter.add_vnflaf_events_generator(rdate)
            reporter.add_ha_state_report(rdate)

        reporter.run()

    except:
        if reporter.log:
            reporter.log.exception("Exception in main()")
        raise

if __name__ == "__main__":
    main()
