package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
)

type sleepArguments struct {
	Ms uint `json:"ms" jsonschema:"Milliseconds to wait"`
}

func Sleep() (Tool, error) {
	schema, err := jsonschema.For[sleepArguments](nil)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Schema: New("sleep", "Wait for the given number of milliseconds.", schema),
		Run:    sleep,
	}, nil
}

func sleep(ctx context.Context, arguments string) (string, uint32) {
	var args sleepArguments
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fail(err)
	}

	time.Sleep(time.Duration(args.Ms) * time.Millisecond)

	return fmt.Sprintf("Slept %d ms", args.Ms), 0
}
