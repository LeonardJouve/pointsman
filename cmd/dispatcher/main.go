package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/env"
	"github.com/LeonardJouve/pointsman/tool"
	"github.com/LeonardJouve/pointsman/workspace"
)

func main() {
	if env.IsDevelopment() {
		if err := env.Load(".env"); err != nil {
			log.Fatal(err)
		}
	}

	addr, ok := env.Get("DISPATCHER_ADDR")
	if !ok {
		log.Fatal("DISPATCHER_ADDR is required")
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	tools, err := tool.All()
	if err != nil {
		log.Fatal(err)
	}

	workspaces, err := workspace.Load()
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	dispatch.RegisterDispatcherServer(s, dispatch.NewServer(tools, workspaces))

	log.Fatal(s.Serve(listener))
}
