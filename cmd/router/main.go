package main

import (
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LeonardJouve/pointsman/api"
	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/env"
	"github.com/LeonardJouve/pointsman/provider"
)

func main() {
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

	routerAddr, ok := env.Get("ROUTER_ADDR")
	if !ok {
		log.Fatal("ROUTER_ADDR is required")
	}

	conn, err := grpc.NewClient(dispatcherAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	client := provider.New(baseURL, apiKey)
	dispatcher := dispatch.NewDispatcherClient(conn)

	handler := api.New(client, dispatcher).Handler()

	log.Fatal(http.ListenAndServe(routerAddr, handler))
}
