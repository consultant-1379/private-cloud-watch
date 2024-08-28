#!/usr/bin/python3.6

"""
Helper functions for all enmscout programs.
"""

import os
import configparser
import logging
import contextlib
import fcntl

config_file = '/opt/enmscout/etc/config.ini'

def load_config(stanza_name):
    """ Load the common config file and return the requested stanza.
    """
    if not os.path.isfile(config_file):
        raise Exception(f"config file doesn't exist: {config_file}")

    config = configparser.ConfigParser(interpolation=configparser.ExtendedInterpolation())
    config.read(config_file)
    return config[stanza_name]

# Load the config using the above function to get the global lock and log
# directories.
conf = load_config('enmscout')
log_dir = conf['log_dir']
lock_dir = conf['lock_dir']

def make_parent_dir(file_to_write):
    """ Make the parent directory for the given file.
    """
    dest_dir = os.path.dirname(file_to_write)
    try:
        os.makedirs(dest_dir)
    except FileExistsError:
        pass

def configure_logging(app_name, log_to_file=True, debug=False):
    """ Set up general logging. We'll log to console and also to the given
    file, if desired
    """
    log = logging.getLogger(app_name)
    log.setLevel(logging.DEBUG)
    formatter = logging.Formatter('%(asctime)s %(levelname)s %(message)s')

    # Log to console.
    ch = logging.StreamHandler()
    if debug:
        ch.setLevel(logging.DEBUG)
    else:
        ch.setLevel(logging.WARNING)
    ch.setFormatter(formatter)
    log.addHandler(ch)

    # Log to file as well, if desired.
    if log_to_file:
        log_file = os.path.join(log_dir, f"{app_name}.log")
        make_parent_dir(log_file)
        fh = logging.FileHandler(log_file)
        if debug:
            fh.setLevel(logging.DEBUG)
        else:
            fh.setLevel(logging.INFO)
        fh.setFormatter(formatter)
        log.addHandler(fh)

    return log

@contextlib.contextmanager
def interprocess_lock(lock_name):
    """ This is a locking mechanism which uses POSIX locking on a lock file
    specified in the config. It tries only once (non-blocking).

    Note: If the lock file is deleted at any time, you might hit a race
    condition which can cause multiple processes to run at once, both thinking
    they have the lock; in this case, one of them would have opened the file
    before it was deleted, and then later acquired a lock on the unlinked file
    descriptor. So put the lock file somewhere like /run/lock/ where people
    aren't as likely to mess around.

    Thanks to Tom Killian for helping me understand this race condition.

    This is meant to be used in a "with" context block, hence it's a
    @contextlib.contextmanager decorator. The lock gets closed at the end of
    the "with" by the finally clause.
    """
    lock_file = os.path.join(lock_dir, f"{lock_name}.lock")
    make_parent_dir(lock_file)
    a_lock = open(lock_file, 'a')
    try:
        fcntl.lockf(a_lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except IOError as exc:
        raise Exception("An instance is already running") from exc
    else:
        yield a_lock
    finally:
        a_lock.close()

