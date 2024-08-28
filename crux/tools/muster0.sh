#!/bin/sh

export LANG=C

help(){
	cat <<'EOF'
	muster0 is a script to help set up source files for pastiche.
Usage is
	muster0 [--help] destdir srcfiles ...

The srcfiles are copied into destdir with their destination names
being the pastiche checksums. In addition, a catalog file is created in destdir
with a line for every function in a srcfile. If no such function exists, then
a fictional name of @ is used.
EOF
exit 0
}
#nm fulcrum | grep ' T github.com/erixzone/crux/pkg/ruck.' | grep _

case "$1" in
"-help"|"--help")
	help
	;;
esac

dest=$1; shift

(
	pastiche cksum $*
	nm --defined-only $*     # on some systems, -U instead of --defined-only
) | awk -vdest=$dest '
BEGIN {
	symt = ""
}
NF==1 {
	file = $1
	sub(":$", "", file); sub(".*plugin_", "plugin_", file)
	symt = symt "@" file " " hash[file] "\n"
	next
}
NF==2 {
	xx = $1; sub(".*plugin_", "", xx)
	hash[xx] = $2
	printf("cp %s %s/%s\n", $1, dest, $2)
	next
}
/ T github.com\/erixzone\/crux\/pkg\/ruck\..*_/ {
	sym = $0
	sub(".*ruck[.]", "", sym)
	symt = symt sym " " hash[file] "\n"
	next
}
	{
	next
}
END {
	printf("cat > %s/symtab <<\\EOF\n%sEOF\n", dest, symt)
}'
