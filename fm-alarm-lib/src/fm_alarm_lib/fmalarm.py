import hashlib
import json
import time

import requests
from requests import RequestException

from common.constants import Severity
from common.exceptions import EnmAlarmException
from common.utils import get_logger
from consul_utils.consul_utils import ConsulUtils

SCRIPT_NAME = __name__
LOG = get_logger(SCRIPT_NAME)


class FmAlarm:
    """Class to manage FM alarm and state."""

    def __init__(self, specific_problem, probable_cause, event_type, managed_object_instance,
                 perceived_severity, record_type, additional_attributes, fm_url, threshold, consul_config):
        self.specific_problem = specific_problem
        self.probable_cause = probable_cause
        self.event_type = event_type
        self.managed_object_instance = managed_object_instance
        self.perceived_severity = perceived_severity
        self.record_type = record_type
        self.additional_attributes = additional_attributes
        self.fm_url = fm_url
        self.threshold = int(threshold)
        self.hash = self.get_alarm_hash()
        self.consul_util = ConsulUtils(consul_config)

    def send_alarm_request(self):
        """
        Method to send FM alarm POST to fm service
        :return: None
        :raises: EnmAlarmException if alarm raise request unsuccessful
        """
        if self.perceived_severity is Severity.CLEARED and not self.consul_util.does_hash_exist(self.hash):
            LOG.info("Alarm does not exist, no need to clear.")
            return

        time_delta = time.time() - time.mktime(time.localtime(self.get_alarm_last_raised()))
        if time_delta > self.threshold or self.perceived_severity is Severity.CLEARED:
            LOG.info(str.format("Threshold for alarm exceeded, or alarm cleared. Sending to FM.\n Severity = {}",
                                self.perceived_severity.name))
            try:
                payload = {
                    'Content-Type': 'application/json',
                    'Accept': 'application/json'
                }
                response = requests.post(self.fm_url, data=self.create_json_string(), headers=payload)
                if response.ok:
                    if self.perceived_severity is Severity.CLEARED:
                        self.consul_util.delete_by_hash(self.hash)  # delete from consul
                        LOG.info("Alarm deleted from consul.")
                    else:
                        self.consul_util.add_key_and_currtime(self.hash)  # update on consul
                        LOG.info("Alarm saved to consul.")
                else:
                    raise EnmAlarmException(str.format("Can't Raise ENM alarm."
                                                       "\nResponse code: {} "
                                                       "\nResponse Message: {}",
                                                       response.status_code, str(response.content)))
            except (RequestException) as exp:
                raise EnmAlarmException(str.format("Error connecting to ENM.\n{} ", str(exp)))
        else:
            LOG.info(str.format("Time exceeded is less than the threshold at {0:.2f}", time_delta))

    def get_alarm_hash(self):
        """
        Generate a sha1 hash of the alarm using specific problem  and probable cause
        :return: a sha1 hash
        """
        string_to_hash = str(self.specific_problem) + str(self.probable_cause)
        string_to_hash = string_to_hash.encode('utf-8')
        return str(hashlib.sha1(string_to_hash).hexdigest())

    def get_alarm_last_raised(self):
        """
        Get the last time a new FM alarm was raised for the given alarm.
        :return: epoch time in seconds of when the alarm was last raised.
        """
        value = self.consul_util.get_value_by_hash(self.hash)
        if value[1] is not None:
            return float(value[1]['Value'])
        return 0

    def create_json_string(self):
        """
        create a json string to send to FM service.
        :return: A json string representation of the alarm.
        """
        data = {
            "specificProblem": self.specific_problem,
            "probableCause": self.probable_cause,
            "eventType": self.event_type,
            "managedObjectInstance": self.managed_object_instance,
            "perceivedSeverity": self.perceived_severity.name,
            "recordType": self.record_type.name,
            "additionalAttributes": self.additional_attributes
        }

        return json.dumps(data)
