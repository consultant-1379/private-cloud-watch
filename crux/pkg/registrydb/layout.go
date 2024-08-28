// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package registrydb

import (
	"database/sql"

	c "github.com/erixzone/crux/pkg/crux"
)

const endpointTableSQL = `
CREATE TABLE IF NOT EXISTS endpoint (endpointuuid TEXT NOT NULL PRIMARY KEY, blocname TEXT, hordename TEXT,
	nodename TEXT, servicename TEXT, serviceapi TEXT, servicerev TEXT, principal TEXT,
	address TEXT, startedts TEXT, lastts TEXT, expirests TEXT, removedts TEXT,
	status TEXT);`

const deleteFromEndpoint = `
	DELETE FROM endpoint;`

const clientTableSQL = `
CREATE TABLE IF NOT EXISTS client (clientuuid TEXT NOT NULL PRIMARY KEY, blocname TEXT, hordename TEXT,
	servicename TEXT, serviceapi TEXT, servicerev TEXT, principal TEXT,
	keyid TEXT, pubkey TEXT, startedts TEXT, lastts TEXT, expirests TEXT, removedts TEXT,
	status TEXT);`

const deleteFromClient = `
	DELETE FROM client;`

const allowedTableSQL = `
CREATE TABLE IF NOT EXISTS allowed (ruleuuid TEXT NOT NULL PRIMARY KEY, ruleid TEXT, epgroupid TEXT, clgroupid TEXT, ownerid TEXT);`

const deleteFromAllowed = `
	DELETE FROM allowed;`

const clGroupTableSQL = `
CREATE TABLE IF NOT EXISTS clgroup (clgroupid TEXT NOT NULL PRIMARY KEY, servicename TEXT, hordename TEXT);`

const deleteFromCLGroup = `
	DELETE FROM clgroup;`

const epGroupTableSQL = `
CREATE TABLE IF NOT EXISTS epgroup (epgroupid TEXT NOT NULL PRIMARY KEY, servicename TEXT, hordename TEXT);`

const deleteFromEPGroup = `
	DELETE FROM epgroup;`

const stateTimeTableSQL = `
CREATE TABLE IF NOT EXISTS statetime (stateno INT NOT NULL PRIMARY KEY, starts TEXT, endts TEXT);`

const deleteFromStateTime = `
	DELETE FROM statetime;`

const epStateTableSQL = `
CREATE TABLE IF NOT EXISTS epstate (entryno INTEGER PRIMARY KEY, endpointuuid TEXT, ruleid TEXT, addstate INT, delstate INT, curstate INT);`

const deleteFromEPState = `
	DELETE FROM epstate;`

const clStateTableSQL = `
CREATE TABLE IF NOT EXISTS clstate (entryno INTEGER PRIMARY KEY, clientuuid TEXT, ruleid TEXT, addstate INT, delstate INT, curstate INT);`

const deleteFromCLState = `
	DELETE FROM clstate;`

const badRequestTableSQL = `
CREATE TABLE IF NOT EXISTS badrequest (requestuuid TEXT NOT NULL PRIMARY KEY, keyid TEXT, stateno INT, requestts TEXT, error TEXT);`

const deleteFromBadRequest = `
	DELETE FROM badrequest;`

func makeEndpointTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := endpointTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromEndpoint
	}
	return execE(db, sqlCmd)
}

func makeEPGroupTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := epGroupTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromEPGroup
	}
	return execE(db, sqlCmd)
}

func makeEPStateTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := epStateTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromEPState
	}
	return execE(db, sqlCmd)
}

func makeClientTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := clientTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromClient
	}
	return execE(db, sqlCmd)
}

func makeCLGroupTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := clGroupTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromCLGroup
	}
	return execE(db, sqlCmd)
}

func makeCLStateTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := clStateTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromCLState
	}
	return execE(db, sqlCmd)
}

func makeAllowedTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := allowedTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromAllowed
	}
	return execE(db, sqlCmd)
}

func makeStateTimeTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := stateTimeTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromStateTime
	}
	return execE(db, sqlCmd)
}

func makeBadRequestTable(db *sql.DB, clear bool) *c.Err {
	sqlCmd := badRequestTableSQL
	if clear {
		sqlCmd = sqlCmd + deleteFromBadRequest
	}
	return execE(db, sqlCmd)
}

func execE(db *sql.DB, sqlCmd string) *c.Err {
	_, err := db.Exec(sqlCmd)
	if err != nil {
		return c.ErrF("%v", err)
	}
	return nil
}

func newRegistryDB(filename string) (*sql.DB, *c.Err) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, c.ErrF("%v", err)
	}

	return db, nil
}

// InitializeRegistryDB - initializes SQLite database file with our tables,
// when clear = true, clears out the existing database databae data if file already exists
func InitializeRegistryDB(dbfile string, clear bool) *c.Err {
	// Set up the file in SQLite
	db, cerr := newRegistryDB(dbfile)
	if cerr != nil {
		return c.ErrF("Could not initialize sqlite database file %s : %v", dbfile, cerr)
	}

	// Make all the tables (cleared out) with "CREATE TABLE IF NOT EXISTS"
	// and DELETE FROM
	err := makeEndpointTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize Endpoint table: %v", err)
	}

	err = makeEPGroupTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize EPGroup table: %v", err)
	}

	err = makeEPStateTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize EPState table: %v", err)
	}

	err = makeClientTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize Client table: %v", err)
	}

	err = makeCLGroupTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize CLGroup table: %v", err)
	}

	err = makeCLStateTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize CLState table: %v", err)
	}

	err = makeAllowedTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize Allowed table: %v", err)
	}

	err = makeStateTimeTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize StateTime table: %v", err)
	}

	err = makeBadRequestTable(db, clear)
	if err != nil {
		return c.ErrF("Could not initialize BadRequest table: %v", err)
	}

	db.Close()
	return nil
}

// RulesInit -  TEMPORARY for testing
// Thoughts - To flesh out - make a "cruxadmin" keypair in
// an admin package. Don't check into git (.gitignore)
// the resulting keypair.  Script into makefile
// that makes the keypair if it does not exist
// package up the public key into steward.
// Sign the allowed.json file - create a
// allowed_signed.json file containing the
// rules. and a json pubkey for steward to
// read in at startup and add to its whitelist.
// At runtime on steward, read the allowed_signed.json
// file and check the signature with the pubkey.
// Provide tooling to sign mods to the allowed.json file.
// Provide a grpc client to talk to steward
// and provide updates to allowed.json, where
// the grpcsig signature comes from the
// cruxadmin private key.
// keyid: /steward/cruxadmin/key/fingerprint
// Provide ways to add/delete/replace rules.
func RulesInit(dbfile, rulefile string) *c.Err {
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		return c.ErrF("in RulesInit - sql.Open failed : %v", err)
	}
	defer db.Close()

	// TODO This is temporary with hard coded hordes for test cases.
	// Work this out into a system that loads a signed bit of json
	// rules, derr := dobaserules(db)
	derr := dobaserules(db)
	if derr != nil {
		return c.ErrF("Could not initialize rules and allowed table: %v", derr)
	}
	return nil
}
