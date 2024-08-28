#!/bin/sh
sh -c './grpcsigsrv &'
sleep 5
# should trigger 4 database restarts in the watcher

echo "===1 Removing whitelist - all gone"
rm pubkeys_test.db

sleep 25

echo "===2 Replacing whitelist - bobo removed"
cp pubkeys_test2.ro pubkeys_test.db
ls -alt

sleep 25

echo "===3 Touching whitelist 1"
touch pubkeys_test.db

sleep 5

echo "===4 Replacing whitelist - bobo is back"
cp pubkeys_test.ro pubkeys_test.db

sleep 5

echo "===5 Touching whitelist 2"
touch pubkeys_test.db

echo "===Done"
sleep 5
exit 0
