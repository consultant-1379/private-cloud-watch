import json
import unittest
from unittest import TestCase

import requests_mock

from common.exceptions import ConsulUtilsException
from consul_utils.consul_utils import ConsulUtils


class TestGetValueByKey(TestCase):
    """Class to unit test value by key"""

    def setUp(self):
        self.baseurl = 'http://localhost:8500'

    def test_get_value_by_key_exists(self):
        """Assert object is returned by given hash when key exists"""
        key_hash = "b51abe34098043925b8d0d001ad1f6ade0f48608"
        url = str.format("{}/v1/kv/alarms/{}", self.baseurl, key_hash)
        response = [
            {
                "LockIndex": 0,
                "Key": "alarms/b51abe34098043925b8d0d001ad1f6ade0f48608",
                "Flags": 0,
                "Value": "MTU3MzEyNzEyMS45NA==",
                "CreateIndex": 33243,
                "ModifyIndex": 33243
            }
        ]
        headers = {
            "x-consul-index": "33243"
        }
        with requests_mock.Mocker() as m:
            m.get(url,
                  text=json.dumps(response),
                  headers=headers)
            consul_config = {
                'consul_url': 'localhost',
                'consul_port': '8500',
                'alarms_folder': 'alarms'
            }
            consul_util = ConsulUtils(consul_config)
            value = consul_util.get_value_by_hash(key_hash)

        self.assertEqual('1573127121.94', value[1]['Value'])
        self.assertEqual('alarms/b51abe34098043925b8d0d001ad1f6ade0f48608', value[1]['Key'])

    def test_get_value_by_key_not_exists(self):
        key = "doesnotexist"
        url = str.format("{}/v1/kv/alarms/{}", self.baseurl, key)
        headers = {
            "x-consul-index": "0"
        }
        with requests_mock.Mocker() as m:
            m.get(url, headers=headers, status_code=404)
            consul_config = {
                'consul_url': 'localhost',
                'consul_port': '8500',
                'alarms_folder': 'alarms'
            }
            consul_util = ConsulUtils(consul_config)
            value = consul_util.get_value_by_hash(key)
        self.assertIsNone(value[1])


class TestDoesKeyExist(TestCase):

    def setUp(self):
        self.baseurl = 'http://localhost:8500'

    def test_does_key_exists_true(self):
        key_hash = "b51abe34098043925b8d0d001ad1f6ade0f48608"
        url = str.format("{}/v1/kv/alarms/{}", self.baseurl, key_hash)
        response = [
            {
                "LockIndex": 0,
                "Key": "alarms/b51abe34098043925b8d0d001ad1f6ade0f48608",
                "Flags": 0,
                "Value": "MTU3MzEyNzEyMS45NA==",
                "CreateIndex": 33243,
                "ModifyIndex": 33243
            }
        ]
        headers = {
            "x-consul-index": "33243"
        }
        with requests_mock.Mocker() as m:
            m.get(url,
                  text=json.dumps(response),
                  headers=headers)
            consul_config = {
                'consul_url': 'localhost',
                'consul_port': '8500',
                'alarms_folder': 'alarms'
            }
            consul_util = ConsulUtils(consul_config)

            self.assertTrue(consul_util.does_hash_exist("b51abe34098043925b8d0d001ad1f6ade0f48608"))

    def test_get_value_by_key_not_exists(self):
        key = "doesnotexist"
        url = str.format("{}/v1/kv/alarms/{}", self.baseurl, key)
        headers = {
            "x-consul-index": "0"
        }
        with requests_mock.Mocker() as m:
            m.get(url, headers=headers, status_code=404)
            consul_config = {
                'consul_url': 'localhost',
                'consul_port': '8500',
                'alarms_folder': 'alarms'
            }
            consul_util = ConsulUtils(consul_config)
            self.assertFalse(consul_util.does_hash_exist(key))


class TestAddKey(TestCase):

    def setUp(self):
        self.baseurl = "http://localhost:8500"

    def test_add_successful(self):
        key = "alarms/key"
        value = "value"
        url = str.format("{}/v1/kv/{}", self.baseurl, key)
        headers = {
            "x-consul-index": "0"
        }
        with requests_mock.Mocker() as m:
            m.put(url, headers=headers, status_code=200, text="true")

            consul_config = {
                'consul_url': 'localhost',
                'consul_port': '8500',
                'alarms_folder': 'alarms'
            }
            consul_util = ConsulUtils(consul_config)
            self.assertTrue(consul_util.add_key_value(key, value))

    def test_add_exception(self):
        key = "alarms/key"
        url = str.format("{}/v1/kv/{}", self.baseurl, key)
        consul_config = {
            'consul_url': 'localhost',
            'consul_port': '8500',
            'alarms_folder': 'alarms'
        }
        consul_util = ConsulUtils(consul_config)
        with requests_mock.Mocker() as m:
            m.put(url, headers={}, status_code=500)
            with self.assertRaises(ConsulUtilsException) as raised_exp:
                consul_util.add_key_value(key, "test")

        self.assertEqual("Error while talking to consul.\n500", raised_exp.exception.message.strip())


class TestDeleteByKey(TestCase):
    def setUp(self):
        self.baseurl = "http://localhost:8500"

    def test_delete_by_key_success(self):
        key = "alarms/key"
        url = str.format("{}/v1/kv/{}", self.baseurl, key)
        with requests_mock.Mocker() as m:
            m.delete(url, status_code=500)

            consul_config = {
                'consul_url': 'localhost',
                'consul_port': '8500',
                'alarms_folder': 'alarms'
            }
            consul_util = ConsulUtils(consul_config)
            with self.assertRaises(ConsulUtilsException) as raised_exp:
                consul_util.delete_by_key(key)
            self.assertEqual("Error while talking to consul.\n500", raised_exp.exception.message.strip())

    def test_delete_by_key_exception(self):
        key = "alarms/key"
        url = str.format("{}/v1/kv/{}", self.baseurl, key)
        headers = {
            "x-consul-index": "0"
        }
        with requests_mock.Mocker() as m:
            m.delete(url, headers=headers, status_code=500)

            consul_config = {
                'consul_url': 'localhost',
                'consul_port': '8500',
                'alarms_folder': 'alarms'
            }
            consul_util = ConsulUtils(consul_config)
            with self.assertRaises(ConsulUtilsException) as raised_exp:
                consul_util.delete_by_key(key)
            self.assertEqual("Error while talking to consul.\n500", raised_exp.exception.message.strip())


if __name__ == '__main__':
    unittest.main()
