import json
import time
import unittest
from unittest import TestCase

import mock
import requests_mock
from mock import patch

from common.constants import Severity, RecordType
from common.exceptions import EnmAlarmException
from consul_utils.consul_utils import ConsulUtils
from fm_alarm_lib.fmalarm import FmAlarm

PACKAGE = "fm_alarm_lib.fmalarm."
MOCK_LOG = PACKAGE + "LOG"


def get_fm_alarm_object():
    """Helper to get a fm alarm object."""
    consul_config = {
        'consul_url': 'localhost',
        'consul_port': '8500',
        'alarms_folder': 'alarms'
    }
    return FmAlarm(additional_attributes={},
                   specific_problem="specific_problem",
                   probable_cause="probable_cause",
                   event_type="event_type",
                   managed_object_instance="mo",
                   perceived_severity=Severity.MINOR,
                   record_type=RecordType.ALARM,
                   fm_url="http://fm_url",
                   threshold=500,
                   consul_config=consul_config
                   )


class TestGetAlarmHash(TestCase):
    """Class for unit testing get_alarm_hash"""

    def test_get_alarm_hash(self):
        """Assert that calculated hash is equal to the generated hash."""
        expected_hash = "8c3b840fd1adea3e50dfa091cae732803c746948"
        fm = get_fm_alarm_object()
        self.assertEqual(expected_hash, fm.get_alarm_hash())


class TestCreateJson(TestCase):
    """Class to unit test create_json_string method"""

    def test_create_json_string(self):
        """Assert that the generated json is equal to the expected json."""
        data = {
            "specificProblem": "specific_problem",
            "probableCause": "probable_cause",
            "eventType": "event_type",
            "managedObjectInstance": "mo",
            "perceivedSeverity": Severity.MINOR.name,
            "recordType": RecordType.ALARM.name,
            "additionalAttributes": {}
        }
        json_string = json.dumps(data)
        fm = get_fm_alarm_object()
        self.assertEqual(json_string, fm.create_json_string())


class TestGetAlarmLastRaised(TestCase):
    """Class to unit test last_alarm_raised"""

    def test_get_alarm_last_raised(self):
        """Assert that last alarm raised return value is equal to expected value"""
        value = (33243, {
            "LockIndex": 0,
            "Key": "alarms/8c3b840fd1adea3e50dfa091cae732803c746948",
            "Flags": 0,
            "Value": "1573127121.94",
            "CreateIndex": 33243,
            "ModifyIndex": 33243
        })
        with patch.object(ConsulUtils, 'get_value_by_hash', return_value=value) as mock_get_value_by_hash:
            fm = get_fm_alarm_object()
            result = fm.get_alarm_last_raised()
        self.assertEquals(1573127121.94, result)
        mock_get_value_by_hash.assert_called_with("8c3b840fd1adea3e50dfa091cae732803c746948")


class TestSendAlarmRequest(TestCase):
    """Class to unit test send_alarm_request."""

    def setUp(self):
        self.baseurl = 'http://fm_url'

    @mock.patch(MOCK_LOG)
    def test_send_no_clear(self, mock_log):
        """Assert alarm clear not sent if no alarm exists. """
        fm = get_fm_alarm_object()
        fm.perceived_severity = Severity.CLEARED
        log_calls = [mock.call("Alarm does not exist, no need to clear.")]
        with patch.object(ConsulUtils, 'does_hash_exist', return_value=False):
            fm.send_alarm_request()
        mock_log.info.assert_has_calls(log_calls)

    @mock.patch(MOCK_LOG)
    def test_send_time_not_exceeded(self, mock_log):
        """Assert alarm not sent if not enough time has passed since the last alarm."""
        fm = get_fm_alarm_object()
        fm.perceived_severity = Severity.MINOR
        log_calls = [mock.call("Time exceeded is less than the threshold at 100.94")]
        return_value = (0, {
            "Value": '1573126121.94'
        })
        with patch.object(ConsulUtils, 'get_value_by_hash',
                          return_value=return_value):
            with patch.object(time, 'time', return_value=1573126221.94):
                fm.send_alarm_request()
        mock_log.info.assert_has_calls(log_calls)

    @mock.patch(MOCK_LOG)
    @mock.patch("fm_alarm_lib.fmalarm.ConsulUtils")
    def test_send_clear(self, mock_consul, mock_log):
        """Assert alarm clear is sent successfully."""
        fm = get_fm_alarm_object()
        fm.perceived_severity = Severity.CLEARED
        log_calls = [mock.call("Threshold for alarm exceeded, or alarm cleared. Sending to FM.\n Severity = CLEARED"),
                     mock.call("Alarm deleted from consul.")]
        with requests_mock.Mocker() as m:
            m.post(self.baseurl)
            mock_consul.does_hash_exist.return_value = True
            mock_consul.get_value_by_hash.return_value = (0, {
                "Value": '1573126121.94'
            })
            mock_consul.delete_by_hash.return_value = True
            with patch.object(time, 'time', return_value=1573127121.94):
                fm.send_alarm_request()
        mock_log.info.assert_has_calls(log_calls)

    @mock.patch(MOCK_LOG)
    @mock.patch("fm_alarm_lib.fmalarm.ConsulUtils")
    def test_send_alarm(self, mock_consul, mock_log):
        """Assert alarm create is successful"""
        fm = get_fm_alarm_object()
        log_calls = [mock.call("Threshold for alarm exceeded, or alarm cleared. Sending to FM.\n Severity = MINOR"),
                     mock.call("Alarm saved to consul.")]
        with requests_mock.Mocker() as m:
            m.post(self.baseurl)
            mock_consul.does_hash_exist.return_value = True
            mock_consul.get_value_by_hash.return_value = (0, {
                "Value": '1573126121.94'
            })
            mock_consul.delete_by_hash.return_value = True
            with patch.object(time, 'time', return_value=1573127121.94):
                fm.send_alarm_request()
        mock_log.info.assert_has_calls(log_calls)

    @mock.patch("fm_alarm_lib.fmalarm.ConsulUtils")
    def test_send_alarm_exception(self, mock_consul):
        """Assert that EnmAlarmException is raised when alarm not sent successfully."""
        fm = get_fm_alarm_object()
        with requests_mock.Mocker() as m:
            m.post(self.baseurl, status_code=500)
            mock_consul.does_hash_exist.return_value = True
            mock_consul.get_value_by_hash.return_value = (0, {
                "Value": '1573126121.94'
            })
            mock_consul.delete_by_hash.return_value = True
            with patch.object(time, 'time', return_value=1573127121.94):
                with self.assertRaises(EnmAlarmException) as raised_exp:
                    fm.send_alarm_request()
            self.assertEqual("Can't Raise ENM alarm.\nResponse code: 500 \nResponse Message:",
                             raised_exp.exception.message.strip())


if __name__ == '__main__':
    unittest.main()
