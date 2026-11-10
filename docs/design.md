# Design record

The architecture and its decisions (D1-D32) are in [SPEC.md](../SPEC.md). This file holds the implementation state:
what is built, the facts it rests on, what is next and open.

## Built

**provider**: one OpenAI-compatible Chat Completions client (`openai-go/v3`).
`Context(agent, messages)` is the pure build function of D9: it prepends the agent's prompt and returns a `Request`
with the agent's model, tools and config. `Request` is an alias of the SDK params: marshal it to see the body as
sent. `Send(ctx, request)` makes the call and returns the reply message. SDK retries are off: retries belong to the
router (D16). `Message`, `Reply`, `Tool` and `Request` are aliases of the SDK types; `System`, `User` and
`ToolResult` build messages.

**agent**: config only, no runtime state (SPEC §3); the context (the log) belongs to the session, so many sessions
share one agent. `agent.New(model, prompt, tools, config)` returns `Agent{Model, Prompt, Tools, Config}`, with
`Config{Effort, MaxTokens}`.
- `Effort` is the SDK's `shared.ReasoningEffort` and its constants (no own copy of the list; a second provider would
  give it its own type). `MaxTokens` is a `uint`. Both are optional in the API: the zero value is not sent.
- `provider` imports `agent`, never the reverse.

**tool**: `Schema{Name, Description, Parameters}`, a tool's declaration as sent to the model.
- Data, not an interface with `Run`: the router never runs tool logic (SPEC §1). The tool instances live in the
  dispatcher.
- `Parameters` is a `*jsonschema.Schema` (`google/jsonschema-go`); `tool.New(name, description, parameters)` builds
  a `Schema`.
- `provider.Tools` converts schemas to the wire format (the SDK's map) and returns the conversion error, which
  `Context` passes on. It is a plain function.

**session**: `Session{ID, Agent, Messages}`. `session.New(agent)` returns a `*Session` with a random ID
(stdlib `uuid.New()`; V7 if sorting by creation is ever needed) and no messages. `AppendUser`, `AppendReply` and
`AppendToolResult` are the only way the loop writes the log; a log entry is a `provider.Message`. The ID never reaches
the model (D3): `Context` takes the agent and the messages only.

## Facts: the Chat Completions context

- Stateless: every request carries `model`, `tools[]`, `messages[]` and generation settings.
- Roles: `system`/`developer` (prompt), `user`, `assistant` (with `tool_calls[]`), `tool` (with `tool_call_id`).
  Every tool call needs one `tool` message before the next assistant turn, or the request is rejected.
- The response also carries `finish_reason` (`"length"` means truncated), `usage` and `model`; `Send` returns
  only the message. `Reply.ToParam()` drops `annotations`, and provider extra fields (e.g. `reasoning_content`) are
  not carried back.
- A captured first request of a coding agent: about 26 KB of prefix from agent config (a 16 KB `developer` prompt
  built from base prompt, tool summaries, rules, skills and the working directory; 10 tool schemas, 10 KB), and one
  user message from the session. It sets `store: false`, `stream: true` with `include_usage`, `reasoning_effort` and
  `max_completion_tokens`, and sends user content as parts.

## Next

Dispatch. The router sends each tool call over gRPC to a dispatcher, a separate process that holds the tool
instances. A tool is a function there: it may start a Docker container (one per call, the session's workspace
mounted) or run directly. The router never runs tool logic (SPEC §1, D2). The contract:
```proto
service Dispatcher {
  rpc Call(Request) returns (Result);
}

message Request {
  string session_id = 1; // the dispatcher finds the session's workspace from it
  string call_id    = 2; // also the idempotency key (D23)
  string name       = 3; // the tool
  string arguments  = 4; // the model's JSON, passed through
}

message Result {
  string output = 1;
  uint32 code   = 2; // 0 = success; a Docker tool returns the container's exit code
}
```
A tool error is a `Result` with `code != 0`, sent to the model. An infrastructure error (dispatcher down, Docker
failure, unknown tool) is a gRPC status error, handled by the router (D24). The model sees only text, so the router
writes a non-zero code into the tool message; how comes with the loop. Large outputs (D25), streaming output and tool
listing are left out until their piece. MCP was weighed and set aside: its gains (tool discovery, reuse of existing
servers) do not apply, and it is a large dependency with its own session state.

The contract is in `dispatch/dispatch.proto`; regenerate with `go generate ./dispatch` (needs protoc,
protoc-gen-go and protoc-gen-go-grpc on PATH).

Layout: one Go module. The two apps will be `cmd/router` and `cmd/dispatcher`, each added by the change that writes
its `main`; the library packages stay where they are.

Then the session loop, built on dispatch: `Send`, append the reply, run each tool call, stop when the reply has no
tool calls. Streaming is a change of its own (it changes how `Send` reads the response, D30).

## Open

- **Truncated replies.** `finish_reason == "length"` is a partial response (D10): `Send` should return an error
  or the reason.
- **Workspace storage.** Who owns it (the dispatcher, from the `session_id`?) and when is it deleted (with the
  session?).
- **Where a tool's schema lives.** `agent.Tools` holds schemas while the dispatcher holds the tool instances: one
  source of truth would put the schema with the instance, and the agent would only name its tools.
- **`developer` or `system` role.** Some OpenAI-compatible servers reject `developer`; `provider` has only `System`.

## Deferred

- Validating tool arguments against their schema in the router (the library is already a dependency): decided with
  dispatch.
