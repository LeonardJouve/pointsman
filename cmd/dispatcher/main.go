package main

import (
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/LeonardJouve/pointsman/dispatch"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "listen address")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	dispatch.RegisterDispatcherServer(s, dispatch.NewServer(nil))

	log.Fatal(s.Serve(listener))
}
