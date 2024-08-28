#!/usr/bin/python

# produce a json file mapping compute nodes to servers.
# run on genie-utility as user admin.

import sys
import os
import json

import urllib3
from keystoneauth1 import loading, session
from novaclient import client

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

def main():
	nova = NovaClient()

	hlist = [ h.hypervisor_hostname for h in nova.hypervisors.list() \
		if h.running_vms > 0 ]

	hdict = dict()
	opts = { 'all_tenants': 'yes' }
	for h in hlist:
		opts['host'] = h
		hdict[h[7:17]] = [ s.name for s in nova.servers.list(search_opts=opts) ]

	json.dump(hdict, sys.stdout)


def getKeystone(rcfile=None):
	uninvited = ('IDENTITY_API_VERSION', 'INTERFACE', 'REGION_NAME', )
	if rcfile is None:
		rcfile = os.path.expanduser('~/rc/keystone.sh')
	with open(rcfile, 'r') as fp:
		kv = [ l[10:].strip().split('=', 1) \
			for l in fp if l.startswith('export OS_') ]
	return dict( (k.lower(), v[1:-1] if v.startswith(('"', "'")) else v) \
		for k,v in kv if k not in uninvited )

def NovaClient(rcfile=None):
	loader = loading.get_plugin_loader('password')
	auth = loader.load_from_options(**getKeystone(rcfile))
#	verify='/some/chosen/cert/bundle.pem'
	sess = session.Session(auth=auth, verify=False)
	return client.Client('2', session=sess)

if __name__ == '__main__':
	main()
