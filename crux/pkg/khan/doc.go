/*
Package khan implements the khan service.

A khan specification is a list of expressions that specify where
various services should be deployed across a set of nodes.
It is declarative, and the expressions are evaluated as a whole.
A simple example is
	sp := pick(segp, 5, ALL)
	ns3 := 3
	s3 := pick(s3, ns3, !LABEL(KV))
	pick(poot, 2, s3&sp)

This says that the service "segp" will be deployed with a count of 1
on each of 5 nodes. The set of 5 nodes (khan picks) is assigned to the variable "sp".
The variable ns3 has the value of 3. The service "s3" will be deployed with a count of 1 on
ns3 (== 3) nodes which do not have the label "KV", and this set is assigned to "s3".
The service "poot" will be deployed with a count of 1 on 2 nodes that belong to both s3 and sp.

The above has a small lie; it actually describes the "pickh" operator, which is rarely used.
The normal "pick" operator silently fails gracefully by putting more instances onto
a node when there are fewer nodes available than the count. (so in the above example,
if ALL only had 4 nodes, then one node would get 2 instances.) In this case, the "conform"
output would contain a warning; if it had said "pickh", then khan would fail to find a solution
and return an error.

Administrative:
	The primary interface (used by the stix subcommand "khan") is Khan() in khan.go.
This file also describes where the specification comes from, and the various breadcrumbs
that khan leaves behind, including explanations of what khan did, and a history of past actions.

The function lineup() (lineup.go) is the glue code that extracts stuff from the KV store
ready for the code that actually computes the answers.

The function Diff() (diff.go) takes an answer distribution and the current distribution
and performs the appropriate actions to "make it so".

Implementation:
	This problem is very (NP) hard. There are commercial (and other) solutions, such as
Micorsofts Z3 system, and are generally known as SMT (satisfiability modulo theories) solvers.
But at the time of writing, these were too hard and too few to interface to. So we wrote our own
which should work fine for small to medium scale problems. Be warned: the code is not straightforward.

The strategy is an intertwined pair of routines that exhaustively (and recursively) look at all
possible solutions. By look at, we mean it feeds them (thru a channel) to a goroutine that looks
for the solution "closest" to the existing state of services. The recursive procedure looks like this:
	reduce(el expr_list){
		is there a pick operator without a value that we can use (all of its parameters are known)?
		then
			e := pick operator expression
			drive(e, el-e)
		else
			evaluate el (and send off any solution)
		fi
	}
	drive(e expr, el expr_list){
		over all the possible distributions that satisfy the pick operator in e {
			evaluate el (assigning the current distribution to the pick operator)
			reduce(el)
		}
	}
There are other arguments, but this is the heart of the matter. Each recursive call to reduce operates
on a smaller list with one less pick operator, so it clearly terminates. But at each level, the number
of solutions multiplies by the number of possible solutions for the pick operator at hand.

reduce() and drive() are in solve.go, and strew.go contains strew() which generates possible solutions.
The distributions are implemented as the type "diaspora".

Language:
	The language is parsed via the normal Unix tools yacc and lex (of rather, the Go version "nex").
The grammar is in parse.y (the generated code is over in stix/gen) and the lexer is in lexer.nex.
Both are standard examples of simple expression syntaxes. There is a helper script (gen.bash)
that stops vet and lint whining about the generated code; this may need to be updated should
vet and/or lint get fussier.

Sequencing:
	Khan also offers a certain amount of sequencing support. As an example:
A simple test of the S3 server involves the following components in this sequence:
	Start the kv service
	load the kv store.
	Start the S3 server (note that it needs a bunch of stuff to be in the kv store)
	Run a test program that, amongst other stuff, creates some S3 jobs.
	When the S3 jobs have finished, shut everything down.
We can describe this as activities with prerequisites:
	Activity	Prerequisites
	Start kv
	Start kv_load	kv.ready
	Start s3	kv_load.done
	Start s3_test 	S3.ready (functionally kv.ready, but to do timing, S3.ready)
	Stop everything	s3_test.done

You specify sequencing relationships to khan, with directives of the form
	start p after {[n[%] of] q.stage}*
which would then be topologically sorted into a starting order. n can either be an absolute number or
a percentage (of planned). (The stage has no formal semantics and we take what a service says at face value.)
These are handled as constraints in diff.go, mainly in the function allow().
*/
package khan
