package main

import (
	"context"
	"flag"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LeonardJouve/pointsman/agent"
	"github.com/LeonardJouve/pointsman/api"
	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/env"
	"github.com/LeonardJouve/pointsman/loop"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

func main() {
	prompt := flag.String("prompt", "", "the session's user prompt")
	flag.Parse()

	if env.IsDevelopment() {
		if err := env.Load(".env"); err != nil {
			log.Fatal(err)
		}
	}

	apiKey, ok := env.Get("LLM_API_KEY")
	if !ok {
		log.Fatal("LLM_API_KEY is required")
	}

	baseURL, ok := env.Get("LLM_BASE_URL")
	if !ok {
		log.Fatal("LLM_BASE_URL is required")
	}

	dispatcherAddr, ok := env.Get("DISPATCHER_ADDR")
	if !ok {
		log.Fatal("DISPATCHER_ADDR is required")
	}

	conn, err := grpc.NewClient(dispatcherAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	client := provider.New(baseURL, apiKey)
	dispatcher := dispatch.NewDispatcherClient(conn)

	a := agent.New("", "fake", "", []string{"sleep"}, agent.Config{})

	s, err := session.New(&a)
	if err != nil {
		log.Fatal(err)
	}

	if err := s.AppendUser(*prompt); err != nil {
		log.Fatal(err)
	}

	if err := loop.Run(context.Background(), s, client, dispatcher); err != nil {
		log.Print(err)
	}

	for i, m := range s.Messages() {
		api.WriteMessage(os.Stdout, i, m)
	}

	api.WriteStatus(os.Stdout, s.Status())
}
