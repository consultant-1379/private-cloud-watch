// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// steward server utilities

package steward

import (
	"container/list"
	"database/sql"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	// Sqlite - comment required
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/clog"
	c "github.com/erixzone/crux/pkg/crux"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
	"github.com/erixzone/crux/pkg/muck"
	rb "github.com/erixzone/crux/pkg/registrydb"
)

// server - implements cruxgen.StewardServer
type server struct {
}

var stewardLogger clog.Logger

// Ingestor - our handle to the Ingest system
var Ingestor *Ingest

// Spewer - our handle to the Fanout system
var Spewer *Fanout

// PingTest -- Returns a PING for a PONG, PONG for a PING
// and a test grpc error code and message for any other value
func (s *server) PingTest(ctx context.Context, ping *pb.Ping) (*pb.Ping, error) {
	if ping.Value == pb.Pingu_PING {
		return &pb.Ping{Value: pb.Pingu_PONG}, nil
	}
	if ping.Value == pb.Pingu_PONG {
		return &pb.Ping{Value: pb.Pingu_PING}, nil
	}
	return nil, grpc.Errorf(codes.FailedPrecondition, "PingTest Error")
}

func (s *server) Heartbeat(context.Context, *pb.HeartbeatReq) (*pb.HeartbeatReply, error) {
	return nil, nil
}

// EndpointUpdate - processes EndpointUpdate grpc requests, arriving from reeve servers
func (s *server) EndpointUpdate(ctx context.Context, ep *pb.EndpointData) (*pb.Acknowledgement, error) {
	ack := pb.Acknowledgement{}
	ack.Ack = pb.Ack_WORKING
	// capture keyid from Authentication header in ctx
	reeveid, rerr := grpcsig.WhoSigned(ctx)
	// TODO is reeveid really a reeve server?

	if rerr != nil {
		ack.Ack = pb.Ack_FAIL
		// No state or request uuid response if reeve client keyid missing
		// but if so, should never even reach here
	} else {
		ack.Remoteuuid = Ingestor.IngestEndpoint(ep, reeveid)
		ack.State = Ingestor.GetState()
	}
	ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	msg := fmt.Sprintf("steward processing EndpointUpdate from %s", reeveid)
	pidstr, _ := grpcsig.GetPidTS()
	stewardLogger.Log("SEV", "INFO", "PID", pidstr, "TS", ack.Ts, msg)
	return &ack, nil
}

// ClientUpdate - processes ClientUpdate grpc requests, arriving from reeve servers
func (s *server) ClientUpdate(ctx context.Context, cli *pb.ClientData) (*pb.Acknowledgement, error) {
	ack := pb.Acknowledgement{}
	ack.Ack = pb.Ack_WORKING
	// capture keyid from Authentication header in ctx
	reeveid, rerr := grpcsig.WhoSigned(ctx)
	// TODO is reeveid really a reeve server?

	if rerr != nil {
		ack.Ack = pb.Ack_FAIL
		// No state or request uuid response if reeve client keyid missing
	} else {
		ack.Remoteuuid = Ingestor.IngestClient(cli, reeveid)
		ack.State = Ingestor.GetState()
	}
	ack.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.00Z07:00")
	msg := fmt.Sprintf("steward processing ClientUpdate from %s", reeveid)
	pidstr, _ := grpcsig.GetPidTS()
	stewardLogger.Log("SEV", "INFO", "PID", pidstr, "TS", ack.Ts, msg)
	return &ack, nil
}

var totalendpoints int
var totalclients int

/*
func dumpList(dlhead *list.List, clock rb.StateClock) {
	i := 0
	j := 0
	fmt.Printf("Dump of state %d: %v\n", clock.State, clock)
	// walk the list, count the number of endpoints, clients that are on it
	for e := dlhead.Front(); e != nil; e = e.Next() {
		ievent := &e.Value
		b := (*ievent).(rb.FB).BMe() // reflection free link type interrogation (returns the type byte)
		switch b {
		case rb.AmENDPOINT:
			i++
			// the comment below shows how to cast this, so I leave it here
			// fmt.Printf("%d. Endpoint %s\n",i, (*ievent).(*EpUpdate).TxUuid)
		case rb.AmCLIENT:
			j++
			// fmt.Printf("%d. Client %s\n",i, (*ievent).(*ClUpdate).TxUuid)
		default:
		}
	}
	fmt.Printf("Clients: %d, Endpoints: %d\n", j, i)
	totalendpoints = totalendpoints + i
	totalclients = totalclients + j
}
*/

func stewardLaunchError(msg string, logger clog.Logger) *c.Err {
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
	return c.ErrF(msg)
}

// Launch - starts up steward grpc service for organza
func Launch(nod idutils.NodeIDT, nid idutils.NetIDT, impif **grpcsig.ImplementationT, stopch *chan bool) *c.Err {
	if impif == nil {
		return c.ErrF("no implementation interface provided")
	}

	imp := *impif
	stewardLogger = imp.Logger
	msg := fmt.Sprintf("steward Launch() - starting steward server")
	pidstr, ts := grpcsig.GetPidTS()
	stewardLogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg)
	// Start steward server proper
	s := imp.NewServer()
	pb.RegisterStewardServer(s, &server{})
	grpc_prometheus.Register(s)
	lis, lerr := net.Listen("tcp", nid.Port)
	if lerr != nil {
		msg2 := fmt.Sprintf("steward Launch() failed - in net.Listen : %v", lerr)
		return stewardLaunchError(msg2, stewardLogger)
	}
	// Ready to serve
	go s.Serve(lis)
	// Put up the stoping function
	stopfn := func(server *grpc.Server, nod idutils.NodeIDT, nid idutils.NetIDT, logger clog.Logger, stop *chan bool) {
		msg1 := fmt.Sprintf("%s GracefulStop Service  %s", nod.String(), nid.String())
		pidstr, ts := grpcsig.GetPidTS()
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg1)
		<-*stop
		server.GracefulStop()
		lis.Close()
		msg2 := fmt.Sprintf("%s Service Stopped  %s", nod.String(), nid.String())
		logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg2)
	}
	go stopfn(s, nod, nid, stewardLogger, stopch)
	msg3 := fmt.Sprintf("%s Serving %s", nod.String(), nid.String())
	pidstr, ts = grpcsig.GetPidTS()
	stewardLogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg3)
	return nil
}

// StartSteward - starts up the steward() system
func StartSteward(nod idutils.NodeIDT, nid idutils.NetIDT, impif **grpcsig.ImplementationT) *c.Err {
	if impif == nil {
		return c.ErrF("no implementation interface passed to StartSteward")
	}
	imp := *impif

	stewardLogger = imp.Logger
	msg := fmt.Sprintf("StartSteward - starting steward server")
	pidstr, ts := grpcsig.GetPidTS()
	stewardLogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg)
	dbpath := muck.StewardDir() + "/steward.db"

	// Start up the Steward database and Ingestor
	// keep its previous contents
	// DB has its own logger
	dblog := clog.Log.With("focus", StewardRev, "mode", "steward-DB")
	derr := StartStewardDB(dbpath, dblog, false)
	if derr != nil {
		msg := fmt.Sprintf("StartStewardDB failed - %v", derr)
		pidstr, ts := grpcsig.GetPidTS()
		stewardLogger.Log("SEV", "FATAL", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%s", msg)
	}

	// Start up the Steward server itself
	serr := StartStewardServer(*imp, nid.Port)
	if serr != nil {
		msg := fmt.Sprintf("StartSteward failed - %v", serr)
		pidstr, ts := grpcsig.GetPidTS()
		stewardLogger.Log("SEV", "FATAL", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%s", msg)
	}
	msg = fmt.Sprintf("%s Serving %s", nod.String(), nid.String())
	pidstr, ts = grpcsig.GetPidTS()
	stewardLogger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg)
	return nil
}

// StartStewardServer - starts steward() service,
func StartStewardServer(imp grpcsig.ImplementationT, port string) *c.Err {
	if port == "" {
		return c.ErrF("StartStewardServer error - no port specified for reeve")
	}
	// Start  gRPC server with Interceptors for http-signatures inbound
	s := imp.NewServer()

	pb.RegisterStewardServer(s, &server{})
	grpc_prometheus.Register(s)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return c.ErrF("StartStewardServer error - net.Listen failed: %v", err)
	}
	go s.Serve(lis)
	return nil
}

// DB - our sqlite database
var DB *sql.DB
var dbguard sync.Mutex

// StartStewardDB - starts the Steward DB and Ingestor
func StartStewardDB(filepath string, logger clog.Logger, clear bool) *c.Err {
	dbguard.Lock()
	defer dbguard.Unlock()
	// Alloc and Start Ingestor at state = 1, timebox = 3 sec
	// if you start at state = 0, it doesn't get logged...
	Ingestor = StartNewIngest(1, 3*time.Second, DBList, logger)
	err := rb.InitializeRegistryDB(filepath, clear)
	if err != nil {
		Ingestor.Quit()
		msg := fmt.Sprintf("StartStewardDB failed on InitializeRegistryDB - %v", err)
		pidstr, ts := grpcsig.GetPidTS()
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%v", msg)
	}
	err = rb.RulesInit(filepath, "")
	if err != nil {
		Ingestor.Quit()
		msg := fmt.Sprintf("StartStewardDB failed on RulesInit : %v", err)
		pidstr, ts := grpcsig.GetPidTS()
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%v", msg)
	}
	var derr error
	DB, derr = sql.Open("sqlite3", filepath)
	if derr != nil {
		Ingestor.Quit()
		msg := fmt.Sprintf("StartStewardDB - Cannot open sqlite3 database : %v", derr)
		pidstr, ts := grpcsig.GetPidTS()
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		return c.ErrF("%v", msg)
	}
	Spewer = StartNewFanout(3*time.Second, logger)
	return nil
}

// StopStewardDB - stops the DB and ingestor goroutines
func StopStewardDB(debug bool) {
	dbguard.Lock()
	defer dbguard.Unlock()
	if debug {
		logger := Ingestor.logger
		err := rb.DumpEndpoints(DB)
		if err != nil {
			msg := fmt.Sprintf("StopStewardDB - DumpEndpoints database error %v", err)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		}
		err = rb.DumpEndpointStates(DB)
		if err != nil {
			msg := fmt.Sprintf("StopStewardDB - DumpEndpointStates database error %v", err)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		}
		err = rb.DumpClients(DB)
		if err != nil {
			msg := fmt.Sprintf("StopStewardDB - DumpClients database error %v", err)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		}
		err = rb.DumpClientStates(DB)
		if err != nil {
			msg := fmt.Sprintf("StopStewardDB - DumpClientStates database error %v", err)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		}
		err = rb.DumpDBErrors(DB) // for debug purposes
		if err != nil {
			msg := fmt.Sprintf("StopStewardDB - DumpDBErrors database error %v", err)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		}
		err = rb.DumpDBStateTime(DB) // for debug purposes
		if err != nil {
			msg := fmt.Sprintf("StopStewardDB - DumpDBtateTime database error %v", err)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg)
		}
	}
	Ingestor.logger.Log("SEV", "INFO", "stopping steward fanout")
	Spewer.Quit() // CWVH - stop in reverse order started. Stop Spewer event loop first
	// so it is not triggered
	// again by anything in a dangling process() activity from Ingestor.Quit()
	Ingestor.logger.Log("SEV", "INFO", "stopping steward injestor")
	Ingestor.Quit() // goroutine process() (DBist) may dangle - run to completion.
	Ingestor.logger.Log("SEV", "INFO", "stopping steward sqlite database")
	DB.Close()
}

// DBList - Ingestor Callback function -parses &  pushes data batch into sqlite DB
// When its part is done - it pushes the clock interval to the Spewer/Fanout
// Parse errors are marked in the BadRequest database.
// Database write errors are logged.
func DBList(dlhead *list.List, clock rb.StateClock) {
	var endpoints []rb.EndpointRow
	var clients []rb.ClientRow
	logger := Ingestor.logger
	// walk the list, count the number of endpoints, clients that are on it
	for e := dlhead.Front(); e != nil; e = e.Next() {
		ievent := &e.Value
		b := (*ievent).(rb.FB).BMe() // reflection free link type interrogation (returns the type byte)
		switch b {
		case rb.AmENDPOINT:
			ep := (*ievent).(*rb.EpUpdate)
			endpoint, err := rb.MakeEndpointRow(ep)
			if err != nil {
				msg1 := fmt.Sprintf("DBList MakeEndpointRow error %v", err)
				pidstr, ts := grpcsig.GetPidTS()
				logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg1)
				// PUSH TxUuid, error, clock data into epStateTable
				merr := rb.MarkBadRequest(DB, ep.TxUUID, clock.State, ep.ReeveKeyID, err)
				if merr != nil {
					msg2 := fmt.Sprintf("DBList MarkBadRequest error %v", merr)
					pidstr, ts := grpcsig.GetPidTS()
					logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg2)
				}
			} else {
				endpoints = append(endpoints, *endpoint)
			}
		case rb.AmCLIENT:
			cl := (*ievent).(*rb.ClUpdate)
			client, err := rb.MakeClientRow(cl)
			if err != nil {
				msg3 := fmt.Sprintf("DBList MakeClientRow error %v", err)
				pidstr, ts := grpcsig.GetPidTS()
				logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg3)
				merr := rb.MarkBadRequest(DB, cl.TxUUID, clock.State, cl.ReeveKeyID, err)
				if merr != nil {
					msg4 := fmt.Sprintf("DBList MarkBadRequest error %v", merr)
					pidstr, ts := grpcsig.GetPidTS()
					logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg4)
				}
			} else {
				clients = append(clients, *client)
			}
		default:
		}
	}

	// Do the DB inserts
	e := len(endpoints)
	c := len(clients)
	msg5 := fmt.Sprintf("DBlist Clients: %d, Endpoints: %d", c, e)
	pidstr, ts := grpcsig.GetPidTS()
	logger.Log("SEV", "INFO", "PID", pidstr, "TS", ts, msg5)
	totalendpoints = totalendpoints + e
	totalclients = totalclients + c

	if e > 0 {
		derr := rb.InsertEndpoints(DB, endpoints)
		if derr != nil {
			msg6 := fmt.Sprintf("DBList InsertEndpoints error : %v", derr)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg6)
		}
		// make EntryStateRow from endpoints.
		var esrows []rb.EntryStateRow
		for _, ep := range endpoints {
			es := rb.EntryStateRow{
				EntryUUID: ep.EndpointUUID,
				AddState:  int(clock.State),
				CurState:  int(clock.State),
			}
			esrows = append(esrows, es)
		}
		terr := rb.InsertEndpointStates(DB, esrows)
		if terr != nil {
			msg7 := fmt.Sprintf("DBList InsertEndpointStates error : %v", terr)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg7)
		}
	}
	if c > 0 {
		derr := rb.InsertClients(DB, clients)
		if derr != nil {
			msg8 := fmt.Sprintf("DBList InsertClients error : %v", derr)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg8)
		}

		var csrows []rb.EntryStateRow
		for _, cl := range clients {
			cl := rb.EntryStateRow{
				EntryUUID: cl.ClientUUID,
				AddState:  int(clock.State),
				CurState:  int(clock.State),
			}
			csrows = append(csrows, cl)
		}
		uerr := rb.InsertClientStates(DB, csrows)
		if uerr != nil {
			msg9 := fmt.Sprintf("DBList InsertClientStates error : %v", uerr)
			pidstr, ts := grpcsig.GetPidTS()
			logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg9)
		}
	}

	// Mark this complete in the database
	terr := rb.MarkStateTime(DB, clock)
	if terr != nil {
		msg10 := fmt.Sprintf("DBList MarkStateTime error : %v", terr)
		pidstr, ts := grpcsig.GetPidTS()
		logger.Log("SEV", "ERROR", "PID", pidstr, "TS", ts, msg10)
	}

	// Push clock update along to the fanout event loop.
	Spewer.NewInterval(clock)
}
