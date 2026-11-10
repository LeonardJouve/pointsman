---
name: code-design-principles
description: Good code design principles. Use when writing, reviewing, simplifying, renaming, commenting, or restructuring code in any language.
---

# Code Design Principles

Code is written for humans first and machines second. Every rule serves one goal: reduce the time a human needs to understand code well enough to change it safely. Language-independent.

Sources: [1] Ousterhout, *A Philosophy of Software Design*, 2nd ed., 2021. [2] Boswell & Foucher, *The Art of Readable Code*, 2011. [3] Fowler, *Refactoring*, 2nd ed., 2018.

## Core ideas

1. **The metric is time-till-understanding** [2]. "Understand" is a high bar: the reader can change the code, spot bugs, and see how it interacts with the rest. The reader is often you, six months later.
2. **Humans hold three or four things at a time** [2]. Code a compiler parses easily can still overload a reader.
3. **Fewer lines is not the goal.** A comment can make code faster to read even though it adds a line. A clever one-liner can take longer to read than two plain lines [2].
4. **Readers skim, then dive.** They scan names, signatures, and comments first. Design so the skim gives the right picture [1][2].
5. **Working code is not enough** [1]. Complexity is incremental: no single shortcut breaks a system, but hundreds do. Sweat the small stuff.
6. **Stop when it is readable.** Do not polish code that is already easy to understand [2].

Self-check before finishing any change: *If a colleague who has never seen this file read it once, would they understand it and dare to change it?*

## Where to look

Load a reference file only when the task touches its topic.

| Task involves | Read |
|---|---|
| Naming things; grouping, alignment, ordering | [references/naming-and-layout.md](references/naming-and-layout.md) |
| Writing, judging, or deleting comments | [references/comments.md](references/comments.md) |
| Expressions, conditionals, loops, nesting, variables, splitting functions, test style | [references/code-level.md](references/code-level.md) |
| Designing a module or interface; layering; error handling; the red-flag list | [references/module-design.md](references/module-design.md) |
| Behavior-preserving restructuring; smell to remedy lookup; step-by-step mechanics | [references/refactoring.md](references/refactoring.md) |

Cross-references: reference files cite each other by file name and section.

## Rules to apply without opening a reference

**Names:** use specific verbs (`Fetch`, not `Get`, when it hits the network); put units in the name (`delay_secs`); no generic names (`tmp`, `retval`, `data`); booleans read as questions, never negated; a name must not suggest something false or cheap-looking work that is expensive. Test: *what else could this mean?*

**Comments:** record what the code cannot: intent, rationale, units, pitfalls, big picture. Delete comments that restate code. If a comment excuses a bad name, rename instead. Keep interface comments free of implementation detail. Update comments in the same change as the code.

**Expressions:** if an expression exceeds 3–4 things, name a subexpression (explaining variable) or a concept (summary variable). Prefer `if/else` to clever `&&`/`||` or nested ternaries.

**Control flow:** changing value on the left, stable value on the right. Positive and simple case first; avoid negations. Return early; use guard clauses and `continue` to cut nesting.

**Variables:** drop temporaries and flags that carry no weight; shrink scope; prefer write-once; one role per variable.

**Functions:** one task at a time; extract unrelated subproblems into general-purpose helpers; a "get" must not have side effects.

**Design:** deep modules (small interface, much hidden behind it); a simple interface beats a simple implementation; pull complexity downward rather than exporting it as config; define errors out of existence where semantics allow; no pass-through layers; design significant interfaces twice.

**Scope:** write less code. Do not build features nobody needs (YAGNI). Optimize only measured hot spots.

## Review checklist

Ask in order. Stop at the first "yes" and fix it before continuing.

1. Would a new reader need a comment to decode a name? Rename.
2. Is any comment restating code or covering for a bad name? Remove or rename.
3. Is anything surprising, slow, or dangerous undocumented? Comment it.
4. Is any expression bigger than a reader can hold? Add an explaining or summary variable.
5. Is anything nested deeper than needed, or negated? Guard clauses, De Morgan.
6. Does a variable live longer or change more often than needed? Narrow it.
7. Does a function do two jobs, or does a "get" have side effects? Split Loop, Separate Query from Modifier, one task at a time.
8. Does any red flag (module-design.md) or smell (refactoring.md) apply? Fix the design, not the symptom.
9. Is the code already easy to understand? Stop. Do not polish further.

## Procedures

### Add a feature to messy code

1. Run tests. Add characterization tests if missing.
2. Refactor until the change is easy ("make the change easy, then make the easy change"). Commit.
3. Make the behavior change. Test. Commit.
4. Leave touched code a little better than found.

Never add behavior and refactor in the same step. For disciplined small-step mechanics, read refactoring.md.

### Design a new module

1. List the decisions it will hide.
2. Sketch two interfaces.
3. Write the interface comments first.
4. Pick the one that is deeper and simpler to use.
5. Implement; pull complexity into the body.
6. Run the review checklist.

### Review someone else's code

1. Read as the reader would: names and signatures first, then bodies.
2. Note where you paused or reread; each pause is a finding.
3. For each, name the smell or red flag and propose the smallest refactoring.
4. Keep structural changes and behavior changes in separate commits.

## Precedence

1. Time-till-understanding wins over compactness, cleverness, and tidy factoring [2].
2. Simple interfaces win over simple implementations [1].
3. Follow local convention unless it is the cause of the problem.
