import os
from os.path import expanduser

from enum import Enum

OPTIONAL_ARGS = ['additional_attributes']

MANDATORY_ARGS = ['config_file', 'event_type', 'managed_object_instance', 'probable_cause', 'specific_problem',
                  'record_type', 'severity']

MANDATORY_CONFIG = {
    "enm": ["haproxyIntIp", "haproxyPort"],
    "consul": ["url", "port", "folder"],
    "alarm": ["alarmThreshold"]
}

ENM_FM_PATH = "/internal-alarm-service/internalalarm/internalalarmservice/translate"

DEFAULT_CONFIG_DIRECTORY_PATH = os.path.join("/usr/local/etc", "fmalarm")
DEFAULT_CONFIG_PATH = os.path.join(DEFAULT_CONFIG_DIRECTORY_PATH, "alarm.conf")


class Severity(Enum):
    INDETERMINATE = 1
    CRITICAL = 2
    MAJOR = 3
    MINOR = 4
    WARNING = 5
    CLEARED = 6


class RecordType(Enum):
    ALARM = 1
    NON_SYNCHABLE_ALARM = 2
    ERROR_MESSAGE = 3
