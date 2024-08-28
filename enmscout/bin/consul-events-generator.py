#!/usr/bin/python3.6

"""
Consul events generator which leverages the ENM Consul Visualizer data.
by Loren Jan Wilson, 2019-10.

Mission: Grab events from the consulProm prometheus agent and create a log of
events which can be used for other purposes.

Purpose: It's good to have this information available in some format other than
a GUI for generating outage reports, etc.

Inputs:
    - prometheus config for the casbah consul visualizer - e.g. consul.yaml
    - a state file (from the last time we ran this script)
    - events files to keep up to date, if any

Outputs:
    - appended consul events onto the events files

Steps:
    - claim lock
    - load state file from last run
    - get time now (utcnow)
    - print a warning message with the old state file's timestamp if the old state file is too old (> 5 minutes)
    - parse prometheus config
    - grab metrics from {each deployment URL}/metrics
    - build new state
    - compare old state and new state, generating events for each change
    - append each event to a CSV file for this month
    - also append each event to a JSON file for this month
    - write state file
    - free lock

Caveats:
    - This script appends new events to the end of the files on disk, so if the
      number of fields changes between runs, you'll get malformed csv.
"""

import os
import sys
import datetime
import pickle
import yaml
import urllib.request
import json
import csv
import concurrent.futures
import enmscout

class ConsulEventsGenerator:
    def __init__(self, name):
        self.name = name

        self.prometheus_cfg = None
        self.state_file = None
        self.output_dir = None
        self.fresh_time = None
        self.http_timeout = None
        self.max_threads = None
        self.read_config()

        self.log = enmscout.configure_logging(self.name)

        self.former_state = {}
        self.current_state = {}
        self.deployments = []
        self.events = []

    def execute(self):
        """ This is the main routine for this class.
        It does all the stuff in the right order.
        """
        # Read prometheus config.
        self.read_prometheus_config()
        # Lock before doing anything important.
        with enmscout.interprocess_lock(self.name):
            self.log.debug("Collecting consul data and generating events")
            # Load state file from last run.
            self.load_state()
            # Print a warning message if the old state file is too old.
            self.check_freshness()
            # Grab metrics from each deployment.
            self.get_metrics()
            # Compare old state and new state, generating events for each change.
            self.generate_events()
            # Append each event to a CSV file for this month.
            self.append_to_events_file("csv")
            # Also append each event to a JSON file for this month.
            self.append_to_events_file("json")
            # Write state file.
            self.save_state()

    def read_config(self):
        """ Import the config file and make sure all the keys we need are in
        there.
        """
        conf = enmscout.load_config(self.name)

        # We want this to bomb out if any of the keys aren't defined.
        self.prometheus_cfg = conf['prometheus_cfg']
        self.state_file = conf['state_file']
        self.output_dir = conf['output_dir']
        self.fresh_time = int(conf['fresh_time'])
        self.http_timeout = int(conf['http_timeout'])
        self.max_threads = int(conf['max_threads'])

    def load_state(self):
        """ Load state from our last run, which is on disk.

        If the file doesn't exist, that should be ok, we'll just make it on
        this run and load it on the next run.

        If we can't read from it, bomb out.
        """
        if not os.path.exists(self.state_file):
            return

        with open(self.state_file, 'rb') as sf:
            self.former_state = pickle.load(sf)

    def save_state(self):
        """ Save this state to disk. If we can't do it, bomb out.
        """
        with open(self.state_file, 'wb') as sf:
            pickle.dump(self.current_state, sf)

    def check_freshness(self):
        """ Log a warning containing the old state file's timestamp if the old
        state is too old.

        If there isn't a former state to load, this will silently succeed.
        """
        self.current_state['datetime'] = datetime.datetime.utcnow()
        self.dt = self.current_state['datetime']
        if 'datetime' in self.former_state:
            oldest_acceptable_time = self.current_state['datetime'] \
                    - datetime.timedelta(seconds=self.fresh_time)
            if self.former_state['datetime'] < oldest_acceptable_time:
                msg = f"The last run has a timestamp older than {self.fresh_time} seconds: " \
                        + str(self.former_state['datetime'])
                self.log.warning(msg)
                # Create an event if the last run was stale, as well.
                cev = ConsulEvent(self.dt, None, None, msg)
                self.events.append(cev)
        else:
            self.log.warning(f"Couldn't find a former state with which to check freshness.")

    def read_prometheus_config(self):
        """ Read the prometheus config, which tells us how to get metrics.

        Bomb out if we can't do this.
        """
        with open(self.prometheus_cfg, 'r') as pc:
            deployments = []
            prom = yaml.safe_load(pc)
            for depl in prom['scrape_configs'][0]['static_configs']:
                # If there's more than one target per deployment, we'll ignore
                # it. Not a problem today, but tread lightly.
                address = depl['targets'][0]
                depl_name = depl['labels']['tenant']
                deployments.append(Deployment(depl_name, address, timeout=self.http_timeout))
            # Safe to call this twice during a run.
            self.deployments = deployments

    def request_and_parse_all_metrics(self):
        """ Load all metrics concurrently.
        Then parse the responses.
        """
        # Build the data structure.
        if 'metrics' not in self.current_state:
            self.current_state['metrics'] = {}

        with concurrent.futures.ThreadPoolExecutor(max_workers=self.max_threads) as executor:
            # Start the request operations.
            futures_to_depl = { executor.submit(depl.request_metrics): depl for depl in self.deployments }
            # This can fail in a few different ways.
            # We want to allow some requests to succeed even if others fail.
            try:
                for future in concurrent.futures.as_completed(futures_to_depl):
                    depl = futures_to_depl[future]
                    # This will return an exception if the http request failed.
                    exc = future.exception()
                    if exc:
                        self.log.warning(f"request_metrics failed for {depl.name}: {exc}")
                        continue
                    # This should throw an exception if there's no response to
                    # parse, or if the parsing fails for any other reason.
                    try:
                        depl.parse_metrics()
                    except:
                        self.log.traceback(f"parse_metrics failed for {depl.name}")
                        continue
                    # If we got this far, we should have a metrics dict, so
                    # copy it into the current state.
                    self.current_state['metrics'][depl.name] = depl.metrics
            except TimeoutError:
                # concurrent.futures.as_completed() will throw this exception if
                # any of the futures times out.
                self.log.traceback(f"request timeout")

    def get_metrics(self):
        """ Grab metrics from each deployment.
        """
        if self.deployments is None:
            self.log.warning("No deployments from which to grab metrics")
            return

        # Concurrently request all metrics.
        self.request_and_parse_all_metrics()

        # Check to make sure at least one succeeded. If not, we may have a
        # bigger problem on our hands...
        if 'metrics' not in self.current_state:
            msg = f"Couldn't get any URLs at all during this run, failing."
            self.log.critical(msg)
            # Note: before we raise, it could be helpful to attempt writing an
            # event about this, as well, in case nobody looks in the log file.
            raise Exception(msg)

    def generate_events(self):
        """ Generate events from the former and current state.
        Look through former events.
        For each one that exists, check to see if one exists in the new one.
        If it does, create an event for each state change.
        If an expected value is missing, create an event for that, as well.
        """
        if 'metrics' not in self.former_state:
            self.log.info("No former state metrics, so not generating events.")
            return

        if 'metrics' not in self.current_state:
            self.log.info("No current state metrics, so not generating events.")
            return

        dt = self.current_state['datetime']

        # Create event if any deployments are new in this run.
        for depl in self.current_state['metrics']:
            if depl not in self.former_state['metrics']:
                cev = ConsulEvent(dt, depl, None, "New deployment during this run")
                self.events.append(cev)

        for depl in self.former_state['metrics']:
            if depl not in self.current_state['metrics']:
                # If we couldn't get an entire deployment, create an event.
                cev = ConsulEvent(dt, depl, None, "no metrics for deployment on this run")
                self.events.append(cev)
                continue
            for vm, fvalue in self.former_state['metrics'][depl].items():
                if vm not in self.current_state['metrics'][depl]:
                    # If we couldn't get a metric for this VM, create an event.
                    cev = ConsulEvent(dt, depl, vm, "no metric for VM on this run")
                    self.events.append(cev)
                    continue
                cvalue = self.current_state['metrics'][depl][vm]
                if fvalue != cvalue:
                    # The money! This is the whole reason for this code to exist.
                    cev = ConsulEvent(dt, depl, vm, f"transition from {fvalue} to {cvalue}")
                    self.events.append(cev)

    def append_to_events_file(self, fileformat):
        """ Write out our events, keeping what's already there.
        """
        # We only support csv and json for now.
        if fileformat not in ["csv", "json"]:
            msg = f"fileformat not supported: {fileformat}"
            self.log.critical(msg)
            raise Exception(msg)

        # If there aren't any events, don't do anything.
        if len(self.events) == 0:
            return

        # Create a ConsulEventsFile and then append to it.
        events_file = ConsulEventsFile(self.current_state['datetime'],
                fileformat, self.output_dir, self.events)
        events_file.append()

class Deployment:
    serf_state = ( "None", "Alive", "Leaving", "Left", "Failed" )

    def __init__(self, name, address, timeout):
        self.name = name
        self.address = address
        self.timeout = timeout
        self.response = None
        self.metrics = {}

    def request_metrics(self):
        """ Request metrics.
        """
        metrics_url = f"http://{self.address}/metrics"
        with urllib.request.urlopen(metrics_url, timeout=self.timeout) as response:
            self.response = response.read()

    def parse_metrics(self):
        """ Parse the metrics from the response.
        """
        if not self.response:
            raise Exception("Called parse_metrics() with no response to parse")

        where_we_want = False
        for line in self.response.splitlines():
            line = line.decode('utf-8')
            # Fast-forward to "# TYPE status gauge".
            if "# TYPE status gauge" in line:
                where_we_want = True
                continue
            if not where_we_want:
                continue
            if "status{" not in line:
                continue
            try:
                # Each line looks like:
                # status{name="servicereg-1"} 1
                key_portion, value = line.split(' ', 1)
                unused, vm, unused = key_portion.split('"', 2)

                # If this value has a corresponding serf state, use it.
                # If not, just use the literal value.
                try:
                    state = self.serf_state[int(value)]
                    value = state
                except:
                    pass

                self.metrics[vm] = value
            except:
                self.log.warning(f"Can't parse a line: {line}")
                self.log.exception("Line parse error")

        # If we get to the end of the parse routine and we got nothing, raise
        # an exception.
        if len(self.metrics) == 0:
            raise Exception("No metrics after parsing response")

class ConsulEvent:
    def __init__(self, dt, deployment, vm, message):
        self.event_time = dt.strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3]
        self.deployment = deployment
        self.vm = vm
        self.message = message

    def generate_csv(self):
        return f"{self.event_time},{self.deployment},{self.vm},{self.message}"

class ConsulEventsFile:
    def __init__(self, dt, fileformat, output_dir, events):
        self.dt = dt
        self.fileformat = fileformat
        self.output_dir = output_dir
        self.events = events
        self.set_output_file()
        self.make_destination_dir()

    def set_output_file(self):
        dt = self.dt.date()
        self.output_file = os.path.join(self.output_dir,
                f"consul-events-{dt.year}-{dt.month}.{self.fileformat}")

    def make_destination_dir(self):
        """ Make the destination directory.
        """
        try:
            os.makedirs(self.output_dir)
        except FileExistsError:
            pass
        except:
            self.log.critical("Couldn't make destination directory: ", self.output_dir)
            raise

    def append(self):
        # Run the appropriate append routine.
        if self.fileformat == "csv":
            self.append_csv_events()
        elif self.fileformat == "json":
            self.append_json_events()
        else:
            raise Exception(f"Unsupported file format: {fileformat}")

    def append_csv_events(self):
        """ Append routine for a CSV file.
        """
        # Read in events from disk.
        disk_events = self.read_csv_events()
        # Generate a string of our new events.
        new_events = "\n".join([ e.generate_csv() for e in self.events ])
        # Concatenate the strings.
        if len(disk_events) > 0:
            all_event_str = "\n".join([disk_events, new_events])
        else:
            all_event_str = new_events
        # Write all events out to disk.
        self.overwrite_csv_events(all_event_str)

    def append_json_events(self):
        """ Append routine for a JSON file.
        """
        # Read in the json structure from disk.
        json_structure = self.read_json_events()
        # Append our new events.
        for e in self.events:
            json_structure["events"].append(e.__dict__)
        # Write all events out to disk.
        self.overwrite_json_events(json_structure)

    def read_csv_events(self):
        """ Read the file on disk and return a string of events.
        """
        events = []
        try:
            with open(self.output_file, 'r') as infile:
                csvreader = csv.reader(infile)
                # Skip header.
                next(csvreader)
                for row in csvreader:
                    events.append(row)
            return "\n".join(",".join(e) for e in events)
        except:
            # If there's no file on disk, that's ok.
            return ""

    def read_json_events(self):
        """ Read the file on disk and return a json structure.
        """
        try:
            with open(self.output_file, 'r') as infile:
                return json.load(infile)
        except:
            # If there's no file on disk, return an empty structure.
            empty = {}
            empty['events'] = []
            return empty

    def overwrite_csv_events(self, events_string):
        """ Overwrite the given csv file on disk with the given events.
        """
        with open(self.output_file, 'w') as outfile:
            outfile.write("Event Time,Deployment,VM,Event Message\n")
            outfile.write(events_string)
            outfile.write('\n')

    def overwrite_json_events(self, json_structure):
        """ Overwrite the given json file on disk with the given events.
        """
        with open(self.output_file, 'w') as outfile:
            outfile.write(json.dumps(json_structure))

def main():
    # This is the consul events generator.
    ceg = ConsulEventsGenerator('consul-events-generator')
    try:
        ceg.execute()
    except:
        if ceg.log:
            ceg.log.exception("Exception in main()")
        raise

if __name__ == "__main__":
    main()
