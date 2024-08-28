test1: {
	begatfile: test1.bg=4527a83f
	history: clear
	pastiche: []
	inputs: [ file1=23deadbeef file2=deadbeef42 ]
} => {
	dictums: [ d1 d3 d6 d9 d10 ]
	outputs: [ poot=71fea624 ]
}

test1a: test1 + { } => {
	dictums: []
}

test1b: test1 + {
	inputs: [ file1=77665544 ]
} => {
	dictums: [ d6 d10 ]
	outputs: [ poot=28ac321 ]
}

test1c: test1 + test5 + {
	pastiche: [ -abc.o=26a7bb2 ]
} => {
	dictums: []
}
