OPTIONAL_ARGS = ['additional_attributes']

MANDATORY_ARGS = ['config_file', 'event_type', 'managed_object_instance', 'probable_cause', 'specific_problem',
                  'record_type', 'severity']

MANDATORY_CONFIG = {
    "enm": ["haproxyIntIp", "haproxyPort"],
    "consul": ["url", "port", "folder"],
    "alarm": ["alarmThreshold"]
}

ENM_FM_PATH = "/internal-alarm-service/internalalarm/internalalarmservice/"
