test1: {
	begatfile: ../tests/test4.begat
	history: clear
	pastiche: []
	inputs: []
} => {
	dictums: [ dict0001 dict0002 dict0003 dict0004 ]
	outputs: [ rx/all.wc=2841e98dae4de7261742e2308768d6f4d58736e2769c80d810683f5476c2a67111474a4917d4608de2d2f982bfeecd902ab4fc754ed2532a13e9451576a6e62e ]
}

test1a: test1 + {} => {
	dictums: []
	outputs: [ rx/all.wc=2841e98dae4de7261742e2308768d6f4d58736e2769c80d810683f5476c2a67111474a4917d4608de2d2f982bfeecd902ab4fc754ed2532a13e9451576a6e62e ]
}
