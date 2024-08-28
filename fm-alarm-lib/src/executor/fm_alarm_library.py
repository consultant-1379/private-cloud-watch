import json
import sys

from common.constants import Severity, RecordType, ENM_FM_PATH
from common.exceptions import ValidationException, ConfigValidationException, EnmAlarmException, ConsulUtilsException, \
    ConfigParsingException
from common.utils import get_logger, parse_args, parse_config
from common.validators import validate_args, validate_config_sections_and_keys
from fm_alarm_lib.fmalarm import FmAlarm

SCRIPT_NAME = __name__
LOG = get_logger(SCRIPT_NAME)


def main():
    try:
        args = parse_args()
        LOG.info(str.format("Using config file at {}.", args.config_file))
        validate_args(args)
        config = parse_config(args.config_file)
        validate_config_sections_and_keys(config)

        fm_url = str.format("http://{}:{}{}", config['enm']['haproxyIntIp'], config['enm']['haproxyPort'], ENM_FM_PATH)
        consul_config = {
            'consul_url': config['consul']['url'],
            'consul_port': config['consul']['port'],
            'alarms_folder': config['consul']['folder']
        }
        additional_attributes = json.loads(args.additional_attributes)
        alarm = FmAlarm(additional_attributes=additional_attributes,
                        specific_problem=args.specific_problem,
                        probable_cause=args.probable_cause,
                        event_type=args.event_type,
                        managed_object_instance=args.managed_object_instance,
                        perceived_severity=Severity[args.severity],
                        record_type=RecordType[args.record_type],
                        fm_url=fm_url,
                        threshold=config['alarm']['alarmThreshold'],
                        consul_config=consul_config
                        )
        alarm.send_alarm_request()

    except ValidationException as exp:
        LOG.error(exp.__str__())
        LOG.error(" Arg validation process has errors. Exiting application.")
    except ConfigValidationException as exp:
        LOG.error(exp.__str__())
        LOG.error("Config validation process has errors. Exiting application.")
    except EnmAlarmException as exp:
        LOG.error(exp.__str__())
        LOG.error("Sending alarms has errors. Exiting application.")
    except ConsulUtilsException as exp:
        LOG.error(exp.__str__())
        LOG.error("Consul utils has errors. Exiting application.")
    except ConfigParsingException as exp:
        LOG.error(exp.__str__())
        LOG.error("Error parsing config file. Exiting application.")


if __name__ == '__main__':
    LOG.debug("Starting...")
    sys.exit(main())
