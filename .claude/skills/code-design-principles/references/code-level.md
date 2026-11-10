# Code-Level Readability: Expressions, Control Flow, Variables, Reorganizing, Tests

## Contents
- Expressions
- Control flow
- Variables
- Reorganizing code
- Tests

## Expressions [2]

Break giant expressions into pieces a human can hold in mind (3–4 things at a time). Fewer lines is not the goal: this one-liner takes longer to understand than the two lines below it [2]:

```c
assert((!(bucket = FindBucket(key))) || !bucket->IsOccupied());
```
```c
bucket = FindBucket(key);
if (bucket != NULL) assert(!bucket->IsOccupied());
```

1. **Explaining variable**: name a subexpression.

```python
if line.split(':')[0].strip() == "root":
    ...
```
```python
username = line.split(':')[0].strip()
if username == "root":
    ...
```

2. **Summary variable**: name a concept, even when the expression is understandable. `request.user.id == document.owner_id` has five variables; the idea is "does the user own the document?":

```java
final boolean user_owns_document = (request.user.id == document.owner_id);
if (user_owns_document) { /* user can edit this document... */ }
...
if (!user_owns_document) { /* document is read-only... */ }
```
Defining it at the top tells the reader this is a concept used throughout the function.

3. **De Morgan's laws**: "distribute the not and switch and/or":

```c
if (!(file_exists && !is_protected)) Error("Sorry, could not read file.");
```
```c
if (!file_exists || is_protected) Error("Sorry, could not read file.");
```

4. **Do not abuse short-circuit logic.** Clever `&&`/`||` that hides control flow reads worse than an `if` (the `assert` one-liner above is the example).

5. **Simplify a hard condition by solving its complement or stating it in words.** Example from the book's range-overlap problem: instead of enumerating all overlap cases, test the non-overlap cases (`other.end <= begin || other.begin >= end`) and negate.

6. **Ternary vs if/else**: `return exponent >= 0 ? mantissa * (1 << exponent) : mantissa / (1 << -exponent);` is compact, but the `if/else` form is less intimidating. Use `?:` for simple one-line cases, `if/else` otherwise [2].

## Control flow [2]

1. Changing value on the left, stable value on the right: `if (length >= 10)`, `while (bytes_received < bytes_expected)`. English: "if you are at least 18", not "if 18 is at most your age". (Constants on the left, "Yoda notation", guards against `=` typos but hurts reading; modern compilers warn about it.)
2. In `if/else`, put first: the positive case, the simple case, or the interesting case. Avoid negations: `if (debug)` not `if (!nodebug)`.
3. Prefer `while` to `do/while`: a loop's condition belongs at the top, where the reader looks.
4. A `for` that expresses the pattern beats manual guards around a `while`:

```c
for (Node* node = list->head; node != NULL; node = node->next)
    Print(node->data);
```
versus the equivalent `Node* node = list->head; if (node == NULL) return; while (node->next != NULL) {...} if (node != NULL) Print(node->data);`. Same behavior, longer time-till-understanding.

5. **Return early.** Handle failures and trivial cases first, then continue unindented. Multiple exits are fine when they make the flow shorter.

6. **Reduce nesting.** Every level of indentation is one more condition the reader must keep in mind. Two techniques: return early from functions, and `continue` in loops. Fowler's catalog example (Replace Nested Conditional with Guard Clauses) [3]:

```js
function getPayAmount() {
  let result;
  if (isDead)
    result = deadAmount();
  else {
    if (isSeparated)
      result = separatedAmount();
    else {
      if (isRetired)
        result = retiredAmount();
      else
        result = normalPayAmount();
    }
  }
  return result;
}
```
```js
function getPayAmount() {
  if (isDead) return deadAmount();
  if (isSeparated) return separatedAmount();
  if (isRetired) return retiredAmount();
  return normalPayAmount();
}
```

7. **Name the intent of a condition** (Decompose Conditional) [3]:

```js
if (!aDate.isBefore(plan.summerStart) && !aDate.isAfter(plan.summerEnd))
  charge = quantity * plan.summerRate;
else
  charge = quantity * plan.regularRate + plan.regularServiceCharge;
```
```js
if (summer())
  charge = summerCharge();
else
  charge = regularCharge();
```

8. **Avoid code that jumps unexpectedly**: threads, callbacks, exceptions, and function pointers make the path invisible. When you must use them, comment or name the flow so the reader can follow it.

## Variables [2]

1. **Eliminate variables that carry no weight.**
   - Useless temporary: `now = datetime.now(); root_message.last_view_time = now` becomes `root_message.last_view_time = datetime.now()`.
   - Intermediate result: `var remove_one = function(array, value_to_remove) { var index_to_remove = null; for (...) {...} if (index_to_remove !== null) array.splice(index_to_remove, 1); }` can `splice` inside the loop and return, with no `index_to_remove`.
   - Control-flow flag: a `done` boolean that only exits a loop becomes `break` or `return`. Fowler's version is Replace Control Flag with Break [3].
2. **Shrink the scope.** A variable visible everywhere forces the reader to track every place that touches it. Make class members private, avoid globals, move declarations down to first use, and split a large class into smaller ones that share less state.
3. **Prefer write-once variables.** `const`, `final`, immutable values: "the more places a variable is modified, the harder it is to reason about its current value."
4. **Use a variable once and only where its role is single.** If one name plays two roles, split it (Split Variable [3]).

## Reorganizing code [2]

1. **Extract unrelated subproblems.** Ask of each block: what is the high-level goal? Which lines are not directly about it? Move those into a function with a general name. Example: a function that finds the closest place on a sphere to a given point can pull out `ComputeDistanceBetween(lat1, lon1, lat2, lon2)`.
2. **Do one task at a time.** List the tasks a function performs (read input, validate, update totals, format output) and separate them into blocks or functions.
3. **Turn thoughts into code.** Describe the logic aloud in plain words, then write code that matches the words. If the words are clumsy, the code will be too.
4. **Write less code.** Do not build features nobody needs. Question and simplify requirements. Use the standard library. A smaller codebase is easier to hold in mind.
5. Keep helper functions general-purpose: they should know nothing about the caller's project.

## Tests [2]

1. **Make each test case read in a line or two**: input and expected output, built through helper functions (see `CheckFullName` in naming-and-layout.md, Aesthetics).
2. **Use the simplest input that exercises the case.** Prefer small, readable values over large random ones.
3. **Failure messages must name the input and the expected/actual values.**
4. **Name tests for behavior**: `Test_EmptyList_ReturnsZero`, not `Test1`.
5. **Design for testability**: few dependencies, minimal global state, a small interface. Code that is hard to test is usually hard to read.
6. Test the behavior that matters; do not chase 100% coverage for its own sake.
