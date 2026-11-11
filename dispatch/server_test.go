package dispatch

import (
	"context"
	"testing"

	"github.com/LeonardJouve/pointsman/tool"
)

func tools() []tool.Tool {
	return []tool.Tool{
		{
			Schema: tool.Schema{Name: "echo"},
			Run: func(ctx context.Context, arguments string) (string, uint32) {
				return arguments, 3
			},
		},
	}
}

func TestCall(t *testing.T) {
	result, err := NewServer(tools()).Call(context.Background(), &Request{Name: "echo", Arguments: `{"a":1}`})
	if err != nil {
		t.Fatal(err)
	}

	if result.Output != `{"a":1}` || result.Code != 3 {
		t.Errorf("got output %q code %d, want %q code 3", result.Output, result.Code, `{"a":1}`)
	}
}

func TestCallUnknownTool(t *testing.T) {
	if _, err := NewServer(tools()).Call(context.Background(), &Request{Name: "missing"}); err == nil {
		t.Error("want an error for an unknown tool")
	}
}
