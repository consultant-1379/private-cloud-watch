import unittest
from unittest import TestCase

from common.exceptions import ConfigParsingException
from common.utils import parse_args, parse_config, get_logger


class TestParseArgs(TestCase):
    """Class for unit testing validate args"""

    def test_parse_args(self):
        """Assert if values can be accessed after parsing"""
        args = ["-f ./valid.conf", "-p problems", "-c cause", "-e test", "-m python", "-s CLEARED",
                "-r ALARM", "-a 0"]
        result = parse_args(args)
        self.assertEqual('./valid.conf', result.config_file.strip())
        self.assertEqual('problems', result.specific_problem.strip())
        self.assertEqual('cause', result.probable_cause.strip())
        self.assertEqual('test', result.event_type.strip())
        self.assertEqual('python', result.managed_object_instance.strip())
        self.assertEqual('CLEARED', result.severity.strip())


class TestParseConfig(TestCase):
    """Class for unit testing parse config"""

    def test_parse_config(self):
        """Assert if returns parsed config whe valid config file path is passed"""
        path = "./testResources/valid.conf"
        result = parse_config(path)

        self.assertEqual(["enm", "consul", "alarm"], result.sections())
        self.assertEqual("localhost", result["enm"]["haproxyIntIp"])
        self.assertEqual("8081", result["enm"]["haproxyPort"])
        self.assertEqual("localhost", result["consul"]["url"])
        self.assertEqual("8500", result["consul"]["port"])
        self.assertEqual("alarms", result["consul"]["folder"])
        self.assertEqual("300", result["alarm"]["alarmThreshold"])

    def test_parse_config_invalid_file_format(self):
        path = "./testResources/invalid.conf"
        with self.assertRaises(ConfigParsingException) as raised_exception:
            parse_config(path)
        self.assertIn("Error parsing config.", raised_exception.exception.message)


class UtilsGetLogTest(TestCase):
    """Class for unit testing get_log function."""

    def test_get_log(self):
        """Assert if the log object is created."""
        log = get_logger()
        self.assertIsNotNone(log, "Should have returned a log object.")


if __name__ == '__main__':
    unittest.main()
