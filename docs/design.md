# Design record

The architecture and its decisions (D1-D32) are in [SPEC.md](../SPEC.md). This file holds the implementation state:
what is built, the facts it rests on, what is next and open.

## Built

**provider**: one OpenAI-compatible Chat Completions client (`openai-go/v3`).
`Context(agent, messages)` is the build function of D9: it prepends the agent's prompt and returns a `Request`
with the agent's model, tools and config. It resolves the agent's tool names with `tool.Get` (an unknown name
is an error), so it is not pure and builds the schemas on every call. `Request` is an alias of the SDK params:
marshal it to see the body as sent. `Send(ctx, request)` makes the call and returns the reply message. SDK retries
are off: retries belong to the router (D16). `Message`, `Reply`, `Tool` and `Request` are aliases of the SDK types;
`System`, `User` and `ToolResult` build messages. `Send` returns a plain error when `finish_reason` is `"length"`
(D10): the partial reply is dropped.
`Models(ctx)` lists the provider's model ids (the SDK's models list); a provider error is returned as is, nothing is
cached.

**agent**: config only, no runtime state (SPEC §3); the context (the log) belongs to the session, so many sessions
share one agent. `Agent{Model, Prompt, Tools, Config}` is also the API's JSON (tags `model`, `prompt`, `tools`, and
`Config` embedded so `effort` and `max_tokens` are flat). `Tools` are tool names, not schemas: `agent` does not
import `tool`. `agent.New(model, prompt, tools, config)` builds one; `Config{Effort, MaxTokens}`.
- `Effort` is the SDK's `shared.ReasoningEffort` and its constants (no own copy of the list; a second provider would
  give it its own type). `MaxTokens` is a `uint`. Both are optional in the API: the zero value is not sent.
- `provider` imports `agent` and `tool`, never the reverse.

**tool**: `Schema{Name, Description, Parameters}`, a tool's declaration as sent to the model.
`tool.Get(name)` returns the built-in tool by its name (an unknown name is an error); `provider.Context` and the
`api` check loop over the agent's names with it.
- Data, not an interface with `Run`: the router never runs tool logic (SPEC §1). The tool instances live in the
  dispatcher.
- `Parameters` is a `*jsonschema.Schema` (`google/jsonschema-go`); `tool.New(name, description, parameters)` builds
  a `Schema`.
- `provider.Tools` converts schemas to the wire format (the SDK's map) and returns the conversion error, which
  `Context` passes on. It is a plain function.
- `Tool{Schema, Run}` pairs a schema with its function,
  `Run func(ctx, arguments string) (output string, code uint32)`: `arguments` is the model's JSON, parsed by the
  tool; a tool failure is a non-zero `code` with the message as `output`. A tool has no error path: errors are the
  dispatcher's own (an unknown tool). The router imports a tool for its schema (`provider`
  resolves names with it), the dispatcher for its function. One module builds both apps from one commit, so the two
  cannot drift; a `List` RPC would replace the import only if tools must be deployed without the router. The
  workspace parameter comes with the Docker tool, and so does where tool instances live.

**dispatch**: the gRPC contract (below) and `NewServer(tools []tool.Tool)`, which implements `Dispatcher.Call`: it
looks the tool up by `Schema.Name` (a name is written once), runs it and returns the `Result`. An unknown tool is a
plain Go error: it reaches the router as gRPC `Unknown`. The loop writes a non-zero code into the tool message. Tests
call `Call` directly, without a gRPC server.

**session**: `Session{ID, Agent}` and an unexported log. `session.New(agent)` returns a `*Session` with a random ID
(stdlib `uuid.New()`; V7 if sorting by creation is ever needed) and no messages. `AppendUser`, `AppendReply` and
`AppendToolResult` are the only way the loop writes the log; a log entry is a `provider.Message`. A mutex guards the
log: the Append methods lock, and `Messages()` locks and returns a copy, so a reader (an HTTP handler) never shares
the slice with the loop. The same mutex guards the run status: `Status` is a string type with three words, `running`,
`succeeded` and `failed` (`Running`, `Succeeded`, `Failed`), so the JSON is the word; `New` sets `Running`;
`Status()` and `SetStatus()` lock. Every Append and `SetStatus` closes a `changed` channel and makes a new one;
`Changed()` returns the current one. The stream calls `Changed()`, then `Status()`, then `Messages()`: a later change
closes the channel, so no wake-up is missed, and the status is read before the log because `loop.Run` sets it last (no
`sync.Cond`, no polling). `loop.Run` calls `SetStatus` once,
before it returns (`Succeeded` or `Failed`). A failed run leaves the log unchanged and the session keeps no
error. The ID never reaches the model (D3): `Context` takes the agent and the messages only.

**loop**: `loop.Run(ctx, s, client, dispatcher)` runs a session until the model answers without tool calls. The
caller appends the user message first and reads the answer from the last message of the log.
- Each turn: `provider.Context`, `Send`, then the dispatcher is called for every tool call of the reply.
- The calls of one reply run at the same time (`sync.WaitGroup`, results by position); every call runs to the end,
  failures are joined. Two calls of one reply touching the same file have no defined order.
- The log stays valid: the reply and all its results are appended together, after every call succeeded. On any
  error (`Send`, a truncated reply, a dispatcher call) `Run` returns it and the log is unchanged. No resume, no retry.
- The request carries the session ID, the call's ID, the function name and the model's arguments as they are.
- A dispatcher error is returned, never written to the model (D24). A `Result` with `code != 0` becomes the tool
  message `output + "\nCommand exited with code N"`; code 0 leaves
  the output as is.

**tools `read` and `write`** (constructors `tool.Read()` and `tool.Write()`, returning the schema-generation error;
`tool.All()` returns every built-in tool, the one list, and the first constructor error):
copies of pi's, same names, parameters, descriptions and limits (2000 lines or 50KB, from the constants that enforce
them). They run directly in the dispatcher, no Docker.
- The parameters schema is generated from the argument struct (`jsonschema.For`, descriptions in `jsonschema` tags,
  optional fields `omitzero`), so the struct is the one source of truth. The generated schema differs from pi's:
  `offset` and `limit` are integers and `additionalProperties` is false. A `limit` of 0 or absent means no limit (pi
  returns nothing for 0).
- `read` is pi's text path only: no images. A first line over the byte limit gets a notice without a hint.
- `read` continues with one note for every cut (line cap, byte cap or `limit`): `[Showing lines A-B of N. Use
  offset=X to continue.]`, instead of pi's three variants.
- Errors (missing file, bad offset, bad arguments JSON) are a `Result` with code 1 and the message as output.
- Paths: relative ones start at `root` (`"."`, in `tool/tool.go`); absolute paths and `..` are not rejected, as in
  pi. Confinement and the root come with workspaces.

**cmd/dispatcher**: listens on `DISPATCHER_ADDR` and serves `dispatch.NewServer` with the tools from `tool.All()`;
its error is fatal.

**cmd/router**: builds `provider.New(LLM_BASE_URL, LLM_API_KEY)`, dials the dispatcher at `DISPATCHER_ADDR` (insecure
gRPC) and serves `api` on `ROUTER_ADDR`; its error is fatal. Both mains read their settings
with `env.Get`. `.env` is loaded (`env.Load`) only when `env.IsDevelopment()`, and the process environment
wins over the file; in production only the process environment is used. `.env` is git-ignored.
All four variables are required; `.env.example` lists them.

**api**: the router's HTTP API, stdlib `net/http` (`ServeMux` patterns, `encoding/json`).
`http.ListenAndServe(addr, api.New(client, dispatcher).Handler())`, with the `*provider.Client` and the
`dispatch.DispatcherClient` the runs use. A create request is decoded straight into `agent.Agent`; its tool
names are checked with `tool.Get`, the same lookup `Context` uses. `Server` holds the agents
(`map[uuid.UUID]agent.Agent`) and the sessions (`map[uuid.UUID]*session.Session`) in memory behind one mutex: they
vanish at restart; storage is a later piece.
- `POST /agents` takes `model`, `prompt`, `tools` (names), and optional `effort` and `max_tokens`; it returns
  `201` and `{"id": "<uuid>"}`, the id generated by the router.
- `GET /agents` returns all the agents as a map of id to agent (`tools` are the stored names); `{}` when empty.
  There is no `GET /agents/{id}` for now.
- `GET /models` returns the model ids the provider lists, a JSON array (`[]` when empty); a provider error is a `502`
  with `{"error": "..."}`. `POST /agents` does not check its model against it.
- `POST /sessions` takes `{"agent": "<agent id>", "prompt": "..."}` (an empty prompt is accepted), creates the session,
  appends the prompt, starts `loop.Run(context.Background(), ...)` in a goroutine and returns `201` and
  `{"id": "<uuid>"}` at once. `Run` sets the status (`succeeded` or `failed`); the goroutine
  only logs its error with `log.Print`. Nothing cancels a run: a stopped router kills its runs and loses its sessions.
- `GET /sessions/{id}` returns `200` and `{"status": "...", "messages": [...]}`, the log as `provider.Message`
  (OpenAI wire format).
- `GET /sessions/{id}/events` is a server-sent event stream (D30), plain `net/http` with `Flush`. Frames use the
  SSE fields and the `data:` line is a JSON object: a log entry is `id: <log index>`, `event: message`, `data:` the
  message JSON; the final frame has no `id`: `event: status`, `data: {"status": "<word>"}`. `api/sse.go` writes
  the frames (`writeMessage`, `writeStatus`, `writeKeepAlive`, over `writeEvent`). A connection starts at entry 0,
  or after the entry named by the `Last-Event-ID` header when it is an integer in the log (past the log it starts at
  the end; anything else, not a number, negative or overflowing, starts at entry 0). When the
  status is not `running` it writes the remaining entries, then `status`, and closes. While running it waits for a
  change, the client leaving, or a 15 s tick that writes the comment `: keep-alive`. A client should close on
  `status`: `EventSource` reconnects when the server closes.
- Errors are `{"error": "..."}`: bad JSON, an unknown tool name, or (for a session) an agent id that
  is not a uuid or not known is a 400; an unknown or malformed session id is a 404.

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
A tool error is a `Result` with `code != 0`, sent to the model. A dispatcher error (dispatcher down, unknown
tool) is a gRPC status error, handled by the router (D24). The model sees only text, so the router
writes a non-zero code into the tool message; how comes with the loop. Large outputs (D25), streaming output and tool
listing are left out until their piece. MCP was weighed and set aside: its gains (tool discovery, reuse of existing
servers) do not apply, and it is a large dependency with its own session state.

The contract is in `dispatch/dispatch.proto`; regenerate with `go generate ./dispatch` (needs protoc,
protoc-gen-go and protoc-gen-go-grpc on PATH).

Layout: one Go module. The two apps are `cmd/router` and `cmd/dispatcher`; the library packages stay where they are.

Plan, bottom-up; one piece is one change, and each lists the decisions it needs (pieces 1-3, 5 and 6.1-6.2, tools, the
server, `cmd/dispatcher`, `cmd/router`, `loop` and `api` agents and sessions, are built):
4. **A Docker tool** (deferred). A function that runs `docker run --rm` through `os/exec`, the session's workspace
   mounted, arguments on stdin, exit code as `code`. Needs: workspace storage (Open), the image, the network
   policy, how Docker's own failures (exit 125-127) are reported: a non-zero code with the message, or a way for
   `Run` to report them.
5. **The session loop.** Built (`loop`, above).
6. **The router API**, in three changes (HTTP, `cmd/router`):
   1. Agents. Built (`api`, above).
   2. Sessions. Built (`api`, above).
   3. The SSE stream. Built (`api` and `session`, above).

Left after 6.3, not yet ordered or designed (each is its own change; the points they need are under Open and
Deferred):
- Streaming from the model: changes how `Send` reads the response (D30).
- Session shutdown: cancel a run on stop. Runs use `context.Background()`, so only the process ending stops them.
- Retry of a failed `Send` in the router (D16).
- Storage for agents and sessions (both in memory today).
- Piece 4, the Docker tool, with workspaces.
- Validating tool arguments against their schema in the router.

## Open

- **Transient provider errors.** The loop does not retry a failed `Send` (D16); it returns the error with the log
  unchanged, so the caller may call `Run` again.
- **A reply before its tool results.** A reader can see the assistant reply before its tool results are appended.
  The window is a few microseconds and harmless; the stream reads the status before the log, so a finished run
  is sent whole.
- **Memory and the model.** Sessions and agents are in memory only.
- **The session holds a copy of its agent.** Kept on purpose: the copy pins the config version (D12, D18). Revisit
  when agents become editable; a pointer is safe only if agents are immutable.
- **No API test with a tool call.** `api` tests use a model that replies without tool calls, so the dispatcher is
  nil there; `loop` covers the tool round.
- **`-race` is unavailable** on the development machine (no cgo): the locks in `session` and `api` are checked by
  reading only.
- **Workspace storage.** Who owns it (the dispatcher, from the `session_id`?) and when is it deleted (with the
  session?).
- **`developer` or `system` role.** Some OpenAI-compatible servers reject `developer`; `provider` has only `System`.

## Deferred

- Validating tool arguments against their schema in the router (the library is already a dependency): decided with
  dispatch.
