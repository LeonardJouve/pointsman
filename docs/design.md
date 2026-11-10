# Design record

The architecture and its decisions (D1-D32) are in [SPEC.md](../SPEC.md). This file holds the implementation state:
what is built, the facts it rests on, what is next and open.

## Built

**provider**: one OpenAI-compatible Chat Completions client (`openai-go/v3`).
`Complete(ctx, model, schemas, messages)` returns the reply message. SDK retries are off: retries belong to the router
(D16). `Message`, `Reply` and `Tool` are aliases of the SDK types; `System`, `User` and `ToolResult` build messages.

**tool**: `Schema{Name, Description, Parameters}`, a tool's declaration as sent to the model.
- Data, not an interface with `Run`: the router never runs tool logic (SPEC §1). A call is executed by its routing
  kind (D20), sandbox or worker, the same way for every tool; that comes with dispatch.
- `Parameters` is a `*jsonschema.Schema` (`google/jsonschema-go`); `tool.New(name, description, parameters)` builds
  a `Schema`.
- `provider.Client.Tools` converts schemas to the wire format (the SDK's map) and returns the conversion error, which
  `Complete` passes on. It is a method on the client because the format belongs to the provider.

## Facts: the Chat Completions context

- Stateless: every request carries `model`, `tools[]`, `messages[]` and generation settings.
- Roles: `system`/`developer` (prompt), `user`, `assistant` (with `tool_calls[]`), `tool` (with `tool_call_id`).
  Every tool call needs one `tool` message before the next assistant turn, or the request is rejected.
- The response also carries `finish_reason` (`"length"` means truncated), `usage` and `model`; `Complete` returns
  only the message. `Reply.ToParam()` drops `annotations`, and provider extra fields (e.g. `reasoning_content`) are
  not carried back.
- A captured first request of a coding agent: about 26 KB of prefix from agent config (a 16 KB `developer` prompt
  built from base prompt, tool summaries, rules, skills and the working directory; 10 tool schemas, 10 KB), and one
  user message from the session. It sets `store: false`, `stream: true` with `include_usage`, `reasoning_effort` and
  `max_completion_tokens`, and sends user content as parts.

## Next

The agent: config only, no runtime state (SPEC §3); the context (the log) belongs to the session, so many sessions
share one agent. Built by `agent.New(model, prompt, tools, config)`: fields `Model`, `Prompt`, `Tools`,
`Config{Effort, MaxTokens}`, which `Complete` sends. `Effort` is the SDK's `shared.ReasoningEffort` and its constants
(no own copy of the list; a second provider would give it its own type). `MaxTokens` is a `uint`. Both are optional
in the API: the zero value is not sent. `Complete` takes the agent (`client.Complete(ctx, agent, messages)`): it prepends the
prompt and sets model, tools and config, so `build` (D9) is its first lines; `provider` imports `agent`, never the
reverse. Streaming is a change of its own (it changes how `Complete` reads the response, D30).

## Open

- **Log entry shape.** A: an entry is a `provider.Message` (recommended: no new type, `build` prepends the prompt).
  B: an own `Entry` type converted by `build`. A capture of a second request (after a tool call) and of a response
  stream would show whether a reply carries what `Message` cannot hold.
- **Truncated replies.** `finish_reason == "length"` is a partial response (D10): `Complete` should return an error
  or the reason.
- **`developer` or `system` role.** Some OpenAI-compatible servers reject `developer`; `provider` has only `System`.

## Deferred

- Validating tool arguments against their schema in the router (the library is already a dependency): decided with
  dispatch.
