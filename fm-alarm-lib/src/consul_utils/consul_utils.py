import time

import consul
from consul import ConsulException
from requests import RequestException

from common.exceptions import ConsulUtilsException
from common.utils import get_logger

SCRIPT_NAME = __name__
LOG = get_logger(SCRIPT_NAME)


class ConsulUtils:
    """Class for consul helper methods."""

    def __init__(self, consul_config):
        self.config = consul_config
        self.consul = consul.Consul(host=self.config['consul_url'], port=self.config['consul_port'], scheme='http').kv

    def get_value_by_hash(self, key_hash):
        """
        Get kv value by alarm hash.
        :param key_hash: calculated alarm hash.
        :return: a json mapping of key value , and attributes.
        """
        key = str.format("{}/{}", self.config['alarms_folder'], key_hash)
        return self.get_value_by_key(key)

    def get_value_by_key(self, key):
        """
        Get kv value by key.
        :param key: kv key.
        :return: a json mapping of key value , and attributes.
        :raises: ConsulException if there is a problem communicating to consul.
        """
        try:
            return self.consul.get(key)
        except (RequestException, ConsulException) as exp:
            raise ConsulUtilsException(str.format("Error while talking to consul.\n{}", str(exp)))

    def add_key_value(self, key, value):
        """
        Add a new kv record.
        :param key: Key for the record.
        :param value: value for the record.
        :raises: ConsulException if there is a problem communicating to consul.
        """
        try:
            return self.consul.put(key, value)
        except (RequestException, ConsulException) as exp:
            raise ConsulUtilsException(str.format("Error while talking to consul.\n{}", str(exp)))

    def add_key_and_currtime(self, key):
        """
        Add a new kv record with the supplied key and the current time in epoch seconds are the value
        :param key: key for the record.
        """
        key = str.format("{}/{}", self.config['alarms_folder'], key)
        self.add_key_value(key, str(time.time()))

    def does_hash_exist(self, key):
        """
        check if a kv record exists in consul.
        :param key: Key to check.
        :return: Boolean value
        """
        value = self.get_value_by_hash(key)
        if value[1] is None:
            return False
        return True

    def delete_by_key(self, key):
        """
        Delete a kv record by key name.
        :param key: Key name to delete.
        :return: ConsulException if there is a problem communicating to consul.
        """
        try:
            return self.consul.delete(key)
        except (RequestException, ConsulException) as exp:
            raise ConsulUtilsException(str.format("Error while talking to consul.\n{}", str(exp)))

    def delete_by_hash(self, key_hash):
        """
        Delete a kv record by calculated alarm hash
        :param key_hash: hash to delete.
        """
        key = str.format("{}/{}", self.config['alarms_folder'], key_hash)
        self.delete_by_key(key)
