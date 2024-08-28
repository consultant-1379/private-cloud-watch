/*
Package lib implements some or most of the begat stuff.

given a begatfile, we can generate a list of dictums, stemming either from
a default or specific targets. if we are given a list of target files, then
we prune that dictum list to those involved in produsing those files.

otherwise, the overall action is
	- generate chores
	- spawn chores
	- wait until fs&exec action stops, then success is all status in [Pretend,Done]

testing strategy:

some tests just check basic stuff:
	parse_test.go:		tests the chain from text source to compiled stuff
	ursort_test.go:		tests the topological sort code
	fsroute_test.go: 	tests routing of file system events
	begat_test.go: 		tests the overall begat logic using a fake exec component

don't know yet how to talk about integration tests here.

the begat logic:

	at the risks of repeating common knowledge, lets review the basics:
+ we'll talk about chores; these are fully expanded dictums ready to execute
+ all the chores for a begat run run with a single uniform filesystem namespace
+ (for this exposition) each chore consists of input files, output files and a recipe
	(which generates the output files from the input files)
+ the chores form a directed graph (which may contain cycles) by the relation
	"depends on"; chore A depends on chore B if an output file of B has the same name
	as an input file for A.
+ a travail is a historical record of a chore that executed

	if the graph was acyclic, then it would be easy to process. we would sort the chores
(the nodes in the graph) with a topological sort, and then process each chore in reverse order:

		if any input files are MISSING { exitwitherror "missing input files" }
		if we have a travail t for this chore and matching our input files {
			for each output file o {
				if o is MISSING { fetch a copy of that file with the checksum from t }
				if o is EXISTING { verify it has the checksum from t }
			}
		}
		if any output files are MISSING { execute the recipe }

	that is to say, as long as we process chores in the right order, all the logic is purely local.
unfortunately, we want to support a couple of features which make this harder:
	1) missing intermediates
	2) dependency cycles

lets quickly review what these mean. take this example of three chores:
		a.out -- [cc a.o b.o] -- a.o -- [cc -c a.c] -- a.c
				      -- b.o -- [cc -c b.c] -- b.c

	"missing intermediates" means that under certain circumstances, we don't require files
to actually exist. imagine that in the above example,
after generating a.out, the leftmost chore removed a.o and b.o (in order to save space). if we run
begat again, should it do anything? naively, the answer is yes, because in begat, we are executing
all the given chores each run (unless we have the cached results). but that seems unnecessary.
if we have a travail (a historical record of an executed chore), and see that when we ran the
"cc -c a.c" chore with the extant a.c, we generated an a.o with, say, checksum ha. similarly,
we previously have generated an b.o with checksum hb. now, if we have a travail where with inputs
cksum(a.o)==ha and cksum(b.o)==hb and the output of "cc a.o b.o" generated an a.out whose checksum
is the same as the existing a.out, then surely we don't need to run anything.

	"cyclic dependencies" is basically where a file is both an input and an output for a chore.
(this can involve multiple chores chained together.) a simple example is
			cc -- [cc -o cc cc.c] -- cc.c
					      -- cc

in this example, we start off with cksum(cc)==h0 and cksum(cc.c)==h1. we modify cc.c so that cksum(cc.c)==h2.
running the chore yields a cc such that cksum(cc)=h3. we run the chore again and get a new cc such that cksum(cc)=h5.
if we run the chore again, cksum(cc) remains h5. essentially, we repeat executing the recipe until we see
the same result repeating. (if it doesnt happen within a modest number of executions, it is an error.)
generally speaking, a chore does not know that it is part of a cycle (although in this trivial example it could).

	both these features require that a chore compute its state potentially multiple times; the question
is how to coordinate all of this? it is worthwhile thinking of a hard example, namely of a long graph
where are the chores "pretend" they ran (because of missing intermediates) but the final (top) chore
really does need to run. this means that (eventually) all the other chores will need to run so that
we generate the missing intermediate files.

proposed solution:
	each file will be one one of
		MISSING, no chksum
		EXISTING, chksum
		PRETEND, chksum
		NEED
	each chore will be in one of these states
		START (just the initial state; no other info)
		PRETEND (we've set output files on the basis of a matching travail)
		EXECUTED (we ran the recipe and generated new output files)
		WAITING (we can't run or pretend until something changes)
		CANRUN (we have all inputs ready and need to run, but have not yet)
		RUNNING (while executing the recipe)
	the chores are arranged in groups (from the topological sort) from "top" (target) to "bottom" (no chores supporting them)
	there is a global flag EXEC_OK which if true allows chores to execute their recipe (if in CANRUN state)

initial state:
	all files are set to MISSING or EXISTING (according to whether or not they exist)
	all chores are set to START
	EXEC_OK = true

control process:
	for progress = true; progress {
		progress = false
		for g := bottom .. top {
			rungroup(g)
			if any chore in g set NEED {
				EXEC_OK = false
				// down to bottom because thats the way NEEDs propagate
				for x := g .. bottom {
					rungroup(x)
				}
				EXEC_OK = true
				g = bottom
				continue
			}
			if chore in g progressed {
				progress = true
			}
		}
		if all chores in top are in [PRETEND,EXECUTED] {
			return success
		}
	}
	return error

rungroup(g []chore):
	for progress = true; progress {
		progress = false
		for c := each chore in g {
			step(c)
			if c set NEED {
				return
			}
			if c.state changed or c changed any of its inputs or outputs {
				progress = true
			}
		}
	}

step(c chore):
	if ((any output is NEED) || (there is no travail matching inputs)) && (state == PRETEND) {
		state = WAITING
		change all PRETEND outputs to MISSING
		change all pretend inputs to NEED
	}
	if (there is a travail t whose inputs' checksum match our inputs) && (state in [START,PRETEND,WAITING]) {
		state = PRETEND
		set all outputs in [MISSING,PRETEND] to be PRETEND with corresponding checksums from t
		return
	}
	if (state in [START,WAITING]) && (all inputs exist) {
		state = CANRUN
	}
	if any input files have changed checksum since RUNNING {
		state = CANRUN
	}
	if (state == CANRUN) && EXEC_OK {
		state = RUNNING
		run recipe
		update files changed in recipe execution
		record travail for this chore execution
		state = EXECUTED
	}

*/
package lib
