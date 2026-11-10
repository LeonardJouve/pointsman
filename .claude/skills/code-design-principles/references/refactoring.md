# Refactoring [3]

## Contents
- Definition and discipline
- When to refactor
- Smells and remedies
- Mechanics of the core moves
- Catalog examples
- Refactoring and performance
- Catalog index

## Definition and discipline

Refactoring is changing the internal structure of code without changing its observable behavior, through a sequence of small steps, each of which keeps the code working.

1. **Self-testing code first.** Run the tests before you start. If none exist, write tests that pin current behavior (characterization tests) around the code you will touch.
2. **Two hats.** Either add behavior or refactor, never both in one step. Swap hats consciously.
3. **Small steps.** Change one thing, compile, run tests. Commit when green. If tests fail and the cause is not obvious within a few minutes, revert to the last green state and take a smaller step.
4. **Do not break callers in one move.** For a widely used signature: add the new form, migrate callers, then remove the old one (parallel change).
5. **Integrate often.** Short-lived branches avoid painful merges after large moves.

## When to refactor

| Trigger | Practice |
|---|---|
| Preparatory | "Make the change easy (warning: this may be hard), then make the easy change." Refactor before adding the feature. |
| Comprehension | Rename and extract as you learn what code does, so the next reader benefits. Move understanding from your head into the code. |
| Litter-pickup | Fix small messes you pass. Defer big ones. |
| Rule of three | First time: just do it. Second time: wince, but do it anyway. Third time: refactor. |
| Code review | Refactor while reviewing to spread understanding. |

Do not refactor code that works and never needs to change. Do not add generality for imagined future needs (YAGNI); refactor when the need is real.

## Smells and remedies

| Smell | What the reader sees | Main refactorings |
|---|---|---|
| Mysterious Name | Names that do not say what the thing does. | Change Function Declaration, Rename Variable, Rename Field |
| Duplicated Code | The same code in more than one place. | Extract Function, Slide Statements, Pull Up Method |
| Long Function | Must read many lines to see the point; comments label sections. | Extract Function, Replace Temp with Query, Introduce Parameter Object, Preserve Whole Object, Decompose Conditional, Split Loop |
| Long Parameter List | Many arguments, some derived from others. | Replace Parameter with Query, Preserve Whole Object, Introduce Parameter Object, Remove Flag Argument, Combine Functions into Class |
| Global Data | Anyone can change it from anywhere. | Encapsulate Variable |
| Mutable Data | Changes in one place surprise another. | Encapsulate Variable, Split Variable, Slide Statements, Extract Function, Separate Query from Modifier, Remove Setting Method, Replace Derived Variable with Query, Change Reference to Value |
| Divergent Change | One module changes for many unrelated reasons. | Split Phase, Move Function, Extract Function, Extract Class |
| Shotgun Surgery | One change edits many modules. | Move Function, Move Field, Combine Functions into Class, Combine Functions into Transform, Split Phase, Inline Function, Inline Class |
| Feature Envy | A function uses another module's data more than its own. | Move Function, Extract Function |
| Data Clumps | The same 3+ fields travel together. | Extract Class, Introduce Parameter Object, Preserve Whole Object |
| Primitive Obsession | Money, ranges, phone numbers as raw numbers/strings. | Replace Primitive with Object, Replace Type Code with Subclasses, Replace Conditional with Polymorphism |
| Repeated Switches | The same `switch` on a type in several places. | Replace Conditional with Polymorphism |
| Loops | Loop with accumulators and conditions. | Replace Loop with Pipeline |
| Lazy Element | A class or function that does too little to justify existing. | Inline Function, Inline Class, Collapse Hierarchy |
| Speculative Generality | Hooks and parameters for needs that never came. | Collapse Hierarchy, Inline Function, Inline Class, Change Function Declaration, Remove Dead Code |
| Temporary Field | A field set only in some circumstances. | Extract Class, Move Function, Introduce Special Case |
| Message Chains | `a.getB().getC().getD()` | Hide Delegate, Extract Function, Move Function |
| Middle Man | A class that mostly delegates. | Remove Middle Man, Inline Function, Replace Superclass with Delegate, Replace Subclass with Delegate |
| Insider Trading | Modules trade private data. | Move Function, Move Field, Hide Delegate, Replace Subclass with Delegate, Replace Superclass with Delegate |
| Large Class | Too many fields or responsibilities. | Extract Class, Extract Superclass, Replace Type Code with Subclasses |
| Alternative Classes with Different Interfaces | Two classes do the same job under different names. | Change Function Declaration, Move Function, Extract Superclass |
| Data Class | Fields and accessors only, no behavior. | Encapsulate Record, Remove Setting Method, Move Function, Extract Function, Split Phase |
| Refused Bequest | A subclass ignores what it inherits. | Push Down Method, Push Down Field, Replace Subclass with Delegate, Replace Superclass with Delegate |
| Comments | A comment explains confusing code. | Extract Function, Change Function Declaration, Introduce Assertion |

## Mechanics of the core moves

Each is applied in the smallest possible steps with tests between them.

- **Extract Function:** create a function named for its purpose (what, not how); copy the fragment; pass in variables it reads; return the one variable it changes; replace the fragment with a call; test.
- **Inline Function:** check it is not polymorphic; find the callers; replace each call with the body; test after each; remove the definition.
- **Extract Variable:** ensure the expression has no side effects; declare an immutable variable; replace the expression; test.
- **Change Function Declaration (rename):** if the function is widely used, first extract the body into a new function with the new name, make the old one call it, migrate callers, delete the old one.
- **Move Function:** examine what the function uses in its context; copy to the target; adapt; make the original delegate; test; consider removing the original.
- **Encapsulate Variable:** create getter/setter; replace each direct access; restrict visibility; test.
- **Replace Conditional with Polymorphism:** build the class hierarchy; use a factory; move the conditional logic into each subclass method; remove the branch from the base.
- **Split Phase:** extract the second phase into its own function; introduce an intermediate data structure as its first argument; extract the first phase to return that structure.

## Catalog examples

**Extract Function** (inverse: Inline Function). Name a fragment by its purpose:

```js
function printOwing(invoice) {
  printBanner();
  let outstanding  = calculateOutstanding();

  //print details
  console.log(`name: ${invoice.customer}`);
  console.log(`amount: ${outstanding}`);  
}
```
```js
function printOwing(invoice) {
  printBanner();
  let outstanding  = calculateOutstanding();
  printDetails(outstanding);

  function printDetails(outstanding) {
    console.log(`name: ${invoice.customer}`);
    console.log(`amount: ${outstanding}`);
  }
}
```
The `//print details` comment became a name. A name beats a comment [2].

**Split Phase.** Separate parsing from computing:

```js
const orderData = orderString.split(/\s+/);
const productPrice = priceList[orderData[0].split("-")[1]];
const orderPrice = parseInt(orderData[1]) * productPrice;
```
```js
const orderRecord = parseOrder(order);
const orderPrice = price(orderRecord, priceList);

function parseOrder(aString) {
  const values = aString.split(/\s+/);
  return ({
    productID: values[0].split("-")[1],
    quantity: parseInt(values[1]),
  });
}
function price(order, priceList) {
  return order.quantity * priceList[order.productID];
}
```

**Introduce Parameter Object.** Data that travels together gets a name:

```js
function amountInvoiced(startDate, endDate) {...}
function amountReceived(startDate, endDate) {...}
function amountOverdue(startDate, endDate) {...}
```
```js
function amountInvoiced(aDateRange) {...}
function amountReceived(aDateRange) {...}
function amountOverdue(aDateRange) {...}
```

**Remove Flag Argument** (also "Replace Parameter with Explicit Methods"). The call site `setDimension("height", 10)` hides what is possible; `setHeight(10)` shows it:

```js
function setDimension(name, value) {
  if (name === "height") {
    this._height = value;
    return;
  }
  if (name === "width") {
    this._width = value;
    return;
  }
}
```
```js
function setHeight(value) {this._height = value;}
function setWidth (value) {this._width = value;}
```

**Separate Query from Modifier.** A reader cannot tell a "get" has a side effect:

```js
function getTotalOutstandingAndSendBill() {
  const result = customer.invoices.reduce((total, each) => each.amount + total, 0);
  sendBill();
  return result;
}
```
```js
function totalOutstanding() {
  return customer.invoices.reduce((total, each) => each.amount + total, 0);
}
function sendBill() {
  emailGateway.send(formatBill(customer));
}
```

**Split Loop.** One loop, one job. The cost of a second pass is usually negligible; profile before worrying:

```js
let averageAge = 0;
let totalSalary = 0;
for (const p of people) {
  averageAge += p.age;
  totalSalary += p.salary;
}
averageAge = averageAge / people.length;
```
```js
let totalSalary = 0;
for (const p of people) {
  totalSalary += p.salary;
}

let averageAge = 0;
for (const p of people) {
  averageAge += p.age;
}
averageAge = averageAge / people.length;
```

**Replace Loop with Pipeline:**

```js
const names = [];
for (const i of input) {
  if (i.job === "programmer")
    names.push(i.name);
}
```
```js
const names = input
  .filter(i => i.job === "programmer")
  .map(i => i.name)
;
```

**Replace Magic Literal** (also "Replace Magic Number with Symbolic Constant"):

```js
function potentialEnergy(mass, height) {
  return mass * 9.81 * height;
}
```
```js
const STANDARD_GRAVITY = 9.81;
function potentialEnergy(mass, height) {
  return mass * STANDARD_GRAVITY * height;
}
```

(Replace Nested Conditional with Guard Clauses and Decompose Conditional are in code-level.md, Control flow.)

## Refactoring and performance

Write clear code first. Optimize only measured hot spots, with a profiler. Well-factored code is easier to tune because the hot spot is isolated. Where speed matters, simplify the critical path rather than adding special cases [1][3].

## Catalog index

Change Function Declaration · Change Reference to Value · Change Value to Reference · Collapse Hierarchy · Combine Functions into Class · Combine Functions into Transform · Consolidate Conditional Expression · Decompose Conditional · Encapsulate Collection · Encapsulate Record · Encapsulate Variable · Extract Class · Extract Function · Extract Superclass · Extract Variable · Hide Delegate · Inline Class · Inline Function · Inline Variable · Introduce Assertion · Introduce Parameter Object · Introduce Special Case · Move Field · Move Function · Move Statements into Function · Move Statements to Callers · Parameterize Function · Preserve Whole Object · Pull Up Constructor Body / Field / Method · Push Down Field / Method · Remove Dead Code · Remove Flag Argument · Remove Middle Man · Remove Setting Method · Remove Subclass · Rename Field · Rename Variable · Replace Command with Function · Replace Conditional with Polymorphism · Replace Constructor with Factory Function · Replace Control Flag with Break · Replace Derived Variable with Query · Replace Error Code with Exception · Replace Exception with Precheck · Replace Function with Command · Replace Inline Code with Function Call · Replace Loop with Pipeline · Replace Magic Literal · Replace Nested Conditional with Guard Clauses · Replace Parameter with Query · Replace Primitive with Object · Replace Query with Parameter · Replace Subclass with Delegate · Replace Superclass with Delegate · Replace Temp with Query · Replace Type Code with Subclasses · Return Modified Value · Separate Query from Modifier · Slide Statements · Split Loop · Split Phase · Split Variable · Substitute Algorithm.
