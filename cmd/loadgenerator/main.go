package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LeonardJouve/pointsman/env"
)

const (
	sessions = 1000
	output   = "latency.csv"
)

type result struct {
	latency time.Duration
	status  string
}

func main() {
	if env.IsDevelopment() {
		if err := env.Load(".env"); err != nil {
			log.Fatal(err)
		}
	}

	addr, ok := env.Get("ROUTER_ADDR")
	if !ok {
		log.Fatal("ROUTER_ADDR is required")
	}

	router := "http://" + addr

	agent, err := post(router, "/agents", map[string]any{"model": "fake", "prompt": "", "tools": []string{"sleep"}})
	if err != nil {
		log.Fatal(err)
	}

	results := make([]result, sessions)
	var wg sync.WaitGroup
	start := time.Now()
	for i := range sessions {
		wg.Go(func() {
			results[i] = measure(router, agent, i)
		})
	}
	wg.Wait()
	wall := time.Since(start)

	writeCSV(results)
	summarize(results, wall)
}

// measure runs session i and returns its latency; a failed request is status "error".
func measure(router, agent string, i int) result {
	start := time.Now()
	status, err := run(router, agent, i)
	if err != nil {
		status = "error"
	}

	return result{latency: time.Since(start), status: status}
}

// run creates the session and reads its events until the status frame; it returns that status.
func run(router, agent string, i int) (string, error) {
	id, err := post(router, "/sessions", map[string]any{"agent": agent, "prompt": strconv.Itoa(i)})
	if err != nil {
		return "", err
	}

	resp, err := http.Get(router + "/sessions/" + id + "/events")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("events: %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if scanner.Text() != "event: status" {
			continue
		}
		scanner.Scan()

		var event struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal([]byte(strings.TrimPrefix(scanner.Text(), "data: ")), &event)

		return event.Status, err
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("events: stream ended without a status")
}

// post sends body as JSON and returns the id of the reply.
func post(router, path string, body any) (string, error) {
	payload, _ := json.Marshal(body)
	resp, err := http.Post(router+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("POST %s: %s", path, resp.Status)
	}

	var reply struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&reply)

	return reply.ID, err
}

func writeCSV(results []result) {
	file, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	w.Write([]string{"session", "latency_ms", "status"})
	for i, r := range results {
		w.Write([]string{strconv.Itoa(i), strconv.FormatInt(r.latency.Milliseconds(), 10), r.status})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func summarize(results []result, wall time.Duration) {
	failed, errored := 0, 0
	latencies := make([]time.Duration, len(results))
	for i, r := range results {
		latencies[i] = r.latency
		switch r.status {
		case "failed":
			failed++
		case "error":
			errored++
		}
	}
	slices.Sort(latencies)

	n := len(latencies)
	round := 100 * time.Millisecond
	fmt.Printf("sessions %d  failed %d  error %d  wall %v\n", n, failed, errored, wall.Round(round))
	fmt.Printf("latency  min %v  median %v  p99 %v  max %v\n", latencies[0].Round(round), latencies[n/2].Round(round), latencies[n*99/100].Round(round), latencies[n-1].Round(round))
}
