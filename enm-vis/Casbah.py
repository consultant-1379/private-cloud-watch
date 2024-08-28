#!/usr/bin/env python3

import sys
import os
import re
from getpass import getpass

import pexpect

class CasbahException(Exception):
	def __init__(self, msg):
		self.msg = msg
	def __str__(self):
		return self.msg


class Casbah(object):

	def __init__(self, pwdfile=None, ecgw=None, casgw=None, timeout=45, signum=None, otpw=None, verbose=False):
		self.pwdfile = pwdfile or os.environ.get('PWDFILE')
		try:
			with open(self.pwdfile, 'r') as fp:
				self.pwdict = dict([ l.rstrip('\n').split('\t', 1) for l in fp])
		except TypeError:
			raise CasbahException('missing pwdfile parameter')
		except FileNotFoundError:
			raise CasbahException('pwdfile not found')
		except PermissionError:
			raise CasbahException('cannot read pwdfile')
		except ValueError:
			raise CasbahException('cannot parse pwdfile')
		self.ecgw = ecgw or os.environ.get('ECGW', 'eusecgw')
		self.casgw = casgw or os.environ.get('CASGW', 'at1-nmaas1-cas1')
		self.timeout = timeout
		self.verbose = verbose
		if signum is None:
			self.proc = pexpect.spawn('ssh-client %s' % self.ecgw)
		else:
			if otpw is None:
				otpw = getpass('Enter /// OTP: ')
			if re.match(r'[0-9]{6}$', otpw) is None:
				raise CasbahException('flakey-looking OTP')
			self.proc = pexpect.spawn('ssh %s@%s' % (signum, self.ecgw))
			self.exch('Ericsson OTP Password:', otpw)
		prompt = r'%s(\.[a-z0-9]+)* > ' % self.ecgw
		self.exch(prompt, 'date')
		self.exch(prompt, 'ssh %s' % self.casgw)
		prompt = '%s > ' % self.casgw
		self.exch(prompt, 'date')
		self.exch(prompt)

	@property
	def before(self):
		return self.proc.before

	@property
	def after(self):
		return self.proc.after

	def pwd(self, id, default=None):
		return self.pwdict.get(id, default)

	def exch(self, expect=None, cmd=None, verbose=False):
		if expect is not None:
			self.proc.expect(expect, timeout=self.timeout)
			if verbose or self.verbose:
				for b in (self.before, self.after):
					print(str(b, encoding='utf-8'), end='')
		if cmd is not None:
			self.proc.sendline(cmd)

	def interact(self):
		self.proc.interact()

	def close(self, force=True):
		self.proc.close(force)


def main():
	c = Casbah(signum=os.environ.get('SIGNUM'), verbose=True)
	c.interact()


if __name__ == '__main__':
	try:
		main()
	except CasbahException as e:
		print(e)
		sys.exit(127)
