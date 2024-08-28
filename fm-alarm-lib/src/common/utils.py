import argparse
import logging
import os

import configparser

from common.constants import DEFAULT_CONFIG_PATH
from common.exceptions import ConfigParsingException

SCRIPT_NAME = __name__


def get_logger(script_name=SCRIPT_NAME, log_level=logging.INFO):
    """
    Create a log object.

    :param script_name: name of the script that created a log object.
    :param log_level: level of printed messages to system console.
    :return: a log object.
    """

    log_format = '%(asctime)s | %(levelname)s | %(name)s.%(funcName)s - %(message)s'
    logging.basicConfig(format=log_format, level=log_level)

    return logging.getLogger(script_name)


def parse_args(provided_args=None):
    """
    Read the arguments from sys.argv and process it into an object.

    :return: a Namespace object with the arguments mapped in it.
    """
    parser = argparse.ArgumentParser()
    parser.add_argument('-f', '--config-file',
                        help=str.format("Config file path. (Default: {})", DEFAULT_CONFIG_PATH),
                        default=DEFAULT_CONFIG_PATH)
    parser.add_argument('-s', '--severity',
                        help="Alarm severity (INDETERMINATE/CRITICAL/MAJOR/MINOR/WARNING/CLEARED), (Default: "
                             "INDETERMINATE)",
                        default='INDETERMINATE')
    parser.add_argument('-r', "--record-type",
                        help="Record type  for internal alarm request (ALARM/NON_SYNCHABLE_ALARM/ERROR_MESSAGE), "
                             "(Default: ERROR_MESSAGE)",
                        default="ERROR_MESSAGE")
    parser.add_argument('-p', "--specific-problem",
                        help="This attribute is used to give a further refinement to the cause of the alarm in Text.")
    parser.add_argument('-c', "--probable-cause",
                        help="This attribute should give a hint of the general problem causing the alarm.")
    parser.add_argument('-e', "--event-type",
                        help="This attribute is used to give cause of raising an alarm.")
    parser.add_argument('-m', "--managed-object-instance",
                        help="This attribute specifies the service name raising internal alarm.")
    parser.add_argument('-a', "--additional-attributes",
                        help="Optional attributes to raise internal alarms (JSON format).",
                        default="{}")
    return parser.parse_args(args=provided_args)


def parse_config(file_path):
    """
    Read the config from config file process it into an object.

    :param: path to config file
    :return: a 2d dictionary with sections and keys
    """
    try:
        config = configparser.ConfigParser()
        config.read(file_path)
        return config
    except configparser.MissingSectionHeaderError as exp:
        raise ConfigParsingException(str.format("Error parsing config.\n{}", str(exp)))


def create_config_dir(path):
    if not os.path.exists(path):
        os.makedirs(path)
