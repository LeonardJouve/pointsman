# Agent Router: Specification

Status: draft · Date: 2026-11-10 · Name: TBD

This document covers the architecture and its decisions. The implementation state (what is built, its facts and open
points) is in [docs/design.md](docs/design.md).

---

## 1. Purpose

A shared harness that runs AI agents. The router runs each agent's loop: it calls the model, sends the model's tool calls to a dispatcher and records everything in a per-session log.

Tools are decoupled from the harness. They live in the dispatcher, a separate process reached over gRPC, and the router never executes tool logic itself.

## 2. Goals and non-goals

**Goals**
- Run many agent sessions concurrently with strict isolation between them.
- Keep the router the single owner of session state.
- Treat the model and the dispatcher as workers the router calls on behalf of a session.
- Survive router crashes without corrupting sessions.
  Set aside for now: the store is a temporary folder per start (load experiment).
- Grow step by step, without a big up-front design.

**Non-goals (for now)**
- Long-lived agent memory across sessions.
- Multiple router instances.
- Multi-tenancy, billing and quotas.
- Agent-to-agent workspace sharing.

---

## 3. Core concepts

| Concept | Definition |
|---|---|
| **Agent** | A versioned configuration: model, system prompt, tools, generation settings. It has no runtime state. |
| **Session** | One run of an agent. It has a unique ID assigned by the router and its own log. |
| **Session log** | An append-only, ordered list of entries for one session. The single source of truth. |
| **Model turn** | One call to the model and its complete response. |
| **Tool call** | One request to run a tool, issued by the model and executed by the dispatcher. |
| **Dispatcher** | A separate process that holds the tool instances and runs tool calls. The router reaches it over gRPC. |
| **Tool** | A function in the dispatcher. It runs directly, or starts a container for the call. |
| **Container** | A throwaway Docker container started by a tool for one call and removed after it. |
| **Workspace** | A session's files. It lives outside any container and is mounted into the containers of that session's calls. |
| **Router** | The component that owns sessions, builds model requests, sends tool calls and appends to logs. |

Identity levels: `agent → session → call`.

---

## 4. Architecture

```
            ┌───────────────────────────────────────────┐
 Clients ──▶│                  Router                   │
            │                                           │
            │  sessions ── session logs (source of truth)│
            │      │                                    │
            │      ├─ Context(agent, log) ─▶ provider ──┼──▶ Model API
            │      │                                    │
            │      └─ tool call (gRPC) ─────────────────┼──▶ Dispatcher
            └───────────────────────────────────────────┘     │
                                                              ├─ tool runs directly
                                                              └─ tool starts a container
                                                                 (session workspace mounted)
```

### Components
- **Session manager:** creates sessions, assigns IDs, serializes work per session.
- **Log store:** append-only storage of session logs.
- **Context builder:** a pure function `Context(agent, log) → model request`.
- **Provider adapter:** translates a model request into one provider's API format and back.
- **Provider limiter:** a shared concurrency cap and retry/backoff per provider.
- **Dispatcher:** a separate process; holds the tool instances, runs each call and returns its result.

---

## 5. The session loop

```
loop:
    request  = Context(agent, log)                # pure
    response = model(request)                     # only side effect on the model side
    log      = log + response
    if response has no tool calls: stop
    for each tool_call in response:
        result = dispatch(session_id, tool_call)  # gRPC call to the dispatcher
        log    = log + result
```

The model holds no session state. The dispatcher holds only the session's workspace.

---

## 6. Decisions

Decision numbers are stable; a missing number is a removed decision.

### Scope

**D1. The router is the runtime.** It runs agent loops, not just tool calls. Agents are configuration.
*Why:* tools are decoupled through the dispatcher, so owning the loop doesn't couple tools to the harness.

**D2. The router stays thin.** It doesn't build prompts beyond assembling config and log, and it doesn't implement memory or tool logic.

### Identity and isolation

**D3. The router assigns session IDs.** They never appear in model context or tool arguments, and the model can't see, write or choose them.
*Why:* if the ID were in model-controlled text, a prompt injection could target another session.

**D4. The router attaches `(session_id, call_id)` to every dispatched call** in the gRPC request, outside the tool arguments.

**D5. One container per tool call, never shared.** Nothing in a container outlives the call; the workspace is the only state kept between calls.

**D6. Isolation comes from the mount.** A tool's container mounts only its session's workspace, so a call can't reach another session's files.

**D7. Credentials stay out of model context.** Model API keys live only in the router. Tool credentials are injected by the dispatcher at execution time and scoped to the session.

### Session log

**D8. The session log is the single source of truth.** It is append-only and ordered.

**D9. Context building is a pure function** of the agent config and the session log. The same inputs always give the same request.
*Why:* it can be tested without a model or dispatcher, and it enables replay and fork.

**D10. Entries are appended only when complete.** A partial model response is never written to the log.

**D11. No provider-side conversation state.** Provider features that remember conversations between calls are not used.
*Why:* it would split state between two places and break replay, fork, model switching and recovery.

**D12. Each log entry records the agent config version** that produced it.

### Model side

**D13. Model calls are stateless.** Each call sends the full context built from the session's own log.

**D14. One model call in flight per session.** A session is a strict sequence of model turn, tools, model turn.

**D15. A shared limiter per provider** caps concurrent requests across all sessions and sits behind a queue.

**D16. Transient provider errors (rate limits, overload) are retried inside the router** with backoff. They never reach the model's context.

**D17. Tool call IDs are mapped explicitly.** The log records `model_tool_id ↔ call_id ↔ result`, and results are returned to the model under the model's own ID.

**D18. Stable request order:** system prompt, then tool schemas, then the log. This lets provider prompt caching work across turns and across sessions of the same agent.

**D19. Streaming is for watchers only.** Tokens may be forwarded live to clients, but only the complete response is logged.

### Tool side

**D22. Tool calls are serialized per session by default.** Parallel execution is allowed only for tools marked parallel-safe.

**D23. Tool calls carry idempotency keys.** Unlike model calls, tool calls can have side effects, so a retry must not run twice. The `call_id` is the key.

**D24. Tool errors are separated from infrastructure errors.** A tool error is a result with a non-zero code: it goes to the model, which can adapt. An infrastructure error (dispatcher down, Docker failure, unknown tool) is a gRPC status error: the router handles it and it never reaches the model.

**D25. Large tool outputs are stored as artifacts.** The log keeps a truncated view plus a reference to the full output.

### Lifecycle and failure

**D28. Model calls are safe to repeat** because they have no side effects. After a router crash, a session resumes from its log and the interrupted model call is retried.

### Waiting on inference

**D29. Holding a connection during inference is acceptable for now.** It uses non-blocking I/O, so a waiting call costs a socket, not a thread.

**D30. Model responses are streamed** to keep connections active through proxy idle timeouts.

**D31. Decoupling is deferred but supported.** If needed, model calls become queued jobs run by workers that append to the session log, the same pattern as tool calls. The design doesn't change, only who makes the call.

**D32. Clients don't hold connections for a whole run.** Starting a run returns the session ID immediately. Clients subscribe to the session's events and can reconnect from the last event they saw.

---

## 7. Client interface (conceptual)

| Operation | Effect |
|---|---|
| Start session | Creates a session for an agent config and returns its ID immediately. |
| Send message | Appends a user message and wakes the session. |
| Stream events | Subscribes to log entries, resumable from a position. |
| Cancel | Stops the session and its running tool calls. |

Planned later: approve/deny for gated tools, and fork from step N.

---

## 8. Deferred

- Multiple router instances, with session-affine routing.
- Model calls as queued jobs (D31).
- Batch inference for evals and replays.
- Hooks (before/after model call, before/after tool call, on stop) and pluggable loop strategies.
- Context compaction policies.
- Workspace snapshot and fork.
- Shared workspaces between sessions.
- Budgets, quotas and per-session cost attribution.

## 9. Open questions

- Are agents task-scoped (one session, then done) or long-lived with memory across sessions?
- What are the main workloads: stateful coding/shell agents or mostly stateless API agents?
