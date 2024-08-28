/*
	this is Go code corresponding to the following proto code

message EndSel {
	string hordeid = 1;		// if blank, then all hordes are valid
	string servicerev = 2;		// if blank, then all revisions ok. if not blank, then only return revisions > servicerev
	string servicename = 3;		// if blank, ten all services ok. if not blank, then only matching services
	string limit = 4;		// return at most limit entries. interpret limit==0 as unlimited
}

message EndPoint {
	string hordeid = 1;
	string servicerev = 2;		// if blank, then all revisions ok. if not blank, then only return revisions > servicerev
	string servicename = 3;		// if blank, ten all services ok. if not blank, then only matching services
	string keyid = 4;
}

message EndRet {
	repeated EndPoint list = 1;
	string error = 2;
}

we have an entry point:
	FetchEndpoints(EndSel) EndRet
*/
package main

import () // nada

type EndSel struct {
	hordeid                 string
	servicerev, servicename string
	limit                   int
}

type EndPoint struct {
	hordeid                 string
	servicerev, servicename string
	keyid                   string
}

type EndRet struct {
	list []EndPoint
	err  string
}

func FetchEndPoints(e EndSel) EndRet {
	return EndRet{err: ""}
}

// sample functions below

// HordeNames returns a list of node names in the horde. if horde is "", then all nodes in cluster
func HordeNames(horde string) []string {
	er := FetchEndPoints(EndSel{hordeid: horde, servicerev: "", servicename: "", limit: 0})
	if er.err != "" {
		// log er.error
		return nil
	}
	m := make(map[string]bool, len(er.list)/5) // approximate guess
	for i := range er.list {
		m[er.list[i].keyid] = true // should be node(keyid)
	}
	var ret []string
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}

// SelectService picks a service keyid. a few embedded rules detailed below.
// if possible, return the currentkey
func SelectService(horde string, name string, oldrev string, curKey string) (rev, keyid string) {
	// as always, get the list
	er := FetchEndPoints(EndSel{hordeid: horde, servicerev: oldrev, servicename: name, limit: 0})
	if er.err != "" {
		// log er.error
		return "", ""
	}
	// do the easy case first: no previous service revision.
	if oldrev == "" {
		foundkey := false
		keyid = ""
		for i := range er.list {
			if er.list[i].servicerev > oldrev {
				oldrev = er.list[i].servicerev
				keyid = er.list[i].keyid
				foundkey = false
			}
			if (oldrev == er.list[i].servicerev) && (er.list[i].keyid == curKey) {
				foundkey = true
			}
		}
		if foundkey {
			// here's where you could do loadbalancing etc
			keyid = curKey
		}
		return oldrev, keyid
	}
	// here's where we run into trouble. we need the highest service revision that matches the major version of what we want.
	foundkey := false
	keyid = ""
	for i := range er.list {
		if false { // this test should be major(oldrev) == major(er.list[i].servicerev)
			continue
		}
		if er.list[i].servicerev > oldrev {
			oldrev = er.list[i].servicerev
			keyid = er.list[i].keyid
			foundkey = false
		}
		if (oldrev == er.list[i].servicerev) && (er.list[i].keyid == curKey) {
			foundkey = true
		}
	}
	if foundkey {
		// here's where you could do loadbalancing etc
		keyid = curKey
	}
	return oldrev, keyid
}
