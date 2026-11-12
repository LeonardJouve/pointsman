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
share one agent. `Agent{ID, Name, Model, Prompt, Tools, Config}` is also the API's JSON and the agent file (tags
`id`, `name`, `model`, `prompt`,
`tools`, and `Config` embedded so `effort` and `max_tokens` are flat). `Name` is an optional free-text label: the
uuid is the key, two agents may share a name and an empty one is accepted. `Tools` are tool names, not schemas:
`agent` does not import `tool`. `agent.New(name, model, prompt, tools, config)` builds one;
`Config{Effort, MaxTokens}`.
- `Effort` is the SDK's `shared.ReasoningEffort` and its constants (no own copy of the list; a second provider would
  give it its own type). `MaxTokens` is a `uint`. Both are optional in the API: the zero value is not sent.
- `provider` imports `agent` and `tool`, never the reverse.
- `agent.Save(a)` writes `<a.ID>.json` in `store.Agents()` through a temp file and a rename; `agent.Load()` reads
  every `*.json` there into a map of id (the file name) to `*Agent`. Neither takes a folder.

**store**: only decides where things live, never reads or writes their files. `store.Dir()` is
a new temporary folder per process (`os.MkdirTemp`, prefix `pointsman-`), made once (`sync.OnceValues`): a
restart starts empty and resumes nothing (loading and `Resume` stay, idle). `store.Agents()` and `store.Sessions()` are
its `agents` and `sessions` folders; both create theirs.

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

**session**: `Session{ID, Agent, Created}` (`Agent` is a `*agent.Agent`) and an unexported log.
`session.New(a)` returns a `*Session` and an error, with a random ID (stdlib `uuid.New()`; V7 if sorting by creation is
ever needed), `Created` set to `time.Now()`, and no messages. `AppendUser` and `AppendRound` are the
only way the loop writes the log and return an error; a log entry is a `provider.Message`. `AppendRound(reply, outputs)`
writes a reply and its tool results, in call order, in one write, so a recorded round is a checkpoint: all of it is in
the file or none (a resumed session restarts from it, and the model API rejects a reply missing results); readers see it
whole, with one wake-up. It does not check `len(outputs)`: the loop builds both. A mutex guards the log: the
Append methods lock, and `Messages()` locks and returns a copy, so a reader (an HTTP handler) never shares the slice
with the loop. The same mutex guards the run status: `Status` is a string type with three words, `running`,
`succeeded` and `failed` (`Running`, `Succeeded`, `Failed`), so the JSON is the word; `New` sets `Running`; `Status()`
and `SetStatus()` lock. Every append and `SetStatus` closes a `changed` channel and makes a new one;
`Changed()` returns the current one. The stream calls `Changed()`, then `Status()`, then `Messages()`: a later change
closes the channel, so no wake-up is missed, and the status is read before the log because `loop.Run` sets it last (no
`sync.Cond`, no polling). `loop.Run` calls `SetStatus` once, before it returns (`Succeeded` or `Failed`); it logs the
error with `log.Print`, as a deferred call cannot return it. A failed run leaves the log unchanged and the session
keeps no error.
The ID never reaches the model (D3): `Context` takes the agent and the messages only.

The file: `<store.Sessions()>/<id>.jsonl` (mode 0600),
one JSON object with one key per line (the keys are the `kind` constants), in `session/store.go`.
`New` creates it (it must not exist) with `{"header":{"created":"<RFC3339>","agent":"<agent id>"}}`: the agent id,
not a copy of the agent. Then `{"message":<provider.Message>}` for every message, and `{"status":"succeeded"}` or
`{"status":"failed"}` when the status is set; `Running` is never written, it is the absence of a status line, so a
file with no status line has not ended. Every write is one append: the file is opened, written and closed each time, the
session holds no handle, and there is no sync. The lines of one round go in one write. The Append methods write first
and change the memory only if it succeeded, so the log never holds a message the file does not; a failed write returns
the error and changes nothing. `SetStatus` is the exception: it sets the memory and wakes the readers even if the write
fails (the stream would wait on `Running` forever), then returns the write error.
`session.Load(agents)` reads every `*.jsonl` in `store.Sessions()` into a map of id (the file name) to `*Session`: the
header gives `Created` and the agent (looked up in `agents`), then the `message` lines in order, and a `status` line
sets the status. A file that cannot be loaded (first line not a header, a line that does not parse, an unknown kind,
an agent not in the map) is an error naming the file, and `cmd/router` exits. Other damage (a torn last line, a
duplicate header, an unknown status word) is not handled. A file with no status line loads as `Running`: it has not
ended, and `api.Server.Resume` runs it again (see `api`).

**loop**: `loop.Run(ctx, s, client, dispatcher)` runs a session until the model answers without tool calls. The
caller appends the user message first and reads the answer from the last message of the log.
- Each turn: `provider.Context`, `Send`, then the dispatcher is called for every tool call of the reply.
- The calls of one reply run at the same time (`sync.WaitGroup`, results by position); every call runs to the end,
  failures are joined. Two calls of one reply touching the same file have no defined order.
- The log stays valid: the reply and all its results are appended together, after every call succeeded. On any
  error (`Send`, a truncated reply, a dispatcher call) `Run` returns it and the log is unchanged. No retry.
- `Run` works on a loaded log as on a new one, so a resumed session repeats the step that was cut off.
- The request carries the session ID, the call's ID, the function name and the model's arguments as they are.
- A dispatcher error is returned, never written to the model (D24). A `Result` with `code != 0` becomes the tool
  message `output + "\nCommand exited with code N"`; code 0 leaves
  the output as is.

**tools `read`, `write` and `sleep`** (constructors `tool.Read()`, `tool.Write()` and `tool.Sleep()`, returning the
schema-generation error; `tool.All()` returns every built-in tool, the one list, and the first constructor error):
copies of pi's, same names, parameters, descriptions and limits (2000 lines or 50KB, from the constants that enforce
them). They run directly in the dispatcher, no Docker.
- `sleep` is not from pi: parameter `ms`, waits with `time.Sleep` in the dispatcher and returns `Slept N ms`. The load
  experiment uses it to hold a session in the dispatcher like a real tool call.
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

**cmd/fakemodel**: a fake OpenAI-compatible model for the load experiment (point `LLM_BASE_URL` at it). It serves `POST
/chat/completions` on `:8090`; every constant is at the top of `main.go`. A session is `rounds` `sleep` tool calls then
`done`. Messages are system, user, then an assistant and a tool message per round; the session is message 1, the turn
is the rounds so far. Each reply waits a random inference time, drawn from a generator seeded by (seed, FNV-1a hash of
the prompt, turn), so a session gets the same work in any arrival order and in any process.

**cmd/load**: the load generator (`go run ./cmd/load`; constants at the top of `main.go`, router address from
`ROUTER_ADDR`). It creates one agent (model `fake`, tool `sleep`, empty prompt),
then starts `sessions` sessions at once, one goroutine each, session i with prompt
`"i"`. Latency is the time from before `POST /sessions` to the `status` frame of its event stream. It writes `output`
(`session,latency_ms,status`, one row per session in order) and prints the counts and min, median, p99 and max latency.
Status is `succeeded` or `failed` as the router reports it, or `error` when a request failed or the stream ended
without a status frame; the run goes on.

**cmd/router**: loads the agents and sessions (`agent.Load`, `session.Load`), builds
`provider.New(LLM_BASE_URL, LLM_API_KEY)`, dials the dispatcher at `DISPATCHER_ADDR` (insecure
gRPC), calls `Resume` on the `api` server and serves it on `ROUTER_ADDR`; its error is fatal. Both mains read their
settings with `env.Get`. `.env` is loaded (`env.Load`) only when `env.IsDevelopment()`, and the process environment
wins over the file; in production only the process environment is used. `.env` is git-ignored.
All four variables are required; `.env.example` lists them.

**api**: the router's HTTP API, stdlib `net/http` (`ServeMux` patterns, `encoding/json`).
`server := api.New(client, dispatcher, agents, sessions)`, `server.Resume()`, then
`http.ListenAndServe(addr, server.Handler())`, with the `*provider.Client`, the
`dispatch.DispatcherClient` the runs use, and the agents and sessions `agent.Load()` and `session.Load` read at
startup (`main` exits on a load error). `New` is a plain constructor.
`Resume` starts a background run for every session in the map whose status is `Running`: a session with no status
line did not end, and `loop.Run` runs it again from its log. The repeated step is safe because nothing incomplete is
written: a round is in the file whole or not at all, so the log ends on a complete message and the step is simply
done again. `createSession` and `Resume` start their runs through one method, `run`, which calls
`loop.Run(context.Background(), ...)` in a goroutine and logs its error with `log.Print`.
A create request is decoded straight into `agent.Agent`; its tool names are checked with `tool.Get`, the same lookup
`Context` uses, the router sets its `ID`, then `agent.Save` writes it before the map entry is set (a save error
is a `500` and nothing is kept).
`Server` holds the agents (`agent.Agents`, persisted) and the sessions (`session.Sessions`,
loaded by `session.Load` at startup and each new one also written to its file) behind one mutex.
- `POST /agents` takes `model`, `prompt`, `tools` (names), and optional `name`, `effort` and `max_tokens`; it returns
  `201` and `{"id": "<uuid>"}`, the id generated by the router.
- `GET /agents` returns all the agents as a map of id to agent (each also has its `id`; `tools` are the stored
  names); `{}` when empty.
  There is no `GET /agents/{id}` for now.
- `GET /models` returns the model ids the provider lists, a JSON array (`[]` when empty); a provider error is a `502`
  with `{"error": "..."}`. `POST /agents` does not check its model against it.
- `GET /tools` returns the schema of every built-in tool (`tool.All()`), a JSON array of `tool.Schema` with its Go
  field names as keys (`Name`, `Description`, `Parameters`); `Run` is not sent. An error is a `500` with
  `{"error": "..."}`.
- `POST /sessions` takes `{"agent": "<agent id>", "prompt": "..."}` (an empty prompt is accepted), creates the session,
  appends the prompt, starts the run (`run`) and returns `201` and
  `{"id": "<uuid>"}` at once. A failed file write (`session.New` or the prompt) is a `500`: the session is not kept and
  no run starts. `Run` sets the status (`succeeded` or `failed`); the goroutine
  only logs its error with `log.Print`. Nothing cancels a run: a stopped router kills its runs
  (the folder is new at each start, so `Resume` finds nothing).
- `GET /sessions` returns a map of session id to `{"created": <RFC3339>, "status": "...", "agent": {...}}` (the
  agent the session holds, with its `id`), `{}` when empty; no messages and no order: the page sorts by `created`.
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
- `GET /` serves the dashboard: static HTML and vanilla JS in `api/web/`, embedded with `go:embed` and served by
  `http.FileServerFS`; the API routes are more specific and win. `index.html` has Bootstrap 5.3.9 from the jsDelivr
  CDN and four tabs (Agents, Sessions, Models, Tools); `main.js` only wires, one ES module per tab. All four lists are
  built. The Agents pane has two parts: a top-right "New agent" button opens `agent-form.js` in a modal (name,
  model from `GET /models`, prompt, tools as checkboxes from `GET /tools`, effort and max_tokens; Create and Cancel). It
  sends `POST /agents`; on success the modal closes, the form resets and the callback reloads the list; on failure the
  modal stays open with the error text. Effort is a literal list of six SDK values (minimal to max), `medium`
  preselected and always sent; the API still treats it as optional. `max_tokens` is sent only when set. `agent.js`
  lists `GET /agents` (the name folded, its attributes unfolded, "No agents" when empty). `session.js` lists
  `GET /sessions` at load and after a create, newest first (sorted in the page by `created`): a folded `<details>` per
  session with the agent name and the local time, and the status as a small coloured circle at the right (Bootstrap
  colours; the word is its hover title and hidden text); "No sessions" when empty. Each unfold of a session opens its
  `EventSource` (`conversation.js`) and redraws the log from the start; folding, or the final `status` event, closes it
  (that event also updates the circle). One centred column of bubbles, the tail on the left for user messages and tool
  results and on the right for the assistant, tool calls as `<pre><code>` after the text. `style.css` holds only the
  tails. A top-right "New session" button opens `session-form.js` in a modal: an agent dropdown refetched from
  `GET /agents` on every open (the id is the label when the name is empty), a prompt, Create and Cancel. It sends
  `POST /sessions`; on success the modal closes, the form resets and the list reloads; on failure the modal stays open
  with the error text. `model.js` lists `GET /models`, and `tool.js` shows `GET /tools`, each tool a folded `<details>`
  in a bordered list, with its name as summary, then its description and parameters as JSON. A failed fetch shows its
  error text instead. `api.js` (`request`: GET or POST JSON, returns the body or the API's error text) and `form.js`
  (`field`) are shared; there is one file per tab otherwise.
- Errors are `{"error": "..."}`: bad JSON, an unknown tool name, or (for a session) an agent id that is not a uuid or
  not known is a 400; an unknown or malformed session id is a 404.

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
- Storage for sessions: writing, loading and resuming are built; the empty log (a header with no messages, which
  resumes with no messages) is open.
- Piece 4, the Docker tool, with workspaces.
- Validating tool arguments against their schema in the router.
- The dashboard tabs, in order: Models, Tools, Agents (list and form) and Sessions (list, form and streamed
  conversation) are all built; the dashboard plan is complete.

## Open

- **Transient provider errors.** The loop does not retry a failed `Send` (D16); it returns the error with the log
  unchanged, so the caller may call `Run` again.
- **Memory and the model.** Agents and sessions are persisted and loaded at startup.
- **The session holds a pointer to its agent.** Safe because agents are immutable. Revisit when agents become
  editable: a session would then need its own copy to pin the config version (D12, D18).
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
