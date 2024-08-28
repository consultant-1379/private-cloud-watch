import json
import os

from common.constants import MANDATORY_ARGS, MANDATORY_CONFIG, Severity, RecordType
from common.exceptions import ValidationException, ConfigValidationException

SCRIPT_NAME = __name__


def validate_args(args):
    """
    Validates the input arguments.
    :param args: ClI arguments.
    :return: validated arguments.
    :raises: validationException when arg is missing or invalid.
    """
    try:
        validate_mandatory_args_not_none(args)
        validate_config_file_path_valid(args)
        validate_severity_type(args)
        validate_record_type(args)
        validate_additional_attributes_json(args)
    except (AttributeError, KeyError, TypeError, OSError, ValueError) as error:
        raise ValidationException(str(error))


def validate_mandatory_args_not_none(argument):
    """
    Validates mandatory input arguments.
    :param argument: ClI arguments.
    :raises: AttributeError when arg is missing or invalid.
    """
    for arg in MANDATORY_ARGS:
        if vars(argument)[arg] is None:
            raise AttributeError(str.format("Missing '{}' parameter.", arg))


def validate_config_file_path_valid(argument):
    """
    Validates that config file exists.
    :param argument: cli arguments.
    :raises: OSError if config file does not exist.
    """
    if not os.path.exists(argument.config_file):
        raise OSError("Can't find config file at ", argument.config_file)


def validate_config_sections_and_keys(config):
    """
    Validate that all needed sections and their keys are present in config file.
    :param config: dictionary with config arguments.
    :raises: ConfigValidationException if missing config section/key .
    """
    for section in MANDATORY_CONFIG:
        for key in MANDATORY_CONFIG[section]:
            try:
                if config[section][key] is None:
                    raise ConfigValidationException(str.format("Empty config [{}] [{}]", section, key))
            except KeyError:
                raise ConfigValidationException(str.format("Missing config [{}] [{}]", section, key))


def validate_additional_attributes_json(arguments):
    """
    Validate if additional attributes is valid json.
    :param arguments: cli args
    """
    try:
        json.loads(arguments.additional_attributes)
    except ValueError as exp:
        raise ValueError(
            str.format("Can't parse additional_attributes, make sure it is in json format.\n{}", exp.message))


def validate_severity_type(argument):
    """
    Validate if severity passed in cli args is valid.
    :param argument: cli args.
    :raises: KeyError if passed severity does not exist in enum.
    """
    if not hasattr(Severity, argument.severity):
        raise KeyError(str.format("Unknown perceived severity type {}.", argument.severity))


def validate_record_type(argument):
    """
    Validates if record type passed in cli args is valid.
    :param argument: cli args.
    :raises: KeyError if passed record type does not exist in enum
    """
    if not hasattr(RecordType, argument.record_type):
        raise KeyError(str.format("Unknown record type {}.", argument.record_type))
