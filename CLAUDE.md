# pointsman

How work is done in this repository and what the user values.

The goal: a codebase one person can read and maintain by hand. Codebases grow out of control when agents write them
fast. Less code is the expected outcome; there is no time pressure, a slow decision that holds beats a fast one
that gets redone.

## Roles and flow

- **Coordinator** (the main session): the only one who talks to the user.
  Keeps the order of the pieces and the review queue. **Writes no code.** Agents never ask the user: they stop and return
  their questions to the coordinator, who turns them into one brief.
- **Developer agent**: answer a question about the codebase, or implement one approved design in it as one atomic change.
  At most two changes at once, never on the same files. Git is read-only for them: no `add`, `rm`, `restore`,
  `stash`, `checkout`, `reset` or `commit`, even to undo a mistake; files are deleted with plain `rm`.
- Subagents run on Sonnet, not Opus. The user owns git: staging, commits, pushes. Changes wait in the working tree.

One change: **study -> one decision with a recommendation -> the user decides -> the developer agent implements -> the user
reviews, asks for changes, commits.**
Studies run ahead; at most two changes wait for review.

A change is one concept (a package, a type, a boundary), stands alone, leaves build/vet/tests green, and is
reviewable in about 15 minutes (a few hundred lines with tests). Build bottom-up: the piece that depends on nothing
first.

## Project skills

- **code-design-principles**: the design rules behind the values below. The developer agent loads it before writing
  or reviewing code; the coordinator checks a change against it before presenting.
- **show-me**: invoked by the user only (`/show-me`), to see the current topic as a diagram or code-shape sketch.

## Putting a decision to the user

The user is the bottleneck and does not know every piece of the code. So:
- **One piece at a time.** Never a batch. Take the next piece, defer the other questions until their piece comes up.
- **Explain from scratch**: what it does and where it sits in the flow, then the code a caller would write.
- **Options with their maintenance cost, and one recommendation.** Say what each option rejects.
- **Only an explicit answer to the exact question is a decision.** "ok", a follow-up question, or an answer to one
  point of a bundle approves nothing else. Never send an agent a point that was not decided.
- Nothing settled by the record, the code or an earlier answer: ask, don't pick silently. Agents too: if the design
  does not cover something, they stop and return the question.
- A decision can be revisited when the facts change. Say so and show the facts; don't defend the old choice.
- The user often decides by rewriting the code by hand or by saying "keep my changes": that is a decision too.

## Design values (each with its why)

- **Hand-maintainable, minimal.** Climb the Ponytail ladder before writing: does it need to exist (YAGNI)? the
  stdlib? an existing dependency? one line? Only then the minimum. Flat layout, name-keyed dispatch, few comments.
- **Structure over small performance or size.** Never bend the design for a minor gain; rank by correctness and
  clarity. A channel, goroutine or table must earn its place: synchronous and plain by default, revisit when a real
  need (`select`, a second user) appears.
- **No speculative structure.** No abstraction, option, column, type, guard or package before the change that needs it.
  A table of one row is a literal; it becomes a table when its rows come. When asked "should this be a struct?",
  weigh it against this first.
- **Plain code.** Short comments, no file-level essays; no helper used once; types hold only the states that exist
  (a bool for a direction, unsigned for counters); no jargon.
- **Names are the user's call.** Propose them with a caller-side example; short
  folder names; one name per idea. A proposed name not objected to is not yet decided; say so.
- **Drop what nothing uses.** What no caller needs is not carried over; re-add it when something does.
- **One source of truth.** One type, one constant, one place per fact; link or merge rather than copy.
- **Dependencies up to date**: bump the `go` line rather than pin an old dependency.

## What the user removes at review (check before presenting)

- A type or struct written twice; several names for one idea; a literal or list written in two places.
- Values stored that can be computed (a hash field next to a name); a type that only wraps a slice for one helper.
- Single-use helpers of a few lines; long functions with a `switch` (one function per case, a method per message).
- Guards against mistakes no fact makes, and checks a later step already catches (regexps, field validation).
- Structure built ahead of its change (one-row tables, columns for the next change, single-instance structs).
- Roles mixed in a file: one file per role. Maps: one entry per line; sets are `map[T]struct{}`.
- Vague or jargon names; and mini-languages in strings where structured data (YAML keys) reads better.
- Lines over 120 columns; style that departs from the committed code (blank line before a return and between blocks).

## Documentation

- **No duplicated content**: look for what exists first; add to the existing file or link. Never copy an external
  repo into this one.
- Minimal docs, short, **in English**.
- **A design record (`docs/`) holds the current state only**: the design and its reasons, the facts, what is next
  and open. When a decision is revisited, edit it in place; no history of superseded options, a rejected option only
  when it explains the current choice in a few words. Another session must be able to take the work back from it.

## Working with the user's edits

- Before briefing an agent, **re-read the files as they are now**; the user may have moved or rewritten code.
- Where the user changed code by hand, it stays where and as they put it. Name each such spot (file, function) in the
  brief and say it stays; prefer their layout over the plan's. A cleanup must not move it back.
- After the agent reports, verify with grep/diff that those spots are untouched.
- When the user says they changed code, read the changes, fix the tests and the record together, and adapt to their
  version rather than restoring yours.

## Briefing agents

Give the developer agent: the user-edit spots, the review list above, the README update, the checks (build, vet, tests,
gofmt; CRLF alone is not a gofmt finding), and the git rule (read-only, plain `rm` to delete) in every brief that
deletes or moves files.
Ask for `git diff --stat`, what was verified and how, and open questions.
