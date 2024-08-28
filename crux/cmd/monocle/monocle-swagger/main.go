package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	gw "github.com/erixzone/crux/gen/cruxgen"
)

var (
	monocleEndpoint = flag.String("monocle_endpoint", "localhost:9090", "Monocle enpoint to forwrad to")
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	fmt.Printf("Use monocle_endpoint flag if your monocle grpc server isn't at : %s\n", *monocleEndpoint)
	err := gw.RegisterMonocleHandlerFromEndpoint(ctx, mux, *monocleEndpoint, opts)
	if err != nil {
		return err
	}

	port := ":8080"
	fmt.Printf("Monocle REST-to-GRPC  proxy listening on : %s\n", port)

	return http.ListenAndServe(port, mux)
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Printf("Monocle REST server failed: %s\n", err)
	}
}
