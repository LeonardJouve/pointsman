package loop

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/LeonardJouve/pointsman/dispatch"
	"github.com/LeonardJouve/pointsman/provider"
	"github.com/LeonardJouve/pointsman/session"
)

// Run sends the log to the model until a reply has no tool calls, then sets the status to Succeeded.
// On an error the log is left as it was and the status is Failed.
func Run(ctx context.Context, s *session.Session, client *provider.Client, dispatcher dispatch.DispatcherClient) (err error) {
	defer func() {
		if err != nil {
			s.SetStatus(session.Failed)
			return
		}

		s.SetStatus(session.Succeeded)
	}()

	for {
		request, err := provider.Context(s.Agent, s.Messages())
		if err != nil {
			return err
		}

		reply, err := client.Send(ctx, request)
		if err != nil {
			return err
		}

		if len(reply.ToolCalls) == 0 {
			s.AppendReply(reply)

			return nil
		}

		outputs, err := callTools(ctx, s, reply, dispatcher)
		if err != nil {
			return err
		}

		s.AppendReply(reply)
		for i, call := range reply.ToolCalls {
			s.AppendToolResult(call.ID, outputs[i])
		}
	}
}

// callTools runs the calls of the reply at the same time and returns their outputs in call order.
func callTools(ctx context.Context, s *session.Session, reply provider.Reply, dispatcher dispatch.DispatcherClient) ([]string, error) {
	outputs := make([]string, len(reply.ToolCalls))
	errs := make([]error, len(reply.ToolCalls))

	var wg sync.WaitGroup
	for i, call := range reply.ToolCalls {
		wg.Go(func() {
			result, err := dispatcher.Call(ctx, &dispatch.Request{
				SessionId: s.ID.String(),
				CallId:    call.ID,
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			})
			if err != nil {
				errs[i] = err

				return
			}

			outputs[i] = result.Output
			if result.Code != 0 {
				outputs[i] += fmt.Sprintf("\nCommand exited with code %d", result.Code)
			}
		})
	}

	wg.Wait()

	return outputs, errors.Join(errs...)
}
