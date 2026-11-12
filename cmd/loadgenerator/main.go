package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LeonardJouve/pointsman/env"
)

const (
	output = "latency.csv"
	binary = "./singlesession"
)

type result struct {
	overhead time.Duration
	status   string
}

type message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls"`
}

type toolCall struct {
	Function function `json:"function"`
}

type function struct {
	Arguments string `json:"arguments"`
}

// the reply text of the fake model
type inference struct {
	Ms int `json:"inference_ms"`
}

type sleepArguments struct {
	Ms int `json:"ms"`
}

type statusEvent struct {
	Status string `json:"status"`
}

type idResponse struct {
	ID string `json:"id"`
}

// process is a started command's stdout; Close reaps the command.
type process struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (p process) Close() error {
	return p.cmd.Wait()
}

func main() {
	sessions := flag.Int("n", 200, "number of sessions started at once")
	single := flag.Bool("single", false, "run each session as a ./singlesession process instead of on the router")
	flag.Parse()

	if env.IsDevelopment() {
		if err := env.Load(".env"); err != nil {
			log.Fatal(err)
		}
	}

	open := openProcess
	if !*single {
		open = connectRouter()
	}

	results := make([]result, *sessions)
	var wg sync.WaitGroup
	start := time.Now()
	for i := range *sessions {
		wg.Go(func() {
			results[i] = measure(open, i)
		})
	}
	wg.Wait()
	wall := time.Since(start)

	if *single {
		removeStores()
	}

	writeCSV(results)
	summarize(results, wall)
}

// connectRouter creates the agent on the router and returns a function that opens session i on it.
func connectRouter() func(int) (io.ReadCloser, error) {
	addr, ok := env.Get("ROUTER_ADDR")
	if !ok {
		log.Fatal("ROUTER_ADDR is required")
	}

	router := "http://" + addr

	agent, err := post(router, "/agents", map[string]any{"model": "fake", "prompt": "", "tools": []string{"sleep"}})
	if err != nil {
		log.Fatal(err)
	}

	return func(i int) (io.ReadCloser, error) {
		return openEvents(router, agent, i)
	}
}

// openEvents creates session i on the router and returns its event stream.
func openEvents(router, agent string, i int) (io.ReadCloser, error) {
	id, err := post(router, "/sessions", map[string]any{"agent": agent, "prompt": strconv.Itoa(i)})
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(router + "/sessions/" + id + "/events")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		resp.Body.Close()

		return nil, fmt.Errorf("events: %s", resp.Status)
	}

	return resp.Body, nil
}

// openProcess starts ./singlesession for session i and returns its stdout.
func openProcess(i int) (io.ReadCloser, error) {
	if _, err := os.Stat(binary); err != nil {
		return nil, errors.New("singlesession binary not found")
	}

	cmd := exec.Command(binary, "-prompt", strconv.Itoa(i))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return process{stdout, cmd}, nil
}

// removeStores deletes the stores the singlesession processes left in the temp folder.
func removeStores() {
	stores, err := filepath.Glob(filepath.Join(os.TempDir(), "pointsman-*"))
	if err != nil {
		log.Fatal(err)
	}
	for _, store := range stores {
		if err := os.RemoveAll(store); err != nil {
			log.Fatal(err)
		}
	}
}

// measure runs session i and returns its overhead: its time beyond the simulated work. A failed request is "error".
func measure(open func(int) (io.ReadCloser, error), i int) result {
	start := time.Now()
	status, work, err := run(open, i)
	if err != nil {
		status = "error"
	}

	return result{
		overhead: time.Since(start) - work,
		status:   status,
	}
}

// run opens session i and reads its events until the status frame. It returns that status and the work the
// session simulated: the inference times in the assistant texts and the sleep ms in their tool calls.
func run(open func(int) (io.ReadCloser, error), i int) (string, time.Duration, error) {
	events, err := open(i)
	if err != nil {
		return "", 0, err
	}
	defer events.Close()

	var work time.Duration
	scanner := bufio.NewScanner(events)
	for scanner.Scan() {
		event, ok := strings.CutPrefix(scanner.Text(), "event: ")
		if !ok {
			continue
		}
		scanner.Scan()
		data := []byte(strings.TrimPrefix(scanner.Text(), "data: "))

		switch event {
		case "message":
			ms, err := inferenceTime(data)
			if err != nil {
				return "", 0, err
			}
			work += ms
		case "status":
			var status statusEvent
			err := json.Unmarshal(data, &status)

			return status.Status, work, err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	return "", 0, fmt.Errorf("events: stream ended without a status")
}

// inferenceTime returns the work in one message frame: the inference time of an assistant text and its sleep ms.
func inferenceTime(data []byte) (time.Duration, error) {
	var msg message
	if err := json.Unmarshal(data, &msg); err != nil {
		return 0, err
	}
	if msg.Role != "assistant" {
		return 0, nil
	}

	var text inference
	if err := json.Unmarshal([]byte(msg.Content), &text); err != nil {
		return 0, err
	}
	work := time.Duration(text.Ms) * time.Millisecond

	for _, call := range msg.ToolCalls {
		var arguments sleepArguments
		if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil {
			return 0, err
		}
		work += time.Duration(arguments.Ms) * time.Millisecond
	}

	return work, nil
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

	var reply idResponse
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
	w.Write([]string{"session", "overhead_ms", "status"})
	for i, r := range results {
		w.Write([]string{strconv.Itoa(i), strconv.FormatInt(r.overhead.Milliseconds(), 10), r.status})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func summarize(results []result, wall time.Duration) {
	failed, errored := 0, 0
	overheads := make([]time.Duration, len(results))
	for i, r := range results {
		overheads[i] = r.overhead
		switch r.status {
		case "failed":
			failed++
		case "error":
			errored++
		}
	}
	slices.Sort(overheads)

	n := len(overheads)
	round := 100 * time.Millisecond
	fmt.Printf("sessions %d  failed %d  error %d  wall %v\n", n, failed, errored, wall.Round(round))

	round = time.Millisecond
	low := overheads[0].Round(round)
	median := overheads[n/2].Round(round)
	p99 := overheads[n*99/100].Round(round)
	high := overheads[n-1].Round(round)
	fmt.Printf("overhead  min %v  median %v  p99 %v  max %v\n", low, median, p99, high)
}
