// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package registrydb

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/pborman/uuid"

	pb "github.com/erixzone/crux/gen/cruxgen"
	c "github.com/erixzone/crux/pkg/crux"
)

// Temporary implementation of a json-based method for adding grpcsig authorisation rules
// regarding which clients are allowed to talk to which endpoints, and hence recieve
// updates of the appropriate netids and public keys, respectively.
// TODO - figure out how to ensure these are not added 2x on restart.
// Deal with Add/Kill individual rules.
// This should work - hash the rule string, store the hash as the ruleuuid
// Same rule string (here numbers 2, 3, 4) appends all clients and endpoints together into a single list of from-to
// Rule 1 is horde specific pastiche intercommunication
// Rule 2 is horde specific pastiche intercommunication
// Rule 3 is steward (which may land in either horde) to reeve (in both hordes) endpoint communication
// Rule 4 is reeve (in both hordes) to steward (which may be in either horde)  endpoint communication
// NOTE interprocess grpc (any service calling reeve) with self-signer bypasses these rules.
func dobaserules(db *sql.DB) *c.Err {
	baserules := `{"rule":"1", "horde":"sharks", "from":"Pastiche", "to":"Pastiche", "owner":"cruxadmin"}
	{"rule":"2", "horde":"jets", "from":"Pastiche", "to":"Pastiche", "owner":"cruxadmin"}
	{"rule":"3", "horde":"sharks", "from":"Steward", "to":"Reeve", "owner":"cruxadmin"}
	{"rule":"3", "horde":"jets", "from":"Steward", "to":"Reeve", "owner":"cruxadmin"}
	{"rule":"4", "horde":"sharks", "from":"Reeve", "to":"Steward", "owner":"cruxadmin"}
	{"rule":"4", "horde":"jets", "from":"Reeve", "to":"Steward", "owner":"cruxadmin"}
	{"rule":"5", "horde":"sharks", "from":"Heartbeat", "to":"HealthCheck", "owner":"cruxadmin"}
	{"rule":"6", "horde":"jets", "from":"Heartbeat", "to":"HealthCheck", "owner":"cruxadmin"}
	{"rule":"7", "horde":"jets", "from":"Proctor", "to":"HealthCheck", "owner":"cruxadmin"}
	{"rule":"8", "horde":"sharks", "from":"Proctor", "to":"HealthCheck", "owner":"cruxadmin"}
	{"rule":"9", "horde":"jets", "from":"Proctor", "to":"Picket", "owner":"cruxadmin"}
	{"rule":"10", "horde":"sharks", "from":"Proctor", "to":"Picket", "owner":"cruxadmin"}
	{"rule":"11", "horde":"jets", "from":"Flock", "to":"Genghis", "owner":"cruxadmin"}
	{"rule":"11", "horde":"sharks", "from":"Flock", "to":"Genghis", "owner":"cruxadmin"}
	{"rule":"11", "horde":"sharks", "from":"Flock", "to":"Genghis", "owner":"cruxadmin"}
	{"rule":"12", "horde":"Admin", "from":"Pastiche", "to":"Pastiche", "owner":"cruxadmin"}
	{"rule":"13", "horde":"Admin", "from":"Steward", "to":"Reeve", "owner":"cruxadmin"}
	{"rule":"14", "horde":"Admin", "from":"Reeve", "to":"Steward", "owner":"cruxadmin"}
	{"rule":"15", "horde":"Admin", "from":"Heartbeat", "to":"HealthCheck", "owner":"cruxadmin"}
	{"rule":"16", "horde":"Admin", "from":"Proctor", "to":"HealthCheck", "owner":"cruxadmin"}
	{"rule":"17", "horde":"Admin", "from":"Proctor", "to":"Picket", "owner":"cruxadmin"}
`
	/*  Rules needed for pkg/sample client foo, server bar
	{"rule":"5", "horde":"sharks", "from":"foo", "to":"bar", "owner":"cruxadmin"}
	{"rule":"6", "horde":"jets", "from":"foo", "to":"bar", "owner":"cruxadmin"}
	*/
	rules := []pb.RuleInfo{}
	rule := pb.RuleInfo{}
	scanner := bufio.NewScanner(strings.NewReader(baserules))
	for scanner.Scan() {
		line := []byte(scanner.Text())
		err := json.Unmarshal(line, &rule)
		if err != nil {
			return c.ErrF("json error in dobaserules : %v", err)
		}
		rules = append(rules, rule)
	}
	// fmt.Printf("Base Rules parsed: %v\n", rules)
	for _, group := range rules {
		_, err := MakeServiceGroup(db, group.From, group.To, group.Horde, group.Rule, group.Owner)
		if err != nil {
			return c.ErrF("db error in dobaserules : %v", err)
		}
	}
	CurrentRules = rules // TODO - Reset the rules

	return nil
}

// CurrentRules - rules used to connect clients/servers
var CurrentRules []pb.RuleInfo

// MakeServiceGroup - adds a rule to the database.
func MakeServiceGroup(db *sql.DB, fromservice, toservice, hordename string, ruleid string, ownerid string) (string, *c.Err) {
	// fmt.Println(query1)
	clgroupid, cerr := makeClgroup(db, fromservice, hordename)
	if cerr != nil {
		return "", cerr
	}
	// fmt.Println(query2)
	epgroupid, eerr := makeEpgroup(db, toservice, hordename)
	if eerr != nil {
		return "", eerr
	}
	ruleuuid, aerr := makeAllowed(db, ruleid, epgroupid, clgroupid, ownerid)
	if aerr != nil {
		return "", aerr
	}
	// fmt.Printf("Service Group %s added by %s to rule %s from clients-%s to endpoints-%s in horde-%s\n", ruleuuid, ownerid, ruleid, fromservice, toservice, hordename)
	return ruleuuid, nil
}

func makeClgroup(db *sql.DB, fromservice, hordename string) (string, *c.Err) {
	tx, derr := db.Begin()
	if derr != nil {
		return "", c.ErrF("error - MakeClgroup failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into clgroup(clgroupid, servicename, hordename) values(?, ?, ?)")
	if derr != nil {
		return "", c.ErrF("error - MakeClgroup failed Prepare - %v", derr)
	}
	defer stmt.Close()
	clgroupid := uuid.NewUUID().String()
	_, derr = stmt.Exec(
		clgroupid,
		fromservice,
		hordename)
	if derr != nil {
		return "", c.ErrF("error - MakeClgroup failed Exec - %v", derr)
	}
	txerr := tx.Commit()
	if txerr != nil {
		return "", c.ErrF("error - MakeClgroup failed Commit -  %v", txerr)
	}
	return clgroupid, nil
}

func makeEpgroup(db *sql.DB, toservice, hordename string) (string, *c.Err) {
	tx, derr := db.Begin()
	if derr != nil {
		return "", c.ErrF("error - MakeEpgroup failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into epgroup(epgroupid, servicename, hordename) values(?, ?, ?)")
	if derr != nil {
		return "", c.ErrF("error - MakeEpgroup failed Prepare - %v", derr)
	}
	defer stmt.Close()
	epgroupid := uuid.NewUUID().String()
	_, derr = stmt.Exec(
		epgroupid,
		toservice,
		hordename)
	if derr != nil {
		return "", c.ErrF("error - MakeEpgroup failed Exec - %v", derr)
	}
	txerr := tx.Commit()
	if txerr != nil {
		return "", c.ErrF("error - MakeEpgroup failed Commit -  %v", txerr)
	}
	return epgroupid, nil
}

func makeAllowed(db *sql.DB, ruleid, epgroupid, clgroupid, ownerid string) (string, *c.Err) {
	tx, derr := db.Begin()
	if derr != nil {
		return "", c.ErrF("error - MakeAllowed failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into allowed(ruleuuid, ruleid, epgroupid, clgroupid, ownerid) values(?, ?, ?, ?, ?)")
	if derr != nil {
		return "", c.ErrF("error - MakeAllowed failed Prepare - %v", derr)
	}
	defer stmt.Close()
	ruleuuid := uuid.NewUUID().String()
	_, derr = stmt.Exec(ruleuuid,
		ruleid,
		epgroupid,
		clgroupid,
		ownerid)
	if derr != nil {
		return "", c.ErrF("error - MakeAllowed failed Exec - %v", derr)
	}
	txerr := tx.Commit()
	if txerr != nil {
		return "", c.ErrF("error - MakeAllowed failed Commit -  %v", txerr)
	}
	return ruleuuid, nil
}

// GetRules - retrieves an array of unique rule identifiers from the allowed table
func GetRules(db *sql.DB) ([]string, *c.Err) {
	// fmt.Println("*********BEGIN getRules")
	// defer fmt.Println("*********END getRules")
	rows, derr := db.Query("SELECT DISTINCT ruleid FROM allowed")
	if derr != nil {
		return nil, c.ErrF("query in GetRules: %v", derr)
	}
	defer rows.Close()
	rules := []string{}
	for rows.Next() {
		var rule string
		derr = rows.Scan(&rule)
		if derr != nil {
			return nil, c.ErrF("scan in GetRules: %v", derr)
		}
		rules = append(rules, rule)
	}
	derr = rows.Err()
	if derr != nil {
		return nil, c.ErrF("rows in GetRules: %v", derr)
	}
	return rules, nil
}
