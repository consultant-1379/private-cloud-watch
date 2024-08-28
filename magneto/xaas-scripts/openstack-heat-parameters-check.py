#!/usr/bin/env python2.7
#
# This script is meant to be run inside each tenant account after grabbing the
# most recent SED file from the LCM services node at /vnflcm-ext/enm/sed.json
#
# It performs the following tasks:
#
# 0. Pull in the info for each stack from the OpenStack API.
#
# 1. Compare the list of parameters for each stack to the list in the template,
# complaining if any parameters in the template are missing from the stack.
#
# 2. Find parameter values that don't match the SED values, and complain.
#
# 3. Check for some known malformed tags and parameters conditions that we've
# seen so far.
#
# If everything's in good shape, we don't output anything.
#
# Written by Loren Jan Wilson, June 2019.
#

import os
import argparse
import json
import yaml
import ast

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

# SED file and templates directory get passed in at the command line.
parser = argparse.ArgumentParser()
parser.add_argument('sed_file', help='sed file')
parser.add_argument('templates', help='templates dir')
args = parser.parse_args()

def main():
    # Set up a client to the openstack heat API.
    global heat_client
    heat_client = create_heat_client()

    # Get the list of all stacks we're allowed to see.
    stacks = heat_client.stacks.list()

    # Load information from all templates to get a list of parameters that
    # should exist.
    global template_parameters
    template_parameters = load_template_parameters()

    # Load in the SED file that was passed in at the command line.
    global sed_defaults
    with open(args.sed_file, "r") as sed_file:
        sed = json.load(sed_file)
    sed_defaults = sed['parameter_defaults']

    # Run checks for each stack, one at a time.
    for stack in stacks:
        verify_stack(stack)

def verify_stack(stack):
    """ Set some globals for this stack, skip the stacks that won't have
    parameters, then run the two check functions against this stack.
    Sorry about the use of globals. If I come back to this again in the
    future, I'll create a class.
    """
    # Set the attributes that we'll print in our errors.
    # This makes it easier to print the errors.
    global stack_name
    global stack_uuid
    global creation_time
    stack_name = stack.stack_name
    stack_uuid = stack.id
    creation_time = stack.creation_time

    # Parse the stack name into a deployment id and a service name. This might
    # be error-prone if the stacks aren't named the way we expect, and I know
    # that our naming isn't consistent, so expect fireworks here.
    global deployment_id
    global service_name
    stack_name_fields = stack_name.split('_')
    deployment_id = stack_name_fields[0]
    service_name = stack_name_fields[1]

    # Skip vnflcm and security group stacks.
    stacks_to_skip = ['vnflcm', 'security_group', 'laf_db_volume', 'vnf_laf', 'network_internal_stack', '_cu_key']
    if any(s in stack_name.lower() for s in stacks_to_skip):
        return

    # Run the two groups of checks.
    check_stack_parameters(stack)
    check_stack_tags(stack)

def check_stack_parameters(stack):
    ''' Check the stack's parameters against the SED values and the template
    defaults. Right now, we just make sure that everything in the template is
    represented, but we compare directly against the SED and expect every
    parameter to be represented there.
    '''

    # I used to try to pull the stack parameters from the "environment". That
    # wasn't correct. However, the "right thing to do" here is ridiculous, so
    # you can't really blame me. The issue turns out to be that the stack
    # objects you get from a "heat.stacks.list" are NOT the same as what you
    # get from a "heat.stacks.get"...the former don't contain the parameters,
    # and the latter do. So if we need the parameters, we have to do a stack
    # list, and then a get for each stack.
    try:
        stack_parameters = heat_client.stacks.get(stack_id=stack.id).parameters
    except:
        print_error("couldn't run get method on stack %s" % stack.id)
        return

    # Check the stack's parameters to make sure all the ones that are
    # supposed to be there are really there.
    if service_name in template_parameters:
        for key in template_parameters[service_name]:
            if key not in stack_parameters:
                print_error("mandatory parameter '%s' is missing" % key)

    # Check each parameter against the defaults in the SED.
    for key, value in stack_parameters.iteritems():
        # Intensely stupid Python client...
        # It sometimes gives you lists translated into their text
        # representations.
        try:
            real_value = ast.literal_eval(value)
        except:
            real_value = value

        # Convert the parameter to something consistent we can compare.
        converted_value = convert_parameter(real_value)
        # Run the comparison.
        if key in sed_defaults:
            converted_default = convert_parameter(sed_defaults[key])
            if not parameters_are_equal(converted_value, converted_default):
                print_error("parameter '%s' set to '%s' but SED is set to '%s'" % (key, converted_value, converted_default))
        else:
            # There wasn't a default... so set our default to the first
            # thing we see. There would obviously be better ways to do
            # this, but if the first one we see happens to be abnormal,
            # we'll print an error for every other stack, which still
            # alerts us to a problem.
            keys_to_skip = ['tags', 'ha_policy', 'OS::stack_id', 'OS::stack_name', 'service_name']
            if key in keys_to_skip:
                continue
            sed_defaults[key] = converted_value

def check_stack_tags(stack):
    ''' Check to make sure the values of the stack's tags are correct.
    Examples from the current ENM installations:
    stack_name: ntwlsenm01_pmserv
    tags: '{u''enm_deployment_id'': u''ntwlsenm01'', u''enm_stack_name'': u''pmserv''}
    '''

    # Check for empty or unset tags.
    if (stack.tags is None) or (len(stack.tags) == 0):
        print_error("no tags defined")
        return

    # If we got here, there is some kind of tag data defined.
    # Let's check to make sure the value for each tag is correct.
    # Complain if we don't end up seeing both of these tags.
    deployment_id_tag_exists = False
    service_name_tag_exists = False

    for tag in stack.tags:
        # If we can't split the tag, error and skip it.
        try:
            tag_key, tag_value = tag.split('=', 1)
        except:
            print_error("malformed tag %s" % tag)
            continue

        # No value? Error and skip it.
        if len(tag_value) == 0:
            print_error("missing value for tag %s" % tag)
            continue

        # Decide what to compare against, or give up if we can't.
        if tag_key == 'enm_deployment_id':
            deployment_id_tag_exists = True
            value_to_compare = deployment_id
        elif tag_key == 'enm_stack_name':
            service_name_tag_exists = True
            value_to_compare = service_name
        else:
            # They may add more tags in future versions of ENM, so this might
            # not always be an error.
            print_error("unknown tag %s" % key)
            continue

        # Run the comparison.
        if tag_value != value_to_compare:
            print_error("tag %s is %s but should be %s" % (tag_key, tag_value, value_to_compare))

    # Complain if we didn't see the two tag types we want.
    if not deployment_id_tag_exists:
        print_error("missing enm_deployment_id tag")
    if not service_name_tag_exists:
        print_error("missing enm_stack_name tag")

def load_template_parameters():
    ''' Load parameters from the templates. We'll use these to make sure that
    each stack has all the parameters defined in its template.
    '''

    parameters = {}
    # Look at the files in the templates directory.
    # Safe to ignore the full definitions.
    _, _, filenames = os.walk(args.templates).next()
    templates = [f for f in filenames if "definition" not in f]

    # Get template name, load the file, and set values in parameters.
    for f in templates:
        template_name = f.split('.')[0]
        parameters[template_name] = set()
        with open(os.path.join(args.templates, f), 'r') as stream:
            contents = yaml.safe_load(stream)
        for param, _ in contents['parameters'].iteritems():
            parameters[template_name].add(param)

    return parameters

def convert_parameter(p):
    ''' The results from the Python client and the SED file and templates are
    in various inconsistent forms. So this function attempts to make the inputs
    consistent.
    '''
    # If it's a one-element list, return the first element.
    # If it's a multi-element list, return the list.
    if isinstance(p, list):
        if len(p) == 1:
            return p[0]
        return p
    # This is in a "try" because it won't work if the type isn't string-like.
    try:
        # If it's text fake-boolean, make it a real boolean.
        if p.lower() in ["true", "false"]:
            return p.lower() == "true"
        # If it's comma-separated, split it into a list.
        if ',' in p:
            return p.split(',')
        # If it's all digits, return an int.
        if p.isdigit():
            return int(p)
    except:
        pass
    # If nothing above was applicable, just return.
    return p

def parameters_are_equal(p1, p2):
    ''' This exists in order to compare lists in a fair unordered fashion.
    '''
    if isinstance(p1, list):
        return set(p1) == set(p2)
    else:
        return p1 == p2

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

def print_error(message):
    # CSV format for now.
    print("%s,%s,%s,%s" % (stack_name, stack_uuid, creation_time, message))

if __name__ == "__main__":
    main()

