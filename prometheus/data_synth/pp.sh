#!/bin/bash

gawk -v ls="[" -v rs="]" '
BEGIN {
	LB = "["
	RB = "]"
	ind = ""
}
 {
	x = $0
	while( 1 ){
		l = index(x, LB)
		r = index(x, RB)
		if((l == 0) && (r == 0)) break
		if((l > 0) && ((r == 0) || (l < r))){
			# LB is leftmost
			ind = ind "\t"
			printf("%s\n%s", substr(x, 1, l), ind)
			x = substr(x, l+1)
			continue
		}
		if((r > 0) && ((l == 0) || (r < l))){
			# RB is leftmost
			ind = substr(ind, 1, length(ind)-1)
			printf("%s\n%s", substr(x, 1, r), ind)
			x = substr(x, r+1)
			continue
		}
	}
	printf("%s\n%s", x, ind)
}'
