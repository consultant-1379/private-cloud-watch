# package muck

Initializes directories in .muck used for storing persistent things that
need to survive process restarts, like keys, uids, etc.

Does not provide access to the things in those directories, with the exception
of a few identifiers at the top level .muck/ directory.

serviceid - an NUID style unique identifier for the .muck and all the services that use it
hoardeid - To be added
flockid - To be added

## You May... add a directory to .muck 
1. modify  setMuckSubdirs() to add another subdirectory 
2. add a public GetXXX() to retrieve the path to that subdirectory.

That's it.  Anything else you want to do inside that subdirectory, keep it out of this package.
