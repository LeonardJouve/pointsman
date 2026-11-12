package api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

const (
	eventMessage   = "message"
	eventStatus    = "status"
	eventKeepAlive = "keep-alive"
)

// event is the JSON of every frame: the client reads the type from Event.
type event struct {
	Event string `json:"event"`
	Data  any    `json:"data,omitempty"`
}

// writeEntry writes a log entry; the frame id is its log index.
func writeEntry(w io.Writer, index int, message provider.Message) {
	data, _ := json.Marshal(event{Event: eventMessage, Data: message})
	fmt.Fprintf(w, "id: %d\ndata: %s\n\n", index, data)
}

// writeStatus writes the final frame, which has no id.
func writeStatus(w io.Writer, status session.Status) {
	data, _ := json.Marshal(event{Event: eventStatus, Data: status})
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func writeKeepAlive(w io.Writer) {
	data, _ := json.Marshal(event{Event: eventKeepAlive})
	fmt.Fprintf(w, "data: %s\n\n", data)
}
