// (c) Ericsson AB 2018 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package steward

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pb "github.com/erixzone/crux/gen/cruxgen"
)

// PingSleep - blocks, pings with delay intervals, until we get a response from server
// or total time exceeds timeout
func PingSleep(client pb.StewardClient, delay time.Duration, timeout time.Duration) error {
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
			return fmt.Errorf("PingIt blocking exceeded timeout")
		}
	}
}
