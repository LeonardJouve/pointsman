package main

import (
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/tool"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "listen address")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	read, err := tool.Read()
	if err != nil {
		log.Fatal(err)
	}

	write, err := tool.Write()
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	dispatch.RegisterDispatcherServer(s, dispatch.NewServer([]tool.Tool{read, write}))

	log.Fatal(s.Serve(listener))
}
