import unittest
from unittest import TestCase

from common.exceptions import ValidationException, ConfigValidationException
from common.utils import parse_args, parse_config
from common.validators import validate_args, validate_config_sections_and_keys, validate_severity_type, \
    validate_record_type, validate_mandatory_args_not_none, validate_additional_attributes_json


class TestValidateArgs(TestCase):
    """Class for unit testing validate args"""

    def test_validate_args_valid_args(self):
        """Assert no exception is raised while validating args"""
        args = ["-f./testResources/valid.conf", "-pproblems", "-ccause", "-etest", "-mpython", "-sCLEARED",
                "-rALARM", "-a0"]
        result = parse_args(args)
        validate_args(result)


class TestMandatoryArgsNotNone(TestCase):
    """Class for unit testing mandatory args not none"""

    def test_validate_mandatory_args_none(self):
        """Assert that AttributeError exception is raised when a mandatory attribute is missing."""
        args = ["-f./testResources/valid.conf", "-ccause", "-etest", "-mpython", "-sCLEARED",
                "-rALARM", "-a0"]
        result = parse_args(args)
        with self.assertRaises(AttributeError) as raised_exp:
            validate_mandatory_args_not_none(result)

        self.assertEqual("Missing 'specific_problem' parameter.", raised_exp.exception.message)

    def test_validate_mandatory_args_not_none(self):
        """Assert that no exception is raised when all attributes are present."""
        args = ["-f./testResources/valid.conf", "-ptest", "-ccause", "-etest", "-mpython", "-sCLEARED",
                "-rALARM", "-a0"]
        result = parse_args(args)
        validate_mandatory_args_not_none(result)


class TestValidateSeverity(TestCase):
    """Class for unit testing validate severity"""

    def test_validate_args_invalid_severity(self):
        """Assert ValidationException exception is raised while validating args"""
        args = ["-f./testResources/valid.conf", "-pproblems", "-ccause", "-etest", "-mpython", "-sCLEARffED",
                "-rALARM", "-a0"]
        result = parse_args(args)
        with self.assertRaises(KeyError) as raised_exp:
            validate_severity_type(result)

        self.assertEqual("Unknown perceived severity type CLEARffED.", raised_exp.exception.message)


class TestValidateRecordType(TestCase):
    """Class for unit testing validate record type"""

    def test_validate_args_invalid_record_type(self):
        """Assert ValidationException exception is raised while validating args"""
        args = ["-f./testResources/valid.conf", "-pproblems", "-ccause", "-etest", "-mpython", "-sCLEARffED",
                "-rALddARM", "-a0"]
        result = parse_args(args)
        with self.assertRaises(KeyError) as raised_exp:
            validate_record_type(result)

        self.assertEqual("Unknown record type ALddARM.", raised_exp.exception.message)


class TestValidateConfigFilePath(TestCase):
    """Class for unit testing validate config file path"""

    def test_validate_args_invalid_config_path(self):
        """Assert ValidationException exception is raised while validating args"""
        args = ["-f./testResource", "-pproblems", "-ccause", "-etest", "-mpython", "-sCLEARED",
                "-rALARM", "-a0"]
        result = parse_args(args)
        with self.assertRaises(ValidationException) as raised_exp:
            validate_args(result)

        self.assertEqual("[Errno Can't find config file at ] ./testResource", raised_exp.exception.message)


class TestValidateConfigSectionsAndKeys(TestCase):
    """Class for unit testing validate config file"""

    def test_validate_config_sections_and_keys_valid(self):
        """Assert that no exception is raised while validating config file"""
        path = "./testResources/valid.conf"
        result = parse_config(path)
        validate_config_sections_and_keys(result)

    def test_validate_config_sections_and_keys_invalid(self):
        """Assert that no exception is raised while validating config file"""
        path = "./testResources/missing.conf"
        result = parse_config(path)
        with self.assertRaises(ConfigValidationException) as raised_exp:
            validate_config_sections_and_keys(result)

        self.assertEqual("Missing config [alarm] [alarmThreshold]", raised_exp.exception.message)


class TestValidateAddtionalAttributes(TestCase):
    """Class to unit test validate additional attributes."""

    def test_validate_valid_json(self):
        """Assert no exception is raised for valid json."""
        attributes = '-a{"test":0}'
        args = ["-f./testResources/valid.conf", "-pproblems", "-ccause", "-etest", "-mpython", "-sCLEARffED",
                "-rALddARM", attributes]
        result = parse_args(args)
        validate_additional_attributes_json(result)

    def test_validate_invalid_json(self):
        """Assert ValueError exception is raised for valid json."""
        attributes = '-a{0:}'
        args = ["-f./testResources/valid.conf", "-pproblems", "-ccause", "-etest", "-mpython", "-sCLEARffED",
                "-rALddARM", attributes]
        result = parse_args(args)
        with self.assertRaises(ValueError) as raised_exp:
            validate_additional_attributes_json(result)
        self.assertIn("Can't parse additional_attributes, make sure it is in json format.",
                      raised_exp.exception.message)


if __name__ == '__main__':
    unittest.main()
