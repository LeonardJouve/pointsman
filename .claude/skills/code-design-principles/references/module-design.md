# Complexity, Module and Interface Design [1]

## Contents
- Complexity
- Principles
- Deep and shallow modules
- Techniques
- Red flags

## Complexity

Complexity is anything in the structure of a system that makes it hard to understand or modify. Complexity is incremental: no single shortcut breaks a system, but hundreds do.

| Symptom | What the reader experiences | Example |
|---|---|---|
| Change amplification | A small change needs edits in many places. | A color constant hard-coded in 40 files. |
| Cognitive load | Too much must be known to make a change. | A function that needs three globals set and a particular call order. |
| Unknown unknowns | It is unclear what must change or what may break. Worst of the three. | Two modules silently assume the same date format; nothing says so. |

Two causes: **dependencies** (code cannot be understood or changed in isolation) and **obscurity** (important information is not obvious).

## Principles

1. Working code isn't enough: passing tests does not mean the design is acceptable.
2. Make continual small investments to improve design (roughly 10–20% of effort); do not wait for a rewrite.
3. Modules should be deep: much functionality behind a small interface.
4. Make the most common usage as simple as possible; callers should not learn rare features first.
5. A simple interface matters more than a simple implementation: pay the cost once in the module, not in every caller.
6. General-purpose modules are deeper. Separate general-purpose and special-purpose code.
7. Different layers should have different abstractions. Adjacent layers with the same abstraction are a smell.
8. Pull complexity downward; define errors and special cases out of existence; design it twice.
9. Comments describe what is not obvious from the code.
10. Design for ease of reading, not ease of writing.
11. The increments of development are abstractions, not features.

## Deep and shallow modules

A module has an interface (what callers must know: signature, side effects, ordering rules, exceptions, performance assumptions) and an implementation. Think of a rectangle: the top edge is the interface, the area is the functionality. Deep modules give a lot of area for a short edge.

- **Deep:** Unix file I/O. Five calls (`open`, `read`, `write`, `lseek`, `close`) hide disk layout, buffering, permissions, caching, device drivers, and concurrency.
- **Shallow:** Java's I/O stack. To read a serialized object from a file the caller writes `new ObjectInputStream(new BufferedInputStream(new FileInputStream(name)))`. Buffering is a default need, yet callers must ask for it. This is **classitis**: many tiny classes, each simple, whose sum is complex.
- **Shallow method:** `private void addNullValueForAttribute(String attribute) { data.put(attribute, null); }` The method name is longer than the body and adds nothing.

## Techniques

1. **Hide information.** Each module encapsulates a few design decisions (file format, algorithm, data structure, protocol) inside its implementation. If two modules both know the same decision, that is Information Leakage.
2. **Do not split by time.** A `FileReader`, `FileParser`, `FileWriter` trio that all know the file format has Temporal Decomposition. One module that owns the format fits information hiding.
3. **Somewhat general-purpose interfaces.** Implement for today's needs; shape the interface so it is not tied to today's use. Example: a text editor's `delete(Position start, Position end)` and `insert(Position, String)` serve every UI feature, whereas `deleteSelection()`, `backspace()`, `deleteKey()` each encode one UI action. Questions: What is the simplest interface that covers all current needs? In how many situations will this method be used? Is it easy to use for today's needs?
4. **Separate general and special-purpose code.** Push special cases up to a caller or down to a dedicated layer. A text-editor core should not know about UI buttons.
5. **Different layer, different abstraction.** Warning signs:
   - *Pass-through method:* `def get_user(id): return self.repo.get_user(id)`. Remove it, expose the lower layer, or give the layer a real job.
   - *Pass-through variable:* a value threaded through a long call chain for a deep callee. Replace with a context object.
   - *Decorator/wrapper layers* that add little.
6. **Pull complexity downward.** Do not add a config parameter just to avoid choosing a sensible default. A retry timeout the module can compute is better than a required argument every caller must guess.
7. **Together or apart?** Combine code that shares information, simplifies the interface, or removes duplication. Split when the parts are general versus special. Splitting a method does not help unless each piece has a clean abstraction. A long method with a simple signature and clear flow is fine; conjoined fragments that cannot be read alone are not.
8. **Define errors out of existence.** Change semantics so the error case is not an error:
   - Unix `unlink` on an open file succeeds; the data lives until the last close. Windows refuses, so every caller handles "file in use".
   - `substring(begin, end)` that clamps to the string bounds instead of throwing.
   - Otherwise: *mask* (handle inside the module), *aggregate* (one handler for many exceptions), or *crash* (unrecoverable). Each exception type in an interface raises caller cost.
9. **Design it twice.** For any significant interface, sketch two radically different designs. Compare by simplicity of interface and ease of use. Even if the first is best, the comparison teaches why.
10. **Choose names and write comments before the body** (comments.md, item 11). It exposes design flaws while they are cheap to fix.
11. **Consistency.** Do similar things in similar ways. Follow local convention.
12. **Obviousness.** Code is obvious if the reader's first guess about behavior is right. Obviousness is lost through event-driven indirection, generic containers (`Pair<Integer, Integer>` returned for a range), and declared types that differ from runtime types.
13. **Inheritance.** Prefer composition. Interface inheritance is fine; implementation inheritance couples a subclass to parent internals.
14. **Getters/setters on every field** expose representation and are shallow. Use only where the abstraction needs them.
15. **Tactical vs strategic.** A tactical programmer asks "how fast can I get this working?" and adds a little complexity with each change. A strategic programmer treats the design as the product. Test-driven development alone is tactical: tests are needed, but they do not replace design.

## Red flags

Run this list on every change. Any "yes" needs a fix or a stated reason.

| Red flag | Definition | Example |
|---|---|---|
| Shallow Module | The interface is not much simpler than the implementation. | A class whose methods mirror its fields one-to-one. |
| Information Leakage | A design decision is reflected in multiple modules. | Two files both know the on-disk file format. |
| Temporal Decomposition | Structure follows execution order, not information hiding. | Separate `read`, `parse`, `write` classes that share the same format knowledge. |
| Overexposure | The API forces callers to know rarely used features to use common ones. | A constructor needing five tuning options to open a plain file. |
| Pass-Through Method | Does almost nothing except pass arguments to a method with a similar signature. | `def get_user(id): return self.repo.get_user(id)` |
| Repetition | A nontrivial piece of code is repeated. | The same 10-line validation in four handlers. |
| Special-General Mixture | Special-purpose code is not cleanly separated from general-purpose code. | A generic cache with a branch for one report type. |
| Conjoined Methods | One cannot be understood without reading the other. | Two methods that share hidden state set in a particular order. |
| Comment Repeats Code | The comment's information is obvious from adjacent code. | See comments.md, Do not write. |
| Implementation Documentation Contaminates Interface | An interface comment describes implementation details. | See comments.md, Do not write. |
| Vague Name | The name conveys little. | `data`, `tmp`, `retval` |
| Hard to Pick Name | No precise, intuitive name exists. | Often the entity does two jobs. |
| Hard to Describe | Complete documentation must be long. | The abstraction is probably wrong. |
| Nonobvious Code | Behavior or meaning cannot be understood easily. | See Techniques, item 12. |
