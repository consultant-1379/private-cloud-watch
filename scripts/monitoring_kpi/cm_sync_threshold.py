#!/usr/bin/env python2.7
# ********************************************************************
# Ericsson LMI               Utility Script
# ********************************************************************
#
# (c) Ericsson LMI 2019 - All rights reserved.
#
# The copyright to the computer program(s) herein is the property of Ericsson LMI.
# The programs may be used and/or copied only with the written permission from Ericsson LMI or
# in accordance with the terms and conditions stipulated in the agreement/contract under which
# the program(s) have been supplied.
#
# ********************************************************************
# Name    : Colin Bennett
# Purpose : Customized alarm to monitor CM sync when KPI breached
# Team    : AetosDios
# ********************************************************************
import argparse
import logging
import subprocess
import enmscripting as enm

from requests import ConnectionError

LOGGER = logging.getLogger(__name__)
LOGGER.setLevel(logging.INFO)


def configure_logger():
    """
    Configures logging for this script.
    """
    console = logging.StreamHandler()
    console.setLevel(logging.INFO)
    # Create formatter and add it to the handlers
    formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s- %(message)s')
    console.setFormatter(formatter)
    # Add the handlers to the logger
    LOGGER.addHandler(console)


def execute_enm_alarm(severity, threshold):
    """
    Execute command to send alarm to ENM.

    :param severity: The ENM alarm severity
    :param message: The ENM alarm message
    """
    cmd = "fm-alarm-lib -r ALARM -s {} -p 'CM SYNC Alarm: Synchronized Nodes below KPI threshold {}%' \
        -c 'CM Sync Status' -e 'CM KPI' -m 'ManagementSystem'".format(severity, threshold)
    try:
        LOGGER.info("Executing command to call 'fm-alarm-lib'")
        proc = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

        stdout, stderr = proc.communicate(cmd)

        if proc.returncode == 0:
            stdout = stdout.strip()
            LOGGER.info("Stdout:  %s", stdout)
            LOGGER.info("Local command execution succeeded.")

        else:
            LOGGER.error(
                "Local command execution failed. Error : %s", stderr.strip())

    except (OSError, ValueError) as err:
        LOGGER.error(
            "Local command execution failed. Error : %s", str(err))


def run_enm_command(session, command):
    """
    Function to run a given ENM command.

    :param session: Session object used to create a command instance to
        execute the CLI commands.
    :param command: The CLI command to be executed.

    :return: The output from the command(s). Lines are single strings.
        Table rows are single strings with tabs delimiting columns.
    """
    cmd = session.command()
    response = cmd.execute(command)

    command_output = response.get_output()
    return command_output


def percentage(part, whole):
    """
    Function that calculates percentage.
    """
    return 100 * int(part)/int(whole)


def check_cm_sync_status(session, threshold):
    """
    Function to checks the CM Sync of nodes against given KPI threshold.

    :param session: Session object used to create a command instance to
        execute the CLI commands.
    :param threshold: KPI Unsynchronized Threshold as a percentage.

    :raises: RuntimeError
    """
    LOGGER.info("Checking CM Synchronized Nodes.")
    cm_succ_str = " instance(s)"

    cm_sync_cmd = ('cmedit get * CmFunction.syncStatus==SYNCHRONIZED')
    out = run_enm_command(session, cm_sync_cmd)
    if not any(cm_succ_str in str(line) for line in out):
        raise RuntimeError("Command '{}' was not successful. "
                           "Output={}".format(cm_sync_cmd, out))

    synchronized = out[-1].value().strip(cm_succ_str)

    cm_unsync_cmd = ('cmedit get * CmFunction.syncStatus==UNSYNCHRONIZED')
    out = run_enm_command(session, cm_unsync_cmd)
    if not any(cm_succ_str in str(line) for line in out):
        raise RuntimeError("Command '{}' was not successful. "
                           "Output={}".format(cm_unsync_cmd, out))

    unsynchronized = out[-1].value().strip(cm_succ_str)
    total = int(synchronized) + int(unsynchronized)
    LOGGER.info("Total nodes: '%s' \nSynchronized nodes: '%s' \nUnsynchronized nodes: '%s'",
                total, synchronized, unsynchronized)

    if total == 0:
        LOGGER.info("Total CM Nodes is 0.")
        return

    unsync_node = percentage(unsynchronized, total)
    LOGGER.info("Percent of Unsynchronized nodes: '%s'", unsync_node)
    if unsync_node >= threshold:
        LOGGER.info("Sending 'WARNING' alarm to ENM.")
        execute_enm_alarm('WARNING', threshold)
    else:
        LOGGER.info("Sending 'CLEARED' alarm to ENM.")
        execute_enm_alarm('CLEARED', threshold)


def main():
    """
    Entry point for starting CM Sync application.
    """
    configure_logger()
    LOGGER.info("Starting...")

    parser = argparse.ArgumentParser()
    parser.add_argument('-t', '--threshold', default=20,
                        type=int,
                        help="KPI Unsynchronized Threshold as a percentage.")
    parser.add_argument('-l', '--launcher', metavar='<Tenancy httpd_fqdn Url>',
                        required=True,
                        help="Tenancy httpd_fqdn  .")
    parser.add_argument('-u', '--user', metavar='<Tenancy GUI Admin username>',
                        required=True,
                        help="Tenancy GUI Admin username.")
    parser.add_argument('-p', '--passwd',
                        metavar='<Tenancy GUI Admin password>', required=True,
                        help="Tenancy GUI Admin password.")

    args = parser.parse_args()

    # Create an ENM session
    session = None
    try:
        enm_url = 'https://' + args.launcher
        session = enm.open(enm_url, args.user, args.passwd)

        check_cm_sync_status(session, args.threshold)

    except (RuntimeError, ConnectionError, ValueError) as err:
        LOGGER.error("ENM Session Failed. Error: %s", err)
    finally:
        # Terminate the ENM terminal session
        if session is not None:
            enm.close(session)

        LOGGER.info("Completed script.")


if __name__ == '__main__':
    main()
