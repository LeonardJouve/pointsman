package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand/v2"
	"net/http"
	"time"
)

const (
	addr         = ":8090"
	seed         = 42
	inferenceMin = 200 * time.Millisecond
	inferenceMax = 800 * time.Millisecond
	sleepMin     = 500 // ms, sent to the sleep tool
	sleepMax     = 3000
	rounds       = 5
)

type request struct {
	Messages []struct {
		Content string `json:"content"`
	} `json:"messages"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /chat/completions", complete)

	log.Fatal(http.ListenAndServe(addr, mux))
}

// complete answers like a model: the same session and turn always get the same reply.
func complete(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// messages are system, user, then one assistant and one tool per round.
	prompt := req.Messages[1].Content
	turn := uint64(len(req.Messages)-2) / 2

	inference, ms := timings(prompt, turn)
	time.Sleep(inference)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply(turn, ms))
}

// timings returns the inference time and the sleep length of a session and turn, the same in any order and process.
func timings(prompt string, turn uint64) (inference time.Duration, ms int) {
	hash := fnv.New64a()
	hash.Write([]byte(prompt))
	random := rand.New(rand.NewPCG(seed, hash.Sum64()^turn))

	inference = inferenceMin + time.Duration(random.Int64N(int64(inferenceMax-inferenceMin)+1))
	ms = sleepMin + random.IntN(sleepMax-sleepMin+1)

	return inference, ms
}

type Message = map[string]any

// reply is the response body: a sleep call of ms before the last round, "done" after it.
func reply(turn uint64, ms int) map[string]any {
	var message Message
	if turn < rounds {
		message = Message{
			"role": "assistant",
			"tool_calls": []any{map[string]any{
				"id":   fmt.Sprintf("call_%d", turn),
				"type": "function",
				"function": map[string]any{
					"name":      "sleep",
					"arguments": fmt.Sprintf(`{"ms": %d}`, ms),
				},
			}},
		}
	} else {
		message = Message{
			"role":    "assistant",
			"content": "done",
		}
	}

	return map[string]any{"choices": []any{map[string]any{"message": message}}}
}
