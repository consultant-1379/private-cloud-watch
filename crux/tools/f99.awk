function mem(		var, val, x) {
	delete m
	for(x = $0; length(x) > 0;){
		if(match(x, "^[a-zA-Z0-9]*=") == 0){
			break
		}
		var = substr(x, RSTART, RLENGTH-1)
		val = substr(x, RLENGTH+1, 1)
		if(val == "\""){
			val = substr(x, RLENGTH+2)
			p = match(val, "\"")
			x = substr(val, p+1)
			val = substr(val, 1, p-1)
		} else {
			val = substr(x, RLENGTH+1)
			if(match(val, "[^ ]*") > 0){
				x = substr(val, RLENGTH+1)
				val = substr(val, 1, RLENGTH)
			} else {
				x = ""
			}
		}
		m[var] = val
		if(match(x, "^[ ]*") > 0){
			x = substr(x, RLENGTH+1)
		}
	}
}
function node(x,	i){
	for(i in nodes) if(nodes[i] == x) return
	nnodes = nnodes+1
	nodes[nnodes] = x
	newnode = 1
}
function prcol(){
	lastcol--
	if(lastcol <= 0) newnode = 1
	if(newnode > 0){
		printf("time ")
		for(i = 1; i <= nnodes; i++) printf(" %13.13s", nodes[i])
		printf("\n")
		newnode = 0
		lastcol = 12
	}
	printf("%s", substr(time, length(time)-5, 5))
	for(i = 1; i <= nnodes; i++) printf(" %13.13s", substr(clust[nodes[i]], 2))
	printf("  %s\n", expl)
}
/^time/ {
	mem()
	if(match(m["msg"], "^inbound") > 0) next
	if(match(m["msg"], "^read") > 0) next
	if(match(m["msg"], "^recv") > 0) next
	if(match(m["msg"], "^start new flock") > 0){
		node(m["node"])
		expl = sprintf("start new flock for %s", m["node"])
		clust[m["node"]] = m["flock"]
		time = m["time"]
		prcol()
		next
	}
	if(match(m["msg"], "^change flock") > 0){
		expl = sprintf("change flock for %s", m["node"])
		clust[m["node"]] = m["flock"]
		time = m["time"]
		prcol()
		next
	}
	if(match(m["msg"], "^membership") > 0) next
	if(match(m["msg"], "^starting stable") > 0){
		expl = m["msg"]
		sub(".*test", "stable", expl)
		time = m["time"]
		prcol()
		next
	}
	print "__" m["msg"]
	next
}
	{ print ">>" $0
}