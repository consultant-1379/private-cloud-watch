// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package registrydb

import (
	"database/sql"
	"fmt"

	pb "github.com/erixzone/crux/gen/cruxgen"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/idutils"
)

// UpdateOnTick - high level call that aggregates the information
// from queries for the updates to reeves required on a given clock state "tick"
func UpdateOnTick(db *sql.DB, tick int32) (TickUpdates, *c.Err) {
	// Do the tick update queries
	tus := TickUpdates{}
	tus.Tick = int(tick)
	rules, rerr := GetRules(db)
	if rerr != nil {
		return TickUpdates{}, c.ErrF("GetRules in UpdateOnTick %v", rerr)
	}
	for _, rule := range rules {
		tu, err := QueryRuleTick(db, rule, tus.Tick)
		if err != nil {
			return TickUpdates{}, c.ErrF("QueryRuleTick in UpdateOnTIck on rule %s : %v", rule, err)
		}
		tus.Updates = append(tus.Updates, *tu)
	}
	return tus, nil
}

// ReeveActions - each reeve gets sent the entire list of endpoint info and/or  keys
type ReeveActions struct {
	Reeves []idutils.NetIDT
	Epinfo []pb.EpInfo
	Keys   []string // the stateadded field is always 0 here, updated at insert time.
}

// TickUpdate - holds the updates required to be disseminated for a
// given ingest clock tick and a given ruleid for each of the
// 6 update cases. Reeves are resolved within the query system,
// so that Steward can consolidate all information to be sent
// to each individual reeve, and send the update in a single transfer
type TickUpdate struct {
	Ruleid       string
	Oldendpoints *ReeveActions // old ep get new cl Keys
	Oldclients   *ReeveActions // old cl get new ep NodeID/NetIDs
	Newendpoints *ReeveActions // new ep get all cl Keys
	Newclients   *ReeveActions // new cl get all ep NodeID/NetIDs
	Delendpoints *ReeveActions // all ep del marked cl Keys
	Delclients   *ReeveActions // all cl del marked NodeID/NetIDs
}

// TickUpdates - All the updates for all the rules in a given ingest clock tick
type TickUpdates struct {
	Tick    int
	Updates []TickUpdate
}

type rulePair struct {
	epgroupid string
	clgroupid string
}

type queryParam struct {
	toservice   string
	tohorde     string
	fromservice string
	fromhorde   string
}

// QueryRuleTick - returns a TickUpdate with the set of updates
// required for fanout for a rule id on a given clock tick
// according to the stored rule specified by ruleid.
// first - this thing digs out the multipart components of the rule
// with a hit to the allowed table, then pulls out the
// details from the epgroup and clgroup table for each line in the
// rule.  Then it fires off the set of parameters for the 6 query cases
// needed in the TickUpdate above.
func QueryRuleTick(db *sql.DB, ruleid string, tick int) (*TickUpdate, *c.Err) {
	// fmt.Printf("\n*********BEGIN QueryRuleTick %s %d\n", ruleid, tick)
	// defer fmt.Printf("*********END QueryRuleTick %s %d\n", ruleid, tick)
	tu := TickUpdate{}

	// Get the common rule parameters from the ruleid
	rules := []rulePair{}
	rows, derr := db.Query("select ruleuuid, epgroupid, clgroupid, ownerid from allowed where ruleid = '" + ruleid + "'")
	if derr != nil {
		return nil, c.ErrF("query in QueryRuleTick: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var ruleuuid string
		var epgroupid string
		var clgroupid string
		var ownerid string
		derr = rows.Scan(&ruleuuid, &epgroupid, &clgroupid, &ownerid)
		if derr != nil {
			return nil, c.ErrF("scan in QueryRuleTick: %v", derr)
		}
		// fmt.Println(ruleuuid, epgroupid, clgroupid, ownerid)
		rules = append(rules, rulePair{epgroupid: epgroupid, clgroupid: clgroupid})
	}
	derr = rows.Err()
	if derr != nil {
		return nil, c.ErrF("rows in QueryRuleTick: %v", derr)
	}
	// fmt.Printf("Common rules for ruleid %s : %v\n", ruleid, rules)
	if len(rules) == 0 {
		// Nothing to do
		return nil, nil
	}

	// Gather the all the listed rule query parameters
	queryParams := []queryParam{}
	for _, rule := range rules {
		qp := queryParam{}
		// Find the rule Endpoint Query parameters
		eprow, derr := db.Query("select servicename, hordename from epgroup where epgroupid = '" + rule.epgroupid + "'")
		if derr != nil {
			return nil, c.ErrF("query 2 in QueryRuleTick: %v", derr)
		}
		defer eprow.Close()
		var toservice string
		var tohorde string
		for eprow.Next() { // This is only one row.
			derr = eprow.Scan(&toservice, &tohorde)
			if derr != nil {
				return nil, c.ErrF("scan 2 in QueryRuleTick: %v", derr)
			}

			// fmt.Println("EP QUERY: " + toservice + " " + tohorde)
		}
		derr = eprow.Err()
		if derr != nil {
			return nil, c.ErrF("rows 3 in QueryRuleTick: %v", derr)
		}
		qp.toservice = toservice
		qp.tohorde = tohorde

		// Find the rule Client Query Parameters
		clrow, derr := db.Query("select servicename, hordename from clgroup where clgroupid = '" + rule.clgroupid + "'")
		if derr != nil {
			return nil, c.ErrF("query 3 in QueryRuleTick: %v", derr)
		}
		defer clrow.Close()
		var fromservice string
		var fromhorde string
		for clrow.Next() { // This is only one row
			derr = clrow.Scan(&fromservice, &fromhorde)
			if derr != nil {
				return nil, c.ErrF("scan 3 in QueryRuleTick: %v", derr)
			}
			// fmt.Println("CL QUERY: " + fromservice + " " + fromhorde)
		}
		derr = clrow.Err()
		if derr != nil {
			return nil, c.ErrF("rows 3 in QueryRuleTick: %v", derr)
		}
		qp.fromservice = fromservice
		qp.fromhorde = fromhorde
		queryParams = append(queryParams, qp)
	}

	// fmt.Printf("Query Parameters for ruleid %s: %v\n", ruleid, queryParams)

	tu.Ruleid = ruleid
	var oerr *c.Err
	tu.Oldendpoints, oerr = queryRuleTickX(db, &queryParams, tick, "oldendpoints")
	if oerr != nil {
		return nil, oerr
		// handle error
	}
	tu.Oldclients, oerr = queryRuleTickX(db, &queryParams, tick, "oldclients")
	if oerr != nil {
		return nil, oerr
		// handle error
	}
	tu.Newendpoints, oerr = queryRuleTickX(db, &queryParams, tick, "newendpoints")
	if oerr != nil {
		return nil, oerr
		// handle error
	}
	tu.Newclients, oerr = queryRuleTickX(db, &queryParams, tick, "newclients")
	if oerr != nil {
		return nil, oerr
		// handle error
	}
	tu.Delendpoints, oerr = queryRuleTickX(db, &queryParams, tick, "delclients")
	if oerr != nil {
		return nil, oerr
		// handle error
	}
	tu.Delclients, oerr = queryRuleTickX(db, &queryParams, tick, "delendpoints")
	if oerr != nil {
		return nil, oerr
		// handle error
	}
	return &tu, nil
}

// queryRuleTickX - given the details of a ruleid as query parameters -
// for whichever of the 6 update cases we are in - kind - assemble the subquery
// needed and fire off the detailed query depending on whether we
// need the reeve, endpoint or key information.
// return the actions required - reeves to contact and keys/epinfo
// to queryRuleTick
func queryRuleTickX(db *sql.DB, queryparams *[]queryParam, tick int, kind string) (*ReeveActions, *c.Err) {
	// run all the queryparams (list arising from a given ruleid) - through all the specified subqueryies,
	// accumulate the results = which are two lists of client and endpoint uuids.
	actions := ReeveActions{}
	for _, qp := range *queryparams {
		var rerr *c.Err

		/*
		   	Update class "kind" expanded into distintic things to gather:
		   kind	oldendpoints     - old ep get new cl Keys
		   	oldendpointsto   - reeves handling old ep
		   	oldendpointsfrom - keys from new cl

		   kind	oldclients       - old cl get new ep NodeID/NetIDs
		   	oldclientsto     - nod/nid of new ep
		   	oldclientsfrom   - reeves handling old cl

		   kind	newendpoints     -  new ep get all cl Keys
		   	newendpointsto   -  reeves handling new ep
		   	newendpoitnsfrom -  keys from all cl

		   kind	newclients       -  new cl get all ep NodeID/NetIDs
		   	newclientsto     -  nod/nid of all ep
		   	newclientsfrom   -  reeves handling new cl

		   kind	delendpoints     -  all ep del marked cl Keys
		   	delendpointsto   -  reeves handling all ep
		   	delendpointsfrom -  keys from del cl

		   kind	delclients       -  all cl del marked ep NodeID/NetIDs
		   	delclientto      -  nod/nid of the ep
		   	delclientfrom    -  reeves handling all cl
		*/

		switch kind {
		case "oldendpoints":
			subquery, qerr := getSubquery(qp.toservice, qp.tohorde, "endpoint", tick, "oldendpointsto", true)
			if qerr != nil {
				return nil, qerr
			}
			// Get reeves from endpoints
			reeves, err := reevesFromSubquery(db, "endpoint", subquery)
			// append onto result
			if err != nil {
				return nil, err
			}
			actions.Reeves = append(actions.Reeves, reeves...)

			subquery, rerr = getSubquery(qp.fromservice, qp.fromhorde, "client", tick, "oldendpointsfrom", false)
			if rerr != nil {
				return nil, qerr
			}
			// Get keys from clients
			keys, kerr := keysFromClients(db, subquery)
			if kerr != nil {
				return nil, kerr
			}
			actions.Keys = append(actions.Keys, keys...)

		case "oldclients":
			subquery, qerr := getSubquery(qp.toservice, qp.tohorde, "endpoint", tick, "oldclientsto", false)
			if qerr != nil {
				return nil, qerr
			}
			// Get nod/nid from endpoints
			epinfo, eerr := infoFromEndpoints(db, subquery)
			if eerr != nil {
				return nil, eerr
			}
			actions.Epinfo = append(actions.Epinfo, epinfo...)

			subquery, rerr = getSubquery(qp.fromservice, qp.fromhorde, "client", tick, "oldclientsfrom", true)
			if rerr != nil {
				return nil, qerr
			}
			// Get reeves from clients
			reeves, err := reevesFromSubquery(db, "client", subquery)
			if err != nil {
				return nil, err
			}

			actions.Reeves = append(actions.Reeves, reeves...)

		case "newendpoints":
			subquery, qerr := getSubquery(qp.toservice, qp.tohorde, "endpoint", tick, "newendpointsto", true)
			if qerr != nil {
				return nil, qerr
			}
			// Get reeves from endpoints
			reeves, err := reevesFromSubquery(db, "endpoint", subquery)
			if err != nil {
				return nil, err
			}

			actions.Reeves = append(actions.Reeves, reeves...)

			subquery, rerr = getSubquery(qp.fromservice, qp.fromhorde, "client", tick, "newendpointsfrom", false)
			if rerr != nil {
				return nil, qerr
			}
			// Get keys from clients
			keys, kerr := keysFromClients(db, subquery)
			if kerr != nil {
				return nil, kerr
			}
			actions.Keys = append(actions.Keys, keys...)

		case "newclients":
			subquery, qerr := getSubquery(qp.toservice, qp.tohorde, "endpoint", tick, "newclientsto", false)
			if qerr != nil {
				return nil, qerr
			}
			// Get nod/nid from endpoints
			epinfo, eerr := infoFromEndpoints(db, subquery)
			if eerr != nil {
				return nil, eerr
			}
			actions.Epinfo = append(actions.Epinfo, epinfo...)

			subquery, rerr = getSubquery(qp.fromservice, qp.fromhorde, "client", tick, "newclientsfrom", true)
			if rerr != nil {
				return nil, qerr
			}
			// Get reeves from clients
			reeves, err := reevesFromSubquery(db, "client", subquery)
			if err != nil {
				return nil, err
			}
			actions.Reeves = append(actions.Reeves, reeves...)

		case "delendpoints":
			subquery, qerr := getSubquery(qp.toservice, qp.tohorde, "endpoint", tick, "delendpointsto", true)
			if qerr != nil {
				return nil, qerr
			}
			// Get reeves from endpoints
			reeves, err := reevesFromSubquery(db, "endpoint", subquery)
			if err != nil {
				return nil, err
			}
			actions.Reeves = append(actions.Reeves, reeves...)

			subquery, rerr = getSubquery(qp.fromservice, qp.fromhorde, "client", tick, "delendpointsfrom", false)
			if rerr != nil {
				return nil, qerr
			}
			// Get keys (to del) from clients
			keys, kerr := keysFromClients(db, subquery)
			if kerr != nil {
				return nil, kerr
			}
			actions.Keys = append(actions.Keys, keys...)

		case "delclients":
			subquery, qerr := getSubquery(qp.toservice, qp.tohorde, "endpoint", tick, "delclientsto", false)
			if qerr != nil {
				return nil, qerr
			}
			// Get nod/nid from endpoints
			epinfo, eerr := infoFromEndpoints(db, subquery)
			if eerr != nil {
				return nil, eerr
			}

			actions.Epinfo = append(actions.Epinfo, epinfo...)
			subquery, rerr = getSubquery(qp.fromservice, qp.fromhorde, "client", tick, "delclientsfrom", true)
			if rerr != nil {
				return nil, qerr
			}
			// Get reeves from clients
			reeves, err := reevesFromSubquery(db, "client", subquery)
			if err != nil {
				return nil, err
			}
			actions.Reeves = append(actions.Reeves, reeves...)

		default:
			return nil, c.ErrF("Bad kind string in queryRuleTickX: %s", kind)
		}
	}
	if len(actions.Reeves) == 0 || (len(actions.Epinfo) == 0 && len(actions.Keys) == 0) {
		// if one or the other side is empty, there is nothing to report
		return nil, nil
	}
	/*
		fmt.Printf("\n#### %d Reeves: %v\n", len(actions.Reeves), actions.Reeves)
		if len(actions.Keys) != 0 {
			fmt.Printf("#### %d Keys: %v\n", len(actions.Keys), actions.Keys)
		}
		if len(actions.Epinfo) != 0 {
			fmt.Printf("#### %d Endpoints: %v\n", len(actions.Epinfo), actions.Epinfo)
		}
	*/
	return &actions, nil
}

// getSubquery - provides the subquery string for the 6 types of queries, each of which has a to/from
// component. The boolen is used to have the subquery return the principal instead of a client
// or endpoint uuid, used when the subquery is used to find a reeeve
func getSubquery(service, horde, table string, tick int, kind string, principal bool) (string, *c.Err) {
	statetable := ""
	uidtype := ""
	selecttype := ""
	if table == "endpoint" {
		statetable = "epstate"
		uidtype = "endpointuuid"
		selecttype = uidtype
	}
	if table == "client" {
		statetable = "clstate"
		uidtype = "clientuuid"
		selecttype = uidtype
	}
	if statetable == "" {
		return "", c.ErrF("Malformed table %s passed to getSubquery", table)
	}
	if principal {
		selecttype = "principal"
	}

	joinqpre := "(SELECT " + table + "." + selecttype + " FROM " + table + " INNER JOIN " + statetable +
		" ON " + table + "." + uidtype + " = " + statetable + "." + uidtype + " WHERE " +
		table + ".servicename = '" + service + "' AND " + table + ".hordename = '" + horde + "' AND "

	subq := ""
	switch kind {
	case "oldendpointsto", "oldclientsfrom":
		subq = joinqpre + statetable + ".addstate < " + fmt.Sprintf("%d", tick) + " AND " + statetable + ".delstate = 0 )"
	case "oldendpointsfrom", "oldclientsto", "newendpointsto", "newclientsfrom":
		subq = joinqpre + statetable + ".addstate = " + fmt.Sprintf("%d", tick) + " AND " + statetable + ".delstate = 0 )"
	case "newendpointsfrom", "newclientsto", "delendpointsto", "delclientsfrom":
		subq = joinqpre + statetable + ".addstate <= " + fmt.Sprintf("%d", tick) + " AND " + statetable + ".delstate = 0 )"
	case "delendpointsfrom", "delclientsto":
		// note - we cannot del anything added in state 0 (nothing exists), so 0 is always excluded.
		subq = joinqpre + statetable + ".delstate = " + fmt.Sprintf("%d", tick) + " AND " + statetable + ".delstate > 0 )"
	default:
		return "", c.ErrF("Bad subquery kind in queryRuleTickX: %s", kind)
	}
	return subq, nil
}

// reevesFromSubquery - Returns the netids for all the reeves found that handle
// the clients/endpoints provided in a given subquery.
func reevesFromSubquery(db *sql.DB, table, subquery string) ([]idutils.NetIDT, *c.Err) {
	prequery := "SELECT servicerev, principal, address FROM endpoint WHERE endpoint.principal IN "
	postquery := " AND endpoint.servicename = 'Reeve'"
	query := prequery + subquery + postquery
	// fmt.Println(query)
	rows, derr := db.Query(query)
	if derr != nil {
		return nil, c.ErrF("query in reevesFromSubquery is malformed - %v", derr)
	}
	defer rows.Close()
	var nidlist []idutils.NetIDT
	for rows.Next() {
		var servicerev string
		var principal string
		var address string
		derr = rows.Scan(&servicerev, &principal, &address)
		if derr != nil {
			return nil, c.ErrF("scan in reevesFromSubquery -  %v", derr)
		}
		nid, nerr := idutils.NewNetID(servicerev, principal, idutils.SplitHost(address), idutils.SplitPort(address))
		if nerr != nil {
			return nil, c.ErrF("NewNetID in reevesFromSubquery -  %v", nerr)
		}
		nidlist = append(nidlist, nid)
	}
	derr = rows.Err()
	if derr != nil {
		return nil, c.ErrF("rows in reevesFromSubquery - %v", derr)
	}
	return nidlist, nil
}

// keysFromClients - Returns the pubkeys (as array of json strings) found
// in clients provided in a given subquery.
func keysFromClients(db *sql.DB, subquery string) ([]string, *c.Err) {
	prequery := "SELECT pubkey FROM client WHERE clientuuid IN "
	query := prequery + subquery
	// fmt.Println(query)
	rows, derr := db.Query(query)
	if derr != nil {
		return nil, c.ErrF("query in keysFromClients is malformed - %v", derr)
	}
	defer rows.Close()
	var keylist []string
	for rows.Next() {
		var pubkey string
		derr = rows.Scan(&pubkey)
		if derr != nil {
			return nil, c.ErrF("scan in keysFromClients -  %v", derr)
		}
		if pubkey == "" {
			return nil, c.ErrF("empty pubkey in keysFromClients")
		}
		keylist = append(keylist, pubkey)
	}
	derr = rows.Err()
	if derr != nil {
		return nil, c.ErrF("rows in keysFromClients - %v", derr)
	}
	return keylist, nil
}

// infoFromEndpoints - Returns the EpInfo (nodeid, netid) found
// in endpoints provided in the given subquery
func infoFromEndpoints(db *sql.DB, subquery string) ([]pb.EpInfo, *c.Err) {
	prequery := "SELECT blocname, hordename, nodename, servicename, serviceapi, servicerev, principal, address FROM endpoint WHERE endpointuuid IN "
	query := prequery + subquery
	// fmt.Println(query)
	rows, derr := db.Query(query)
	if derr != nil {
		return nil, c.ErrF("query in infoFromEndpoints is malformed - %v", derr)
	}
	defer rows.Close()
	var epinfolist []pb.EpInfo
	for rows.Next() {
		var blocname string
		var hordename string
		var nodename string
		var servicename string
		var serviceapi string
		var servicerev string
		var principal string
		var address string
		derr = rows.Scan(&blocname, &hordename, &nodename, &servicename, &serviceapi,
			&servicerev, &principal, &address)
		if derr != nil {
			return nil, c.ErrF("scan in infoFromEndpoints -  %v", derr)
		}
		nid, nerr := idutils.NewNetID(servicerev, principal, idutils.SplitHost(address), idutils.SplitPort(address))
		if nerr != nil {
			return nil, c.ErrF("NewNetID in infoFromEndpoints -  %v", nerr)
		}
		nod, ferr := idutils.NewNodeID(blocname, hordename, nodename, servicename, serviceapi)
		if ferr != nil {
			return nil, c.ErrF("NewNodeID in infoFromEndpoints -  %v", ferr)
		}
		epinfo := pb.EpInfo{}
		epinfo.Nodeid = nod.String()
		epinfo.Netid = nid.String()
		epinfolist = append(epinfolist, epinfo)
	}
	derr = rows.Err()
	if derr != nil {
		return nil, c.ErrF("rows in infoFromEndpoints - %v", derr)
	}
	return epinfolist, nil
}

/*
NOTES:

Some raw SQL used to explore define the queries encoded here, You can try out from a
trapped copy of steward.db included in the test code, and sqlite3 command line.

// Form of the subquery that returns a uuid:
SELECT endpoint.endpointuuid FROM endpoint
INNER JOIN epstate ON endpoint.endpointuuid = epstate.endpointuuid
  WHERE endpoint.servicename = 'pastiche' AND endpoint.hordename = 'sharks' AND epstate.curstate = 1;
9664fb5c-d268-11e8-aae1-0242ac12000b
9664fedf-d268-11e8-aae1-0242ac12000b
9664ff02-d268-11e8-aae1-0242ac12000b
96716262-d268-11e8-aae1-0242ac12000b

// Subquery returning principal used in outer query to return the reeve
SELECT * FROM endpoint
WHERE endpoint.principal IN
	(SELECT endpoint.principal FROM endpoint
	INNER JOIN epstate ON endpoint.endpointuuid = epstate.endpointuuid
	WHERE endpoint.servicename = 'pastiche' AND endpoint.hordename = 'sharks' AND epstate.curstate = 1)
AND endpoint.servicename = 'reeve';

9483d62c-d268-11e8-aae1-0242ac12000b|flock|sharks|f7|reeve|reeve0|reeve0.1|49XES1QOA8EXU5BUT8LMK7|f7:50059|2018-10-17T23:58:44.48Z|2018-10-17T23:58:44.48Z|||UP
9664fd86-d268-11e8-aae1-0242ac12000b|flock|sharks|f2|reeve|reeve0|reeve0.1|8U3VX33WJH9RCP4H4DOLLV|f2:50059|2018-10-17T23:58:47.64Z|2018-10-17T23:58:47.64Z|||UP
9664fe1d-d268-11e8-aae1-0242ac12000b|flock|sharks|f9|reeve|reeve0|reeve0.1|Y7XIIZ0GPXCTXMVNQID911|f9:50059|2018-10-17T23:58:47.64Z|2018-10-17T23:58:47.64Z|||UP
967161f7-d268-11e8-aae1-0242ac12000b|flock|sharks|f3|reeve|reeve0|reeve0.1|646A5VFYR35LLLDO8X5ADL|f3:50059|2018-10-17T23:58:47.72Z|2018-10-17T23:58:47.72Z|||UP

// As above, but constraining results to just what is needed for a NetID
SELECT servicerev, principal, address FROM endpoint
WHERE endpoint.principal IN
	(SELECT endpoint.principal FROM endpoint
	 INNER JOIN epstate ON endpoint.endpointuuid = epstate.endpointuuid
	   WHERE endpoint.servicename = 'pastiche' AND endpoint.hordename = 'sharks' AND epstate.curstate = 1)
AND endpoint.servicename = 'reeve';
reeve0.1|49XES1QOA8EXU5BUT8LMK7|f7:50059
reeve0.1|8U3VX33WJH9RCP4H4DOLLV|f2:50059
reeve0.1|Y7XIIZ0GPXCTXMVNQID911|f9:50059
reeve0.1|646A5VFYR35LLLDO8X5ADL|f3:50059

// Returning pubkey json corresponding to a client subquery
SELECT pubkey FROM client
WHERE clientuuid IN
	(SELECT client.clientuuid FROM client
	 INNER JOIN clstate on client.clientuuid = clstate.clientuuid
	    WHERE client.servicename = 'pastiche' AND client.hordename = 'sharks'
            AND clstate.addstate = 1 AND clstate.delstate = 0);
{"service":"pastiche0.1","name":"49XES1QOA8EXU5BUT8LMK7","keyid":"/pastiche0.1/49XES1QOA8EXU5BUT8LMK7/keys/8f:f6:ea:dc:ca:27:7c:96:f0:ad:ec:a2:24:26:ed:2a","pubkey":"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBDM29CJEiVHNrvmj/l0qVjdOzf1qESYRz1otlH9XTEU 49XES1QOA8EXU5BUT8LMK7-ed25519","stateadded":0}
{"service":"pastiche0.1","name":"8U3VX33WJH9RCP4H4DOLLV","keyid":"/pastiche0.1/8U3VX33WJH9RCP4H4DOLLV/keys/48:2a:b4:24:01:eb:43:f0:e0:60:80:1c:2f:c5:1e:0e","pubkey":"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB66q9cw3Nmm14z8ailnttZakv3DY08SE8+meaXc7q5y 8U3VX33WJH9RCP4H4DOLLV-ed25519","stateadded":0}
{"service":"pastiche0.1","name":"Y7XIIZ0GPXCTXMVNQID911","keyid":"/pastiche0.1/Y7XIIZ0GPXCTXMVNQID911/keys/65:83:84:7a:87:ec:b4:9d:e6:7b:73:a2:14:cc:42:2e","pubkey":"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMeHFMeCWTKBqtxDSRRMpbpoVGnuPerxFA2S/Y/IRTbU Y7XIIZ0GPXCTXMVNQID911-ed25519","stateadded":0}
{"service":"pastiche0.1","name":"646A5VFYR35LLLDO8X5ADL","keyid":"/pastiche0.1/646A5VFYR35LLLDO8X5ADL/keys/e4:a0:65:4d:81:29:be:01:2c:e3:4c:39:71:f7:6a:cd","pubkey":"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBpsILBNu9qykVL7jM7mC7EsiurzR3EayvUqNDfj2bl6 646A5VFYR35LLLDO8X5ADL-ed25519","stateadded":0}


// Returning endpoint information corresponding to an endpoint subquery
SELECT blocname, hordename, nodename, servicename, serviceapi, servicerev, principal, address FROM endpoint
WHERE endpointuuid in
	(SELECT endpoint.endpointuuid FROM endpoint
         INNER JOIN epstate ON endpoint.endpointuuid = epstate.endpointuuid
	 WHERE  endpoicename = 'pastiche' AND endpoint.hordename = 'sharks'
	 AND epstate.addstate < 1 AND epstate.delstate = 0);
flock|sharks|f8|pastiche|pastiche0|pastiche0.1|O7Y0BKTP3DR1QVIR73JU1Q|f8:50051
flock|sharks|f6|pastiche|pastiche0|pastiche0.1|BWNB0H26L7C9KRHC0CEFHV|f6:50051
flock|sharks|f4|pastiche|pastiche0|pastiche0.1|VWPWM1LQ1SAF9L9H67MJBD|f4:50051
flock|sharks|f5|pastiche|pastiche0|pastiche0.1|GZ1X9HZTNCPFPSPZ0T85ZU|f5:50051


*/
