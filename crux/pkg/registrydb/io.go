// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package registrydb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pborman/uuid"

	pb "github.com/erixzone/crux/gen/cruxgen"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
)

// MakeEndpointRow - constructs a database row struct from netid, nodeid, timestamps it, gives it a uuid
// sets the status
func MakeEndpointRow(ep *EpUpdate) (*EndpointRow, *c.Err) {
	netid, err := idutils.NetIDParse(ep.Netid)
	if err != nil {
		return nil, c.ErrF("MakeEndpoint - %v", err)
	}
	if netid.Query == true {
		return nil, c.ErrF("invalid netid - cannot put query form of endpoint in database")
	}
	nodeid, ferr := idutils.NodeIDParse(ep.Nodeid)
	if ferr != nil {
		return nil, c.ErrF("MakeEndpoint - %v", ferr)
	}
	status := ep.Status
	nowts := fmt.Sprintf("%s", time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	epr := EndpointRow{
		EndpointUUID: uuid.NewUUID().String(),
		BlocName:     nodeid.BlocName,
		HordeName:    nodeid.HordeName,
		NodeName:     nodeid.NodeName,
		ServiceName:  nodeid.ServiceName,
		ServiceAPI:   nodeid.ServiceAPI,
		ServiceRev:   netid.ServiceRev,
		Principal:    netid.Principal,
		Address:      netid.Address(),
		StartedTS:    nowts,
		LastTS:       nowts,
		Status:       pb.ServiceState_name[int32(status)],
	}
	return &epr, nil
}

// InsertEndpoints - inserts an array of EndpointRow into database
func InsertEndpoints(db *sql.DB, endpoints []EndpointRow) *c.Err {
	tx, derr := db.Begin()
	if derr != nil {
		return c.ErrF("error - InsertEndpoints failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into endpoint(endpointuuid, blocname, hordename, nodename, servicename, serviceapi, servicerev, principal, address, startedts, lastts, expirests, removedts, status) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if derr != nil {
		return c.ErrF("error - InsertEndpoints failed Prepare - %v", derr)
	}
	defer stmt.Close()
	for _, endpt := range endpoints {
		_, derr = stmt.Exec(endpt.EndpointUUID,
			endpt.BlocName,
			endpt.HordeName,
			endpt.NodeName,
			endpt.ServiceName,
			endpt.ServiceAPI,
			endpt.ServiceRev,
			endpt.Principal,
			endpt.Address,
			endpt.StartedTS,
			endpt.LastTS,
			endpt.ExpiresTS,
			endpt.RemovedTS,
			endpt.Status)
		if derr != nil {
			return c.ErrF("error - InsertEndpoints failed Exec - %v", derr)
		}
	}
	txerr := tx.Commit()
	if txerr != nil {
		return c.ErrF("error - InsertEndpoints failed Commit -  %v", txerr)
	}
	return nil
}

// InsertEndpointStates -
func InsertEndpointStates(db *sql.DB, endpointstates []EntryStateRow) *c.Err {
	tx, derr := db.Begin()
	if derr != nil {
		return c.ErrF("error - InsertEndpointStates failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into epstate (endpointuuid, ruleid, addstate, delstate, curstate) values(?, ?, ?, ?, ?)")
	if derr != nil {
		return c.ErrF("error - InsertEndpointStates failed Prepare - %v", derr)
	}
	defer stmt.Close()
	for _, endptst := range endpointstates {
		_, derr = stmt.Exec(endptst.EntryUUID,
			endptst.RuleID,
			endptst.AddState,
			endptst.DelState,
			endptst.CurState)
		if derr != nil {
			return c.ErrF("error - InsertEndpointStates failed Exec - %v", derr)
		}
	}
	txerr := tx.Commit()
	if txerr != nil {
		return c.ErrF("error - InsertEndpointStates failed Commit -  %v", txerr)
	}
	return nil
}

// InsertClientStates -
func InsertClientStates(db *sql.DB, clientstates []EntryStateRow) *c.Err {
	tx, derr := db.Begin()
	if derr != nil {
		return c.ErrF("error - InsertClientStates failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into clstate (clientuuid, ruleid, addstate, delstate, curstate) values(?, ?, ?, ?, ?)")
	if derr != nil {
		return c.ErrF("error - InsertClientStates failed Prepare - %v", derr)
	}
	defer stmt.Close()
	for _, clientst := range clientstates {
		_, derr = stmt.Exec(clientst.EntryUUID,
			clientst.RuleID,
			clientst.AddState,
			clientst.DelState,
			clientst.CurState)
		if derr != nil {
			return c.ErrF("error - InsertClientStates failed Exec - %v", derr)
		}
	}
	txerr := tx.Commit()
	if txerr != nil {
		return c.ErrF("error - InsertClientStates failed Commit -  %v", txerr)
	}
	return nil
}

// MakeClientRow - constructs a database row struct from nodeid, pubkey, timestamps it, gives it a uuid
// sets the status
func MakeClientRow(cl *ClUpdate) (*ClientRow, *c.Err) {
	nodeid, ferr := idutils.NodeIDParse(cl.Nodeid)
	if ferr != nil {
		return nil, c.ErrF("MakeClientRow - %v", ferr)
	}
	keyinfo, kerr := idutils.KeyIDParse(cl.Keyid)
	if kerr != nil {
		return nil, c.ErrF("MakeClientRow - %v", kerr)
	}
	// json must be parseable
	pk, perr := grpcsig.PubKeyFromJSON([]byte(cl.Keyjson))
	if perr != nil {
		return nil, c.ErrF("MakeClientRow - %v", kerr)
	}
	// embedded keyid in json must be the same as provided
	if pk.KeyID != cl.Keyid {
		return nil, c.ErrF("MakeClientRow - keyid string mismatch with json pubkey")
	}
	status := cl.Status
	nowts := fmt.Sprintf("%s", time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	clr := ClientRow{
		ClientUUID:  uuid.NewUUID().String(),
		BlocName:    nodeid.BlocName,
		HordeName:   nodeid.HordeName,
		ServiceName: nodeid.ServiceName,
		ServiceAPI:  nodeid.ServiceAPI,
		ServiceRev:  keyinfo.ServiceRev,
		Principal:   keyinfo.Principal,
		KeyID:       pk.KeyID,
		PubKey:      cl.Keyjson,
		StartedTS:   nowts,
		LastTS:      nowts,
		Status:      pb.KeyStatus_name[int32(status)],
	}
	return &clr, nil
}

// InsertClients - inserts an array of ClientRow into database
func InsertClients(db *sql.DB, clients []ClientRow) *c.Err {
	tx, derr := db.Begin()
	if derr != nil {
		return c.ErrF("error - InsertClients failed Begin - %v", derr)
	}
	stmt, ierr := tx.Prepare("insert into client(clientuuid, blocname, hordename, servicename, serviceapi, servicerev, principal, keyid, pubkey, startedts, lastts, expirests, removedts, status) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if ierr != nil {
		return c.ErrF("error - InsertClients failed Prepare -  %v", ierr)
	}
	defer stmt.Close()
	for _, client := range clients {
		_, derr = stmt.Exec(client.ClientUUID,
			client.BlocName,
			client.HordeName,
			client.ServiceName,
			client.ServiceAPI,
			client.ServiceRev,
			client.Principal,
			client.KeyID,
			client.PubKey,
			client.StartedTS,
			client.LastTS,
			client.ExpiresTS,
			client.RemovedTS,
			client.Status)
		if derr != nil {
			return c.ErrF("error - InsertClients failed Exec -  %v", derr)
		}
	}
	txerr := tx.Commit()
	if txerr != nil {
		return c.ErrF("error - InsertClients failed Commit -  %v", txerr)
	}
	return nil
}

// GatherReeves - gathers all the reeves for outbound communication
func GatherReeves(db *sql.DB) (*[]pb.EndpointInfo, *c.Err) {
	var cat []catExtract
	// fmt.Println("*********BEGIN GatherReeves")
	// defer fmt.Println("*********END GatherReeves")
	rows, derr := db.Query("SELECT endpointuuid, blocname, hordename, nodename, servicename, serviceapi, servicerev, principal, address FROM endpoint WHERE servicename = 'Reeve' ORDER BY hordename")
	if derr != nil {
		return nil, c.ErrF("query in GatherReeves: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var endpointuuid string
		var blocname string
		var hordename string
		var nodename string
		var servicename string
		var serviceapi string
		var servicerev string
		var principal string
		var address string
		derr = rows.Scan(&endpointuuid, &blocname, &hordename, &nodename, &servicename, &serviceapi, &servicerev, &principal, &address)
		if derr != nil {
			return nil, c.ErrF("scan in GatherReeves: %v", derr)
		}
		// fmt.Println(blocname, hordename, nodename, servicename, serviceapi, servicerev, principal, address)
		cat = append(cat, catExtract{endpointuuid: endpointuuid, blocname: blocname, hordename: hordename,
			nodename: nodename, servicename: servicename, serviceapi: serviceapi, servicerev: servicerev,
			principal: principal, address: address})
	}
	derr = rows.Err()
	if derr != nil {
		return nil, c.ErrF("rows in GatherReeves: %v", derr)
	}
	// fmt.Printf("--\n%v\n--\n", cat)
	var epinfo []pb.EndpointInfo
	for _, item := range cat {
		fid, ferr := idutils.NewNodeID(item.blocname, item.hordename, item.nodename, item.servicename, item.serviceapi)
		if ferr != nil {
			return nil, c.ErrF("NewNodeID in GatherReeves: %v", ferr)
		}
		nid, nerr := idutils.NewNetID(item.servicerev, item.principal, idutils.SplitHost(item.address), idutils.SplitPort(item.address))
		if nerr != nil {
			return nil, c.ErrF("NewNetID in GatherReeves: %v", nerr)
		}
		epinfo = append(epinfo, pb.EndpointInfo{Nodeid: fid.String(), Netid: nid.String(), Filename: "n/a"})

	}
	// fmt.Printf("--\n%v\n--\n", epinfo)

	return &epinfo, nil
}

type catExtract struct {
	endpointuuid string
	blocname     string
	hordename    string
	servicename  string
	serviceapi   string
	servicerev   string
	nodename     string // - one system in this horde
	principal    string
	address      string
}

// CatHorde - catalog pertaining to a specific horde
type CatHorde struct {
	Hordename string
	Info      pb.CatalogInfo
}

// GatherCatalog - gathers the catalog for the flock services with running example netid
func GatherCatalog(db *sql.DB) (*[]CatHorde, *c.Err) {
	var cat []catExtract
	// fmt.Println("*********BEGIN GatherCatalog")
	// defer fmt.Println("*********END GatherCatalog")
	// We get one endpointuuid for an example
	rows, derr := db.Query("SELECT blocname, hordename, servicename, serviceapi, servicerev, max(endpointuuid) FROM endpoint GROUP BY blocname, hordename, servicename, serviceapi, servicerev ORDER BY hordename")
	if derr != nil {
		return nil, c.ErrF("query in GatherCatalog: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var blocname string
		var hordename string
		var servicename string
		var serviceapi string
		var servicerev string
		var endpointuuid string
		derr = rows.Scan(&blocname, &hordename, &servicename, &serviceapi, &servicerev, &endpointuuid)
		if derr != nil {
			return nil, c.ErrF("scan in GatherCatalog: %v", derr)
		}
		cat = append(cat, catExtract{blocname: blocname, hordename: hordename,
			servicename: servicename, serviceapi: serviceapi, servicerev: servicerev,
			endpointuuid: endpointuuid})
		// fmt.Println(blocname, hordename, servicename, serviceapi, servicerev, endpointuuid)
	}
	derr = rows.Err()
	if derr != nil {
		return nil, c.ErrF("rows in GatherCatalog: %v", derr)
	}

	// Get the example server info
	var catfilled []catExtract
	for _, ep := range cat {
		rows, derr := db.Query("SELECT nodename, principal, address FROM endpoint WHERE endpointuuid = '" + ep.endpointuuid + "'")
		if derr != nil {
			return nil, c.ErrF("query 2 in GatherCatalog: %v", derr)
		}
		defer rows.Close()
		for rows.Next() {
			var nodename string
			var principal string
			var address string
			derr = rows.Scan(&nodename, &principal, &address)
			if derr != nil {
				return nil, c.ErrF("scan 2 in GatherCatalog: %v", derr)
			}
			catfilled = append(catfilled, catExtract{blocname: ep.blocname, hordename: ep.hordename,
				servicename: ep.servicename, serviceapi: ep.serviceapi, servicerev: ep.servicerev,
				endpointuuid: ep.endpointuuid, nodename: nodename, principal: principal, address: address})
			// fmt.Println(ep.blocname, ep.endpointuuid, nodename, principal, address)
		}
		derr = rows.Err()
		if derr != nil {
			return nil, c.ErrF("rows 2 in GatherCatalog: %v", derr)
		}
	}

	// Package up with hordename exposed for downstream filtering
	//
	var cathorde []CatHorde
	for _, item := range catfilled {
		fid, ferr := idutils.NewNodeID(item.blocname, item.hordename, item.nodename, item.servicename, item.serviceapi)
		if ferr != nil {
			return nil, c.ErrF("NewNodeID in GatherCatalog: %v", ferr)
		}
		nid, nerr := idutils.NewNetID(item.servicerev, item.principal, idutils.SplitHost(item.address), idutils.SplitPort(item.address))
		if nerr != nil {
			return nil, c.ErrF("NewNetID in GatherCatalog: %v", nerr)
		}
		catinfo := pb.CatalogInfo{Nodeid: fid.String(), Netid: nid.String(), Filename: "n/a"}
		cathorde = append(cathorde, CatHorde{Hordename: item.hordename, Info: catinfo})
	}

	// fmt.Printf("--\n%v\n--\n", cathorde)
	return &cathorde, nil
}

// DumpClients - dumps selected fields from client table
func DumpClients(db *sql.DB) *c.Err {
	// fmt.Println("*********BEGIN DumpClients")
	// defer fmt.Println("*********END DumpClients")
	rows, derr := db.Query("SELECT clientuuid, serviceapi, keyid, hordename FROM client")
	if derr != nil {
		return c.ErrF("query in DumpClients: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var clientuuid string
		var serviceapi string
		var keyid string
		var hordename string
		derr = rows.Scan(&clientuuid, &serviceapi, &keyid, &hordename)
		if derr != nil {
			return c.ErrF("scan in DumpClients: %v", derr)
		}
		// fmt.Println(clientuuid, serviceapi, keyid, hordename)
	}
	derr = rows.Err()
	if derr != nil {
		return c.ErrF("rows in DumpClients: %v", derr)
	}
	return nil
}

// DumpEndpoints - dumps selected fields from endpoint table
func DumpEndpoints(db *sql.DB) *c.Err {
	// fmt.Println("*********BEGIN DumpEndpoints")
	// defer fmt.Println("*********END DumpEndpoints")
	rows, derr := db.Query("SELECT endpointuuid, principal, address, servicename, servicerev, hordename FROM endpoint")
	if derr != nil {
		return c.ErrF("query in DumpEndpoints: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var endpointuuid string
		var principal string
		var address string
		var servicename string
		var servicerev string
		var hordename string
		derr = rows.Scan(&endpointuuid, &principal, &address, &servicename, &servicerev, &hordename)
		if derr != nil {
			return c.ErrF("scan in DumpEndpoints: %v", derr)
		}
		// fmt.Println(endpointuuid, hordename, servicerev, principal, address, servicename)
	}
	derr = rows.Err()
	if derr != nil {
		return c.ErrF("rows in DumpEndpoints: %v", derr)
	}
	return nil
}

// MarkBadRequest - provides a place for errors to land for eventually consistent
// resolution of client or endpoint updates - as they do not land in the main tables
func MarkBadRequest(db *sql.DB, txuuid string, state int32, keyid string, err *c.Err) *c.Err {
	// fmt.Println("**********BEGIN MarkBadRequest")
	// defer fmt.Println("**********END MarkBadRequest")
	tx, derr := db.Begin()
	if derr != nil {
		return c.ErrF("error - MarkBadRequest failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into badrequest(requestuuid, keyid, stateno, requestts, error) values(?, ?, ?, ?, ?)")
	if derr != nil {
		return c.ErrF("error - MarkBadRequest failed Prepare - %v", derr)
	}
	defer stmt.Close()
	nowts := fmt.Sprintf("%s", time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	_, derr = stmt.Exec(txuuid, keyid, state, nowts, err.String())
	if derr != nil {
		return c.ErrF("error - MarkBadRequest failed Exec - %v", derr)
	}
	txerr := tx.Commit()
	if txerr != nil {
		return c.ErrF("error - MarkBadRequest failed Commit - %v", txerr)
	}
	return nil
}

// DumpDBErrors - debug utility for test to see errors in above noted table
func DumpDBErrors(db *sql.DB) *c.Err {
	fmt.Println("*********BEGIN DumpDBErrors")
	defer fmt.Println("*********END DumpDBErrors")
	rows, derr := db.Query("SELECT requestuuid, keyid, stateno, requestts, error FROM badrequest")
	if derr != nil {
		return c.ErrF("query in DumpDBErrors: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var requestuuid string
		var keyid string
		var stateno int32
		var requestts string
		var error string
		derr = rows.Scan(&requestuuid, &keyid, &stateno, &requestts, &error)
		if derr != nil {
			return c.ErrF("scan in DumpDBErrors: %v", derr)
		}
		fmt.Println(requestuuid, keyid, stateno, requestts, error)
	}
	derr = rows.Err()
	if derr != nil {
		return c.ErrF("rows in DumpDBErrors: %v", derr)
	}
	return nil
}

// MarkStateTime - pushes the StateClock into its table, signalling its work is done
func MarkStateTime(db *sql.DB, clock StateClock) *c.Err {
	// fmt.Println("****BEGIN MarkStateTimeRequest")
	// defer fmt.Println("****END MarkStateTimeRequest")
	tx, derr := db.Begin()
	if derr != nil {
		return c.ErrF("error - MarkStateTime failed Begin - %v", derr)
	}
	stmt, derr := tx.Prepare("insert into statetime(stateno, starts, endts) values(?, ?, ?)")
	if derr != nil {
		return c.ErrF("error - MarkStateTime failed Prepare - %v", derr)
	}
	defer stmt.Close()
	starts := fmt.Sprintf("%s", clock.Begin.UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	endts := fmt.Sprintf("%s", clock.End.UTC().Format("2006-01-02T15:04:05.00Z07:00"))
	_, derr = stmt.Exec(clock.State, starts, endts)
	if derr != nil {
		return c.ErrF("error - MarkStateTime failed Exec - %v", derr)
	}
	txerr := tx.Commit()
	if txerr != nil {
		return c.ErrF("error - MarkStateTimet failed Commit - %v", txerr)
	}
	return nil
}

// DumpDBStateTime - debug utility for test to dump the statetime table
func DumpDBStateTime(db *sql.DB) *c.Err {
	fmt.Println("*********BEGIN DumpStateTime")
	defer fmt.Println("*********END DumpStateTime")
	rows, derr := db.Query("SELECT stateno, starts, endts FROM statetime")
	if derr != nil {
		return c.ErrF("query in DumpDBStateTime: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var stateno int32
		var starts string
		var endts string
		derr = rows.Scan(&stateno, &starts, &endts)
		if derr != nil {
			return c.ErrF("scan in DumpDBStateTime: %v", derr)
		}
		fmt.Println(stateno, starts, endts)
	}
	derr = rows.Err()
	if derr != nil {
		return c.ErrF("rows in DumpDBStateTime: %v", derr)
	}
	return nil
}

// DumpClientStates - wip test for db query
func DumpClientStates(db *sql.DB) *c.Err {
	fmt.Println("*********BEGIN DumpClientStates")
	defer fmt.Println("*********END DumpClientStates")
	rows, derr := db.Query("SELECT entryno, clientuuid, ruleid, addstate, delstate, curstate FROM clstate")
	if derr != nil {
		return c.ErrF("query in DumpClientState: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var entryno string
		var clientuuid string
		var ruleid string
		var addstate string
		var delstate string
		var curstate string
		derr = rows.Scan(&entryno, &clientuuid, &ruleid, &addstate, &delstate, &curstate)
		if derr != nil {
			return c.ErrF("scan in DumpClientStates: %v", derr)
		}
		fmt.Println(entryno, clientuuid, ruleid, addstate, delstate, curstate)
	}
	derr = rows.Err()
	if derr != nil {
		return c.ErrF("rows in DumpClientStates: %v", derr)
	}
	return nil
}

// DumpEndpointStates - wip test for db query
func DumpEndpointStates(db *sql.DB) *c.Err {
	fmt.Println("*********BEGIN DumpEndpointStates")
	defer fmt.Println("*********END DumpEndpointStates")
	rows, derr := db.Query("SELECT entryno, endpointuuid, ruleid, addstate, delstate, curstate FROM epstate")
	if derr != nil {
		return c.ErrF("query in DumpEndpointState: %v", derr)
	}
	defer rows.Close()
	for rows.Next() {
		var entryno string
		var endpointuuid string
		var ruleid string
		var addstate string
		var delstate string
		var curstate string
		derr = rows.Scan(&entryno, &endpointuuid, &ruleid, &addstate, &delstate, &curstate)
		if derr != nil {
			return c.ErrF("scan in DumpEndpointState: %v", derr)
		}
		fmt.Println(entryno, endpointuuid, ruleid, addstate, delstate, curstate)
	}
	derr = rows.Err()
	if derr != nil {
		return c.ErrF("rows in DumpEndpointState: %v", derr)
	}
	return nil
}
