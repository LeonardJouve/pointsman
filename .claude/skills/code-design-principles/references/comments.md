# Comments

## Contents
- Purpose
- Do not write
- Do write (12 items)

## Purpose

"The purpose of a comment is to help the reader know what the writer knew when writing the code" [2]. Good code with no comment beats bad code with a good comment [2]. Comments capture what code cannot: intent, rationale, units, constraints, abstraction [1].

A comment can make code faster to read even though it adds a line [2]:

```c
// Fast version of "hash = (65599 * hash) + c"
hash = (hash << 6) + (hash << 16) - hash + c;
```

## Do not write

| Anti-pattern | Example | Fix |
|---|---|---|
| Restating the code | `// Set the profit member to a new value` above `void SetProfit(double profit);` | Delete. |
| Declaration in sentence form | `// Find the Node in the given subtree, with the given name, using the given depth.` above `FindNodeInSubtree(Node* subtree, string name, int depth)` | Delete, or say something new (item 1 below). |
| Crutch for a bad name | `// Releases the handle for this key. This doesn't modify the actual registry.` above `DeleteRegistry(RegistryKey* key)` | Rename: `ReleaseRegistryHandle`. A name is seen at every call site; a comment is not. |
| Crutch for a vague name | `// Enforce limits on the Reply as stated in the Request...` above `CleanReply` | Rename: `EnforceLimitsFromRequest`. |
| Implementation detail in an interface comment | "Uses a hash map internally and resizes at 75% load" on a public `lookup()` | Move to an implementation comment. |

Exception: a comment that is faster to read than the code is worth keeping even if it adds no new fact [2]:

```python
# remove everything after the second '*'
name = '*'.join(line.split('*')[:2])
```

## Do write

1. **Say what the signature cannot** [2]:

```cpp
// Find a Node with the given 'name' or return NULL.
// If depth <= 0, only 'subtree' is inspected.
// If depth == N, only 'subtree' and N levels below are inspected.
Node* FindNodeInSubtree(Node* subtree, string name, int depth);
```

2. **Director's commentary: why this way and not another** [2]:

```cpp
// Surprisingly, a binary tree was 40% faster than a hash table for this data.
// The cost of computing a hash was more than the left/right comparisons.
```
```cpp
// This heuristic might miss a few words. That's OK; solving this 100% is hard.
```
The second stops a reader from hunting for a bug that is not a bug.

3. **Flaws, with markers** [2]: `TODO:` not done yet · `FIXME:` known-broken · `HACK:` admittedly inelegant · `XXX:` danger, major problem. Say how to start: `// This class is getting messy. Maybe we should create a 'ResourceNode' subclass to help organize things.`

4. **The story behind a constant**: `// 1 MB: larger uploads time out behind the load balancer` or `NUM_THREADS = 8 // as long as it's >= 2 * num_processors, that's good enough`.

5. **Pitfalls: what is surprising? how can this be misused?** [2]

```cpp
// Calls an external service to deliver email.  (Times out after 1 minute.)
void SendEmail(string to, string subject, string body);
```
```python
# Runtime is O(number_tags * average_tag_depth), so watch out for badly nested inputs.
def FixBrokenHtml(html): ...
```
Without the first comment, someone calls `SendEmail` inside an HTTP request handler and the web app hangs when the mail service is down.

6. **Big picture, at file or class level** [2]. Imagine touring a new teammate: "This class looks complicated, but it's really just a smart cache. It doesn't know anything about the rest of the system."

7. **Summary comments inside long functions** [2]. If the chunks separate cleanly, extract functions instead:

```python
def GenerateUserReport():
    # Acquire a lock for this user
    ...
    # Read user's info from the database
    ...
    # Write info to a file
    ...
    # Release the lock for this user
```

8. **Precise, compact language** [2]:
   - Avoid pronouns with several possible referents. `// Insert the data into the cache, but check if it's too big first.` becomes `// Insert the data into the cache, but check if the data is too big first.`
   - Use a short example for tricky behavior: `// Strip('abba', 'a') -> 'bb'` shows that it removes every leading and trailing character in the set.
   - State the intent, not the literal step: `// Iterate over the list in reverse order` instead of `// Go through the list, last to first`, and `// Display each item in the cart, showing the most expensive first` rather than a restatement of the loop.
   - Name input/output corner cases: `// Returns 0 if the list is empty`.

9. **Comment the what, the how, or the why**, whichever helps the reader. "Comment the why, not the what" is too simplistic [2].

10. **Beat writer's block** [2]: write the half-baked thought, then sharpen each phrase.
    `// Oh crap, this stuff will get tricky if there are ever duplicates in this list.` becomes `// Careful: this code doesn't handle duplicates in the list (because that's hard to do)`.

11. **Write the interface comment first** [1]. If you cannot describe an abstraction simply and completely, the abstraction is wrong. A long comment needed for completeness (Hard to Describe) signals a design problem.

12. **Keep comments next to the code they describe, and update them in the same change.** Cross-module decisions go in one agreed place, not scattered [1].
