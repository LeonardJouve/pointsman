# Agent Router: Specification

Status: draft · Date: 2026-11-10 · Name: TBD

This document covers system architecture and design decisions only. Implementation choices (language, libraries, storage engine, sandbox technology) are deliberately left open.

---

## 1. Purpose

A shared harness that runs AI agents. The router runs each agent's loop: it calls the model, dispatches the model's tool calls to sandboxed executors and records everything in a per-session log.

Tools are decoupled from the harness. They run in sandboxed environments behind a uniform interface, and the router never executes tool logic itself.

## 2. Goals and non-goals

**Goals**
- Run many agent sessions concurrently with strict isolation between them.
- Keep the router the single owner of session state.
- Treat models and sandboxes as stateless workers the router calls on behalf of a session.
- Survive router crashes without corrupting sessions.
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
| **Agent** | A versioned configuration: model, system prompt, allowed tools, limits, sandbox template. It has no runtime state. |
| **Session** | One run of an agent. It has a unique ID assigned by the router and its own log. |
| **Session log** | An append-only, ordered list of entries for one session. The single source of truth. |
| **Model turn** | One call to the model and its complete response. |
| **Tool call** | One request to run a tool, issued by the model and executed by a sandbox or worker. |
| **Sandbox** | An isolated execution environment bound to exactly one session. |
| **Router** | The component that owns sessions, builds model requests, routes tool calls and appends to logs. |

Identity levels: `agent → session → call`.

---

## 4. Architecture

```
            ┌──────────────────────────────────────────┐
 Clients ──▶│                 Router                    │
            │                                           │
            │  sessions ── session logs (source of truth)│
            │      │                                    │
            │      ├─ build(agent, log) ─▶ provider adapter ─▶ Model API
            │      │                                    │
            │      └─ routing table ─▶ tool dispatch ───┼──▶ Sandboxes (session-bound)
            │                                           │──▶ Workers (stateless tools)
            └──────────────────────────────────────────┘
```

### Components
- **Session manager:** creates sessions, assigns IDs, serializes work per session.
- **Log store:** append-only storage of session logs.
- **Context builder:** a pure function `build(agent_config, session_log) → model request`.
- **Provider adapter:** translates a model request into one provider's API format and back.
- **Provider limiter:** a shared concurrency cap and retry/backoff per provider.
- **Tool registry:** tool schemas plus each tool's routing kind (stateless or session-bound).
- **Routing table:** `session_id → { sandbox_id, endpoint, token, status, last_used }`.
- **Sandbox manager:** creates, tracks and destroys sandboxes.

---

## 5. The session loop

```
loop:
    context  = build(agent_config, log)           # pure
    response = model(context)                     # only side effect on the model side
    log      = log + response
    if response has no tool calls: stop
    for each tool_call in response:
        result = dispatch(session_id, tool_call)  # routed by session_id
        log    = log + result
```

Both sides route on the session ID. The model and the sandboxes never hold session state.

---

## 6. Decisions

### Scope

**D1. The router is the runtime.** It runs agent loops, not just tool calls. Agents are configuration.
*Why:* tools are already decoupled through sandboxes, so owning the loop doesn't couple tools to the harness.

**D2. The router stays thin.** It doesn't build prompts beyond assembling config and log, and it doesn't implement memory or tool logic.

### Identity and isolation

**D3. The router assigns session IDs.** They never appear in model context or tool arguments, and the model can't see, write or choose them.
*Why:* if the ID were in model-controlled text, a prompt injection could target another session.

**D4. The router attaches `(session_id, call_id)` to every dispatched call** in an envelope outside the tool arguments.

**D5. One sandbox per session, never shared.**

**D6. Isolation is enforced on both ends.** On creation, each sandbox gets a per-session token and rejects calls without it, so a routing bug can't silently cross sessions.

**D7. Credentials stay out of model context.** Model API keys live only in the router. Tool credentials are injected into the sandbox at execution time and scoped to the session.

### Session log

**D8. The session log is the single source of truth.** It is append-only and ordered.

**D9. Context building is a pure function** of the agent config and the session log. The same inputs always give the same request.
*Why:* it can be tested without a model or sandbox, and it enables replay and fork.

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

**D20. Each tool declares a routing kind at registration:**
- *stateless*: sent to any available worker;
- *session-bound*: sent to the session's sandbox.

**D21. Sandboxes are created lazily** on a session's first session-bound call and recorded in the routing table.

**D22. Tool calls are serialized per session by default.** Parallel execution is allowed only for tools marked parallel-safe.

**D23. Tool calls carry idempotency keys.** Unlike model calls, tool calls can have side effects, so a retry must not run twice.

**D24. Tool errors are separated from infrastructure errors.** Tool errors go to the model, which can adapt. Infrastructure errors are handled by the router and never reach the model.

**D25. Large tool outputs are stored as artifacts.** The log keeps a truncated view plus a reference to the full output.

### Lifecycle and failure

**D26. Sandboxes are destroyed** when a session ends or after an idle timeout, and their routing entries are removed.

**D27. Sandbox loss is explicit.** If a sandbox dies, the session is marked `environment_lost` and the agent is told in the next tool result. A call is never silently routed to a fresh, empty sandbox.

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
| Cancel | Stops the session and releases its sandbox. |

Planned later: approve/deny for gated tools, and fork from step N.

---

## 8. Deferred

- Multiple router instances, with a shared routing table or session-affine routing.
- Model calls as queued jobs (D31).
- Batch inference for evals and replays.
- Hooks (before/after model call, before/after tool call, on stop) and pluggable loop strategies.
- Context compaction policies.
- Sandbox snapshot and fork.
- Shared workspaces between sessions.
- Budgets, quotas and per-session cost attribution.

## 9. Open questions

- Are agents task-scoped (one session, then done) or long-lived with memory across sessions?
- What are the main workloads: stateful coding/shell agents or mostly stateless API agents?
- Should the agent-facing tool surface follow an existing protocol for compatibility?
- What is the exact shape of a log entry? This is the next thing to pin down.
