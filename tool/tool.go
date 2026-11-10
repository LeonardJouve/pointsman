package tool

import "context"

type Tool struct {
	Schema Schema
	Run    func(ctx context.Context, arguments string) (output string, code uint32, err error)
}
