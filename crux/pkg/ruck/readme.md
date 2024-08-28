---
show_footer: true
---

# pkg/ruck

Package of code that bootstraps crux services.

The flocking and service bootstrapping code for top-level crux executables 
`cmd/fulcrum` and `cmd/ripstop` are in  

`pkg/ruck/bootstrap.go`

Code entry points in `bootstrap.go` are as follows:

`fulcrum flock --strew` uses `BootstrapTest()`

`ripstop flock` uses `BootstrapRipstop()`

Beacon service with
`fulcrum watch` or `ripstop watch`, use `Bootstrap()`, which currently returns 
without doing anything.

For more details, see:  
[`crux/cmd/fulcrum`](https://github.com/erixzone/crux/tree/master/cmd/fulcrum) 

[`crux/cmd/ripstop`](https://github.com/erixzone/crux/tree/master/cmd/ripstop)

# The muster process

The muster process is a way of publishing code to be downloaded by crux.
There are two pieces: `tools/muster0`, which prepares a directory and a symbol table,
and internal Go code, used mainly by `picket` which uses the symbol table.

`muster0` `dir files ...`

takes the set of file arguments, copies them to the destination directory
and renames them as their checksum. It also constructs a `symtable` file
which contains text lines with two fields.
The first field is either a plugin routine name or an __@__ followed by the original
base name of the file.

The second field is the new (checksum) name.
This file is read by `ruck.ReadMuster()` and is used by _picket_ to load plugins
in the following way: plugin specifies both the filename and the plugin function name,
but if the filename is _@_, then _picket_ uses the symbol table to pick the (first)
pair that matches the plugin name.
