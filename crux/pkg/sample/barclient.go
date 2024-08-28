package sample

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
	"github.com/erixzone/crux/pkg/grpcsig"
	"github.com/erixzone/crux/pkg/idutils"
)

// PingSleep - blocks, pings with delay intervals, until we get a response from server
// or total time exceeds timeout
func PingSleep(client pb.BarClient, delay time.Duration, timeout time.Duration) error {
	var total time.Duration
	// Make a ping
	ping := &pb.Ping{Value: pb.Pingu_PING}
	// Ping until we get the server response
	for {
		resp, cerr := client.PingTest(context.Background(), ping)
		if cerr != nil {
			total = total + delay
			time.Sleep(delay)
		} else {
			if resp.Value == pb.Pingu_PONG {
				return nil
			}
		}
		if total > timeout {
			return fmt.Errorf("PingSleep blocking exceeded timeout; last error: %v", cerr)
		}
	}
}

// WakeUpBar - Client call - Dials Bar endpoint, does PingSleep every 1s.
// Returns nil when PingSleep works with an
// authenticated connection to Bar,
// Returns an error if PingSleep times out (10s), with the last gRPC error seen.
func WakeUpBar(signer *grpcsig.AgentSigner, nid idutils.NetIDT) error {
	conn, err := signer.Dial(nid.Address())
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("WakeUpBar - grpc.Dial to bar at %s failed : %v", nid.String(), err)
	}
	barcli := pb.NewBarClient(conn)
	return PingSleep(barcli, 1*time.Second, 10*time.Second)
}
