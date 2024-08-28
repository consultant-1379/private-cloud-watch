# pubkeydb

Simple public key lookups for grpcsig implementation using a boltdb database.

Usage of ./pubkeydb:
	  -a	Bool: Append to existing output file
	  -i string
	    	File input JSON (default "pubkeys.json")
	  -j	Bool: Write all BoltDB JSON Values, quit
	  -k	Bool: Dump key: as a prefix to json
	  -o string
	    	File output BoltDB (default "pubkeys.db")
	  -t	Bool: json io type true = use endpoints, false = use pubkeys
	  -v	Bool: Display Version, quit

Uses a simple json file of public key information to make a BoltDB database file for
fingerprint/public key lookups.  A second "EndPoints" bucket provides for adding endpoints,
switch with =t=T 

Not meant to replace a real LDAP or directory service. This is simply a debug dumper/ and test builder

### Try it out with the test keys

	./maketestjson.sh > pubkeys.json

	cat pubkeys.json
	{"service":"phlogiston","name":"bobo","keyid":"/phlogiston/bobo/keys/2c:f3:65:41:56:97:6c:2d:aa:08:ef:34:f3:ef:e3:c8","pubkey":"ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBABkFVCEb50dR3fFdq6n6gvHpa+s1iYB9tDMX38KIHaE/HEi3eK6zD9ND+E+PfkXVUkeieNBytDuh0wGycoZb/smVQBZcBy+jZgmkFn3snnSiMyQZOdRTPvXx5f4JR5aSmr4e1UOGIumNEHU/qkV/EwA7AM+ex5RJRIVK6y+l2Jb+pXhwA== test-ecdsa-key","stateadded":-1}
	{"service":"jettison","name":"bobo","keyid":"/jettison/bobo/keys/2c:f3:65:41:56:97:6c:2d:aa:08:ef:34:f3:ef:e3:c8","pubkey":"ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBABkFVCEb50dR3fFdq6n6gvHpa+s1iYB9tDMX38KIHaE/HEi3eK6zD9ND+E+PfkXVUkeieNBytDuh0wGycoZb/smVQBZcBy+jZgmkFn3snnSiMyQZOdRTPvXx5f4JR5aSmr4e1UOGIumNEHU/qkV/EwA7AM+ex5RJRIVK6y+l2Jb+pXhwA== test-ecdsa-key","stateadded":-1}
	{"service":"jettison","name":"bobo","keyid":"/jettison/bobo/keys/6f:d4:de:43:e2:6c:de:7c:44:2d:33:4a:d1:35:ab:8a","pubkey":"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINyH0WROU2WyByo3Mq0yUYaI1uaQvTLAL1OLMVLOytQ2 test-ed25519-key","stateadded":-1}
	{"service":"phlogiston","name":"maude","keyid":"/phlogiston/maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22","pubkey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa6Jdf0FADJs6t5xW6RZXGN7mN0bWJYbIKqUrLVtFfZX06WLiynChWmV7QvO4n2Ae08skMDHRaONd1nTcJonh8HMe75OTfQS/4xeCFKIt18BqwT85T7i8vnZvc6pDqLSLgQcFWMqo/51PQomZGdID+mlXK5oZnfsAabwZGcY+tbWUdnI3yzGo2XgBckQCu+nGtqCYyjlNayFs3AQ6tIhMdHmOeg9cyoqvVIb3wQW0wDgxf8rhcGoGO3Tiy4N9BCzdy9NoZd70Uq7jNkmWRS6Zg0IRW4HgIZ63mfM8Ai5HtJIoBDXdUi98OA1DXsV999wLF8JZQ169DfJJMy0bLQbhZ test-rsa-key","stateadded":-1}
	{"service":"jettison","name":"maude","keyid":"/jettison/maude/keys/e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22","pubkey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa6Jdf0FADJs6t5xW6RZXGN7mN0bWJYbIKqUrLVtFfZX06WLiynChWmV7QvO4n2Ae08skMDHRaONd1nTcJonh8HMe75OTfQS/4xeCFKIt18BqwT85T7i8vnZvc6pDqLSLgQcFWMqo/51PQomZGdID+mlXK5oZnfsAabwZGcY+tbWUdnI3yzGo2XgBckQCu+nGtqCYyjlNayFs3AQ6tIhMdHmOeg9cyoqvVIb3wQW0wDgxf8rhcGoGO3Tiy4N9BCzdy9NoZd70Uq7jNkmWRS6Zg0IRW4HgIZ63mfM8Ai5HtJIoBDXdUi98OA1DXsV999wLF8JZQ169DfJJMy0bLQbhZ test-rsa-key","stateadded":-1}

	go run pubkeydb.go
	Loading pubkeys.db from pubkeys.json
	Finished

### Single User Only - Add yourself to the (empty) public key database

Three shell scripts are provided to emit 1 line of json to stdout, containing
the current users - or specified users public key information:

	ecdsapubkeytojson.sh   [public key file (in OpenSSH 1 line format)] [username]-optional
	ed25519pubkeytojson.sh [public key file (in OpenSSH 1 line format)] [username]-optional
	rsapubkeytojson.sh     [public key file (in OpenSSH 1 line format)] [username]-optional

The shell scripts `*pubkeytojson.sh`, which will emit 1 line of json to `stdout`, containig
the current users' public key information, and username from `whoami`

	$ rsakeytojson.sh ~/.ssh/id_rsa.pub > pubkeys.json

You must remove any old database that persists, e.g. the test keys database
(unless you specify append mode `-a` when running `pubkeydb`)

	$ rm -f pubkeys.db

Regenerate the database with only those keys found in pubkeys.json

	$ ./pubkeydb

Move your new `pubkey.db` file to the runtime directory
of your grpcsig-enabled server via a secure channel.


### Multiple User - Add a user to existing pubkeys.db

First make your new public key entry, e.g.:

	$ ecdsapubkeytojson.sh ~/.ssh/id_ecdsa.pub > mykey.json

Or, as an administrator, you can pass the `*keytojson.sh` script to a user and ask them to
reply with the output `mykey.json` file. This is a public key only.

	$ ./pubkeydb -a -i mykey.json
	# Examine contents
	$ ./pubkeydb -j

Move your new `pubkey.db` file to the runtime directory
of your grpcsig-enabled server via a secure channel.

### Removing a user from pubkeys.db

List the users in the current pubkeys.db file

	$ ./pubkeydb -j | json -ga name
	sally
	martin
	billy

Dump all the json to a file

	$ ./pubkeydb -j > pubkeydb_contents.json

Remove one with `grep -v`

	$ cat pubkeydb_contents.json | grep -v billy > newkeys.json

Check it

	$ cat newkeys.json | json -ga name
	sally
	martin

Overwrite pubkeys.json

	$ mv newkeys.json pubkeys.json

Remove the `pubkeys.db` database and rebuild it

	$ rm -f pubkeys.db
	# Regenerate the database from the new `pubkeys.json`
	$ ./pubkeydb

Check it

	$ ./pubkeydb -j | json -ga name
	sally
	martin


Move your new pubkey.db file to the runtime directory
of your grpcsig-enabled server via a secure channel.


### Add json Endpoints to their bucket

Same as adding pubkeys, just add the -t switch, and provide the 
endpoint json file:

	$ ./pubkeydb -a -t -i endpoints_test.json 
	Loading pubkeys.db from endpoints_test.json
	Finished

See the endpoints:

	$ ./pubkeydb -j -t 
	{"flockid":"/flock/horde1/node1/jettison/jettison0","netid":"/jettison1.0.1/NEADGEMAGQESWERR/net/localhost:33330","priority":"wow","rank":1,"stateadded":-1}
	{"flockid":"/flock/horde1/node1/jettison/jettison0","netid":"/jettison2.0.1/UTCXNEADGEMAGQER/net/localhost:33332","priority":"wow","rank":1,"stateadded":-1}
	{"flockid":"/flock/horde1/node1/phlogiston/phlogiston0","netid":"/phlogiston1.0.1/NEICRYGEMAGQSPQM/net/localhost:33331","priority":"wow","rank":1,"stateadded":-1}

### Miscellaneous

## Need to see what's in an existing `pubkey.db` file?

	$ ./pubkeydb -j
	$ ./pubkeydb -j -t

or if you have the npm json utility (by Trent Mick), you can pretty-print it.

	$ ./pubkeydb -j | json -ga
	$ ./pubkeydb -j -t | json -ga


## No current pubkeys.json file - but have a current pubkeys.db file?
Recreate the `pubkeys.json` file like so:

	$ ./pubkeydb -j > pubkeys.json

Then append your key to the file using the Multiple User steps above

## Append a single line entry in `mykey.json` to an existing `pubkeys.db` 

	$ ./pubkeydb -a -i=mykey.json



### TODO

Make a microservice out of this.
Add a -d delete command.
