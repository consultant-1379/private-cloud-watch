#!/usr/bin/env python2.7
#
# This script uses the OpenStack API to find stacks and child resources
# that are in non-healthy states.
#
# If everything's in good shape, we don't output anything.
#
# Written by Loren Jan Wilson, May 2019.
#

import os

from keystoneauth1 import loading
from keystoneauth1 import session
from heatclient import client

# Disable the insecure request warning, because TLS isn't set up for this
# Openstack API on our local host.
from requests.packages.urllib3 import disable_warnings
disable_warnings()
# Wish this worked. It's supposed to, but it doesn't.
#from requests.packages.urllib3.exceptions import InsecureRequestWarning
#disable_warnings(InsecureRequestWarning)

# Here are the statuses we consider to be safe ones.
good_status_list = [
    'CREATE_COMPLETE',
    'UPDATE_COMPLETE',
    'CHECK_COMPLETE',
    'RESUME_COMPLETE'
]

def main():
    # Set up a client to the openstack heat API.
    heat = create_heat_client()
    # Get the list of all stacks we're allowed to see.
    stacks = heat.stacks.list()
    for stack in stacks:
        # For each stack, print error if it has a bad status.
        if stack.stack_status not in good_status_list:
            print_stack_error(stack)
        # Also, get the child resources of this stack, and
        # print an error if any of them aren't good.
        resources = heat.resources.list(stack.id)
        for resource in resources:
            if resource.resource_status not in good_status_list:
                print_resource_error(stack, resource)

def create_heat_client():
    ''' Create and return a Heat API client.
    '''
    # Pull these credentials from the local environment.
    # Not sure if this is going to work from a cronjob...
    # We might have to source $HOME/rc/keystone.sh before running.
    credentials = {
        "username": os.environ.get('OS_USERNAME'),
        "password": os.environ.get('OS_PASSWORD'),
        "auth_url": os.environ.get('OS_AUTH_URL'),
        "project_domain_id": os.environ.get('OS_PROJECT_DOMAIN_ID'),
        "project_domain_name": os.environ.get('OS_PROJECT_DOMAIN_NAME'),
        "project_id": os.environ.get('OS_PROJECT_ID'),
        "project_name": os.environ.get('OS_PROJECT_NAME'),
        "tenant_name": os.environ.get('OS_TENANT_NAME'),
        "tenant_id": os.environ.get("OS_TENANT_ID"),
        "user_domain_id": os.environ.get('OS_USER_DOMAIN_ID'),
        "user_domain_name": os.environ.get('OS_USER_DOMAIN_NAME')
    }

    # Create an openstack heat client and return it.
    loader = loading.get_plugin_loader('password')
    auth = loader.load_from_options(**credentials)
    sess = session.Session(auth=auth, verify=False)
    heat = client.Client('1', session=sess)
    return heat

def print_stack_error(stack):
    ''' Print a detailed error message for the given stack.
    '''
    error_message = '''BAD STACK:
    id: %s
    name: %s
    status: %s
    creation_time: %s
    updated_time: %s
    status_reason: %s'''

    error_args = (
        stack.id,
        stack.stack_name,
        stack.stack_status,
        stack.creation_time,
        stack.updated_time,
        stack.stack_status_reason
    )

    print(error_message % error_args)
    print

def print_resource_error(stack, resource):
    ''' Print a detailed error message for the given stack and resource.
    '''
    error_message = '''BAD RESOURCE:
    id: %s
    name: %s
    status: %s
    creation_time: %s
    updated_time: %s
    status_reason: %s
    parent_stack:
        id: %s
        name: %s
        status: %s'''

    error_args = (
        resource.physical_resource_id,
        resource.resource_name,
        resource.resource_status,
        resource.creation_time,
        resource.updated_time,
        resource.resource_status_reason,
        stack.id,
        stack.stack_name,
        stack.stack_status
    )

    print(error_message % error_args)
    print

if __name__ == "__main__":
    main()

