package api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

const (
	eventMessage = "message"
	eventStatus  = "status"
)

type statusEvent struct {
	Status session.Status `json:"status"`
}

// writeEvent writes a frame: the event name and the JSON of payload.
func writeEvent(w io.Writer, name string, payload any) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data)
}

// writeMessage writes a log entry; the frame id is its log index.
func writeMessage(w io.Writer, index int, message provider.Message) {
	fmt.Fprintf(w, "id: %d\n", index)
	writeEvent(w, eventMessage, message)
}

// writeStatus writes the final frame, which has no id.
func writeStatus(w io.Writer, status session.Status) {
	writeEvent(w, eventStatus, statusEvent{Status: status})
}

func writeKeepAlive(w io.Writer) {
	fmt.Fprint(w, ": keep-alive\n\n")
}
