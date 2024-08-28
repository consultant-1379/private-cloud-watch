#!/bin/bash
#
# Set up the local host to point to ldap using the ldap.conf and nsswitch.conf
# that were already copied to this host.

# The package install will prompt you for the following things even if you tell
# it that you're doing a noninteractive install, so you have to pre-set them.

# Should debconf manage LDAP configuration? No, because they don't support the
# client TLS options, and we need those.
echo "ldap-auth-config        ldap-auth-config/override       boolean false" | debconf-set-selections

# The following lines would be a very convenient way to avoid being prompted by
# libpam-runtime at install time, but there's a bug in pam-auth-update that
# just overwrites these to the defaults no matter what you set these to, so
# you're forced to edit the files manually after installing the packages.
# (This has to be a bug: what's the point of using debconf at all if you just
# overwrite all the values in there before looking at them?)

#echo "libpam-runtime libpam-runtime/profiles multiselect unix, ldap, systemd, mkhomedir" | debconf-set-selections
#echo "libpam-runtime libpam-runtime/override boolean true" | debconf-set-selections

# This script assumes that the files /etc/ldap.conf and /etc/nsswitch.conf have
# already been copied over before this is run. In order to prevent apt from
# nuking those, there are some weird dpkg options specified below.

# Install ldap packages.
DEBIAN_FRONTEND='noninteractive' sudo apt -qy update || exit 1
DEBIAN_FRONTEND='noninteractive' sudo apt -qy install \
    -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' \
    libnss-ldap libpam-ldap nscd ldap-utils || exit 1

# Now edit the pam files by hand.
echo "session required	pam_mkhomedir.so" >>/etc/pam.d/common-session
sed -i 's/ use_authtok / /g' /etc/pam.d/common-password

# Restart nscd.
systemctl restart nscd || exit 1
