This is a directory for synthesising varous metric data.

The program stems from synth.go. You can pretty print the
output (at least for metric 1) via pp.sh, as in
	./pp.sh < m1.test

Metric 1: there is a sample test file in m1.test. the usage is

	synth -m 1 -t 20 nnodes [down downmean downstddev]

If down is not set, then no nodes go down. Otherwise, a node
will on average go down "down" of teh time, and the down time will
have be normally distributed with a mean value of downmean and a
std dev of downstddev. For example,
	synth -m 1 -t 20 10 .1 2 3
will normally yield about 8-10 nodes being down.
