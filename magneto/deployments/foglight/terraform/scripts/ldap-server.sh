#!/bin/bash
#
# Run an openldap server in a docker container. If we can find ldap database
# backups in S3, restore the most recent one.
#
# This uses a docker image which automatically generates backups of the ldap
# databases. It also creates a cronjob to copy those databases to a locally
# configured S3.

service_name="ldap-server"
ldap_dir="/srv/slapd"
database_dir="$ldap_dir/database"
config_dir="$ldap_dir/config"
backup_dir="$ldap_dir/backup"
certs_dir="$ldap_dir/certs"

function make_dirs () {
    echo "Making ldap directories at $ldap_dir"
    for i in "$database_dir" "$config_dir" "$backup_dir" "$certs_dir" ; do
        mkdir -p "$i" || return $?
    done
}

function copy_certs () {
    echo "Copying certs to $certs_dir"
    # OpenLDAP expects that the host cert doesn't include the intermediate ca,
    # and the ca cert includes all possible CAs that we will trust for clients.
    # This is the opposite of what Apache expects, and it makes less sense,
    # because it means all client configs have to change whenever you add or
    # change an intermediate CA.
    cp "/etc/ssl/private/host.crt" $certs_dir
    cp "/etc/ssl/private/host.key" $certs_dir
    cp "/etc/ssl/private/erixzone-ca-chain.crt" $certs_dir
}

function start_ldap () {
    echo "Starting ldap container"

    docker run --name "$service_name" \
        --detach --net=host --restart=always \
        --mount type=bind,source=$database_dir,target=/var/lib/ldap \
        --mount type=bind,source=$config_dir,target=/etc/ldap/slapd.d \
        --mount type=bind,source=$backup_dir,target=/data/backup \
        --mount type=bind,source=$certs_dir,target=/container/service/slapd/assets/certs \
        --env LDAP_TLS_CRT_FILENAME=host.crt \
        --env LDAP_TLS_KEY_FILENAME=host.key \
        --env LDAP_TLS_CA_CRT_FILENAME=erixzone-ca-chain.crt \
        --env LDAP_BACKUP_CONFIG_CRON_EXP="0 */4 * * *" \
        --env LDAP_BACKUP_DATA_CRON_EXP="0 */4 * * *" \
        --env LDAP_BACKUP_TTL="90" \
        osixia/openldap-backup \
        || return $?
    container=""
    while [ -z "$container" ] ; do
        container=$(docker ps -qf name=$service_name)
    done

    echo "Waiting for ldap to come up"
    up=""
    while [ -z "$up" ] ; do
        up=$(docker exec "$container" \
            ldapsearch -x -H ldap://localhost -b dc=example,dc=org \
            -D "cn=admin,dc=example,dc=org" -w admin | grep Success)
        sleep 1
    done
    echo "Ldap service is up"
    return 0
}

function insistent_run () {
    ret="1"
    while [ "$ret" -ne "0" ] ; do
        "$@"
        ret=$?
        sleep 1
    done
}

function restore_from_backup () {
    # This container provides "restore" commands, but they will fail every
    # time, because they don't delete what's currently present in the ldap
    # databases before trying to restore. Further proof that docker containers
    # made by randos are not to be trusted.

    # Note: some kind of prep work happens in this container for the first 40
    # seconds or so, so some of the commands need to be repeatedly run until
    # they succeed.

    echo "Pulling most recent $service_name config backup from spaces"
    # Sort by creation time, then take the most recent one.
    filename=$(s3cmd ls s3://$service_name/ | \
        grep -v '\/$' | grep config | sort | tail -n1 | \
        awk '{print $4}' | sed 's/.*\///g') # If we got one, load it in.
    if [ ! -z "$filename" ]; then
        s3cmd get "s3://$service_name/$filename" || return $?
        mv "$filename" "$backup_dir" || return $?

        echo "Deleting current ldap config"
        insistent_run docker exec $container sv stop /container/run/process/slapd
        rm -rf "${config_dir}"/*
        ls -l "${config_dir}"

        echo "Loading config backup into ldap"
        insistent_run docker exec $container slapd-restore-config $filename
    fi

    echo "Pulling most recent $service_name data backup from spaces"
    # Sort by creation time, then take the most recent one.
    filename=$(s3cmd ls s3://$service_name/ | \
        grep -v '\/$' | grep data | sort | tail -n1 | \
        awk '{print $4}' | sed 's/.*\///g') # If we got one, load it in.
    if [ ! -z "$filename" ]; then
        s3cmd get "s3://$service_name/$filename" || return $?
        mv "$filename" "$backup_dir" || return $?

        echo "Deleting current ldap data"
        insistent_run docker exec $container sv stop /container/run/process/slapd
        rm -rf "${database_dir}"/*
        ls -l "${database_dir}"

        echo "Loading data backup into ldap"
        insistent_run docker exec $container slapd-restore-data $filename
    fi

    return 0
}

function make_backup_cronjob () {
    echo "Creating cronjob that copies ldap backups to s3"
    job_name="ldap-backup-sync"
    chmod 700 /usr/local/bin/$job_name.sh
    cat << EOF > /etc/cron.d/$job_name
5 */4 * * * root /usr/local/bin/$job_name.sh $service_name >/var/log/$job_name.out 2>&1
EOF
}

function main () {
    make_dirs || exit 1
    copy_certs || exit 1
    start_ldap || exit 1
    restore_from_backup || exit 1
    make_backup_cronjob || exit 1
    echo "Done!"
    exit 0
}

main
