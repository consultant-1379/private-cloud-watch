#!/usr/bin/env python3

import argparse
import re
from datetime import datetime, timedelta

# Pass an openstack compute log file in at the command line.
parser = argparse.ArgumentParser()
parser.add_argument('log_file', help='openstack compute log file')
args = parser.parse_args()

def main():
    # Open the file and read it into memory, sorry in advance.
    entries = []
    with open(args.log_file, "r") as log_file:
        for line in log_file:
            # If this line doesn't parse properly, let's add its contents to
            # the "message" field on the last line.
            try:
                fields = line.split(' ', 5)
                year = fields[0].split('-')[0]
                if (not year.isnumeric or len(year) != 4):
                    raise Exception("Doesn't start with a year")
                # Create an entry in the unhealthy matches.
                isotime = fields[0] + 'T' + fields[1]
                #dt = datetime.strptime(isotime, "%Y-%m-%dT%H:%M:%S.%f")
                this_entry = {
                    "date": fields[0],
                    "time": fields[1],
                    "isotime": isotime,
                    "pid": fields[2],
                    "level": fields[3],
                    "thread": fields[4]
                }
                # A little more parsing has to be done to the message itself,
                # to pull out the req ids and the instance id, if either of
                # these are present.
                message = fields[5]
                req_ids = []
                instance = ""
                # This is an "empty" req string.
                if message[:4] == '[-] ':
                    message = message[4:]
                # This is a req string, which has multiple req IDs in brackets.
                if message[:5] == '[req-':
                    s = message[1:].split(']', 1)
                    req_ids = s[0].split()
                    # Remove the whole string from the message.
                    message = s[1].strip()
                # This is an instance ID... they're always in brackets.
                if message[:11] == '[instance: ':
                    s = message[1:].split(']', 1)
                    instance = s[0].split()[1]
                    # Remove this string from the message.
                    message = s[1].strip()
                this_entry["message"] = message
                # Only set them if they aren't empty.
                if len(req_ids) > 0:
                    this_entry["request_ids"] = req_ids
                if instance:
                    this_entry["instance"] = instance
            except:
                # If there's any parsing problem at all above, just append this
                # message verbatim to the last entry, and give up.
                try:
                    entries[-1]["message"] += line
                except:
                    print("Throwing out a log line because it's malformed but I have nowhere to put it")
                continue

            # We've parsed this entry, but it still might actually be a
            # continuation of the previous log entry.
            try:
                last_entry = entries[-1]

                # All 5 of these fields must be the same in order to be
                # considered a continuation.

                if ((this_entry["date"] != last_entry["date"]) or
                    (this_entry["time"] != last_entry["time"]) or
                    (this_entry["pid"] != last_entry["pid"]) or
                    (this_entry["level"] != last_entry["level"]) or
                    (this_entry["thread"] != last_entry["thread"])):
                    raise Exception("Last entry isn't the same as this one")

                # The instance ID must also be the same, if there is one, but
                # in cases where they aren't included in the second message,
                # that's still ok (and common).
                if (("instance" in last_entry) and
                    ("instance" in this_entry) and
                    (last_entry["instance"] != this_entry["instance"])):
                    raise Exception("Different instance ID")

                # The request IDs must also all be the same, if present.
                if (("request_ids" in last_entry) and
                    ("request_ids" in this_entry) and
                    (last_entry["request_ids"] != this_entry["request_ids"])):
                    raise Exception("Different request IDs")

                # All right, assume they're the same entry, and append.
                last_entry["message"] += "\n" + this_entry["message"]

                # In some cases, the instance id won't be set on the first
                # line, but it'll be set later. In that case, set it.
                if (("instance" not in last_entry) and
                    ("instance" in this_entry)):
                    last_entry["instance"] = this_entry["instance"]

                if (("request_ids" not in last_entry) and
                    ("request_ids" in this_entry)):
                    last_entry["request_ids"] = this_entry["request_ids"]
            except:
                # Create a new log entry.
                entries.append(this_entry)

    # Walk through and attempt to annotate the entries by looking around.
    # There should be a certain "blast radius" which healthy or unhealthy
    # patterns are allowed to affect, given the same req ids and instance.
    # Three minutes seems ok for now.
    blast_radius = 180

    # If we see any unhealthy messages, that takes precedence over the healthy
    # messages in the same vicinity. If we don't see either, we don't make a
    # judgment one way or the other.

    # Here are some patterns we know are unhealthy.
    unhealthy_patterns = [
        'Traceback (most recent call last):',
        'Instance failed to shutdown in [\d\.]+ seconds',
        'Instance failed to spawn',
        'Setting instance vm_state to ERROR',
        'Failed to deallocate network for instance',
        'Unable to establish connection to',
        'Failed to establish a new connection',
        'Could not clean up failed build',
    ]

    # And some healthy patterns.
    healthy_patterns = [
        'Instance shutdown successfully after [\d\.]+ seconds',
        'Deletion of [0-9a-z\-]+ complete',
        'Took [\d\.]+ \w+ to destroy the instance on the hypervisor',
        'Took [\d\.]+ \w+ to deallocate network for instance',
        'Took [\d\.]+ \w+ to detach .* volumes for instance'
        'Took [\d\.]+ \w+ to spawn the instance on the hypervisor',
    ]
    unhealthy_re = [ re.compile(s) for s in unhealthy_patterns ]
    healthy_re = [ re.compile(s) for s in healthy_patterns ]

    # Define the matches by creating some dicts keyed from request_ids and
    # instance id.
    unhealthy_matches = {}
    healthy_matches = {}
    unhealthy_instance_matches = {}
    healthy_instance_matches = {}
    for entry in entries:
        # Technically, one message could match both, but unhealthiness will
        # take precedence.
        healthy = False
        unhealthy = False
        for r in healthy_re:
            if r.match(entry["message"]):
                healthy = True
                break
        for r in unhealthy_re:
            if r.match(entry["message"]):
                unhealthy = True
                break
        # Set "null" and flatten these to strings to use them as dict keys.
        this_request_ids = "null"
        if "request_ids" in entry:
            this_request_ids = ",".join(entry["request_ids"])
        this_instance = "null"
        if "instance" in entry:
            this_instance = entry["instance"]
        # If we're unhealthy, append an isotime to the unhealthy array.
        # Otherwise, if we're healthy, do the same to the healthy array.
        if unhealthy:
            if this_request_ids not in unhealthy_matches:
                unhealthy_matches[this_request_ids] = {}
            if this_instance not in unhealthy_matches[this_request_ids]:
                unhealthy_matches[this_request_ids][this_instance] = []
            unhealthy_matches[this_request_ids][this_instance].append(entry["isotime"])
            if this_instance not in unhealthy_instance_matches:
                unhealthy_instance_matches[this_instance] = []
            unhealthy_instance_matches[this_instance].append(entry["isotime"])
        elif healthy:
            if this_request_ids not in healthy_matches:
                healthy_matches[this_request_ids] = {}
            if this_instance not in healthy_matches[this_request_ids]:
                healthy_matches[this_request_ids][this_instance] = []
            healthy_matches[this_request_ids][this_instance].append(entry["isotime"])
            if this_instance not in healthy_instance_matches:
                healthy_instance_matches[this_instance] = []
            healthy_instance_matches[this_instance].append(entry["isotime"])

    # For each log entry, we're going to see if it's in the vicinity of
    # something healthy or unhealthy, and if so, that's going to color our
    # opinion of this entry.
    for entry in entries:
        # Copied from above, turn this into a function.
        this_request_ids = "null"
        if "request_ids" in entry:
            this_request_ids = ",".join(entry["request_ids"])
        this_instance = "null"
        if "instance" in entry:
            this_instance = entry["instance"]
        # Convert the time to a datetime so we can do arithmetic with it.
        time_format = "%Y-%m-%dT%H:%M:%S.%f"
        entry_time = datetime.strptime(entry["isotime"], time_format)
        # Innocent until proven guilty.
        entry["health"] = "unknown"
        # Is there anything nearby that's unhealthy?
        health_time = ""
        unhealthy = False
        if this_request_ids in unhealthy_matches:
            if this_instance in unhealthy_matches[this_request_ids]:
                unhealthy_times = unhealthy_matches[this_request_ids][this_instance]
                for ut in unhealthy_times:
                    this_time = datetime.strptime(ut, time_format)
                    if abs(entry_time - this_time) < timedelta(seconds=blast_radius):
                        unhealthy = True
                        health_time = ut
                        break
        if unhealthy:
            entry["health"] = "unhealthy"
            entry["health_time"] = health_time
            continue
        # It's possible that this is "possibly unhealthy", if the request ids
        # don't match but the instance id does.
        possibly_unhealthy = False
        if "instance" in entry and this_instance in unhealthy_instance_matches:
            unhealthy_times = unhealthy_instance_matches[this_instance]
            for ut in unhealthy_times:
                this_time = datetime.strptime(ut, time_format)
                if abs(entry_time - this_time) < timedelta(seconds=blast_radius):
                    possibly_unhealthy = True
                    health_time = ut
                    break
        if possibly_unhealthy:
            entry["health"] = "possibly_unhealthy"
            entry["health_time"] = health_time
            continue
        # How about healthy? (lots of code repetition here, sorry.)
        healthy = False
        if this_request_ids in healthy_matches:
            if this_instance in healthy_matches[this_request_ids]:
                healthy_times = healthy_matches[this_request_ids][this_instance]
                for ht in healthy_times:
                    this_time = datetime.strptime(ht, time_format)
                    if abs(entry_time - this_time) < timedelta(seconds=blast_radius):
                        healthy = True
                        health_time = ht
                        break
        if healthy:
            entry["health"] = "healthy"
            entry["health_time"] = health_time
            continue
        # Or possibly healthy?
        possibly_healthy = False
        if "instance" in entry and this_instance in healthy_instance_matches:
            healthy_times = healthy_instance_matches[this_instance]
            for ht in healthy_times:
                this_time = datetime.strptime(ht, time_format)
                if abs(entry_time - this_time) < timedelta(seconds=blast_radius):
                    possibly_healthy = True
                    health_time = ht
                    break
        if possibly_healthy:
            entry["health"] = "possibly_healthy"
            entry["health_time"] = health_time
            continue

    for entry in entries:
        print(entry)

if __name__ == "__main__":
    main()
