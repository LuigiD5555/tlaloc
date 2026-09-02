# P0 authoring scaffold — Refactoring_improving_the_design_of_existing_code.pdf

Hand-write the remaining questions into `end-to-end.jsonl` (start from
`end-to-end.draft.jsonl`, which holds the auto-generated ones). Each question
must be answerable **only** from the evidence quoted below. Schema:
`datasets/SCHEMA.md`. Then: `validate --stage end_to_end` → `freeze --scope stage`.

Target: 30 questions over these 10 pages — 6 each of locate / entity / factual / numeric / synthesis.

## Page 107 — "Remove Assignments to Parameters"

- address: `ohf://fold-bench/pages/000107`  ·  page cid: `c101c3ba5011abf50ccd02edf95cd0da8285d22738c2e47ff4f6226002f98e00`
- rendered page: `end-to-end/scaffold-images/p107.png`
- candidate noun phrases: "Remove Assignments"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "Extract Class" or "Remove Assignments to Paramet… → "Remove Assignments to Parameters"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
result += primaryVel * secondaryTime + 0.5 *
secondaryAcc
* secondaryTime * secondaryTime;
}
return result;
}
I'm sure you can think of a lot more refactoring to be done here. Enjoy it. (I'm sure it's better than
eating the haggis—do you know what they put in those things?)
Remove Assignments to Parameters
The code assigns to a parameter.
Use a temporary variable instead.
```
</details>

## Page 119 — "Move Field"

- address: `ohf://fold-bench/pages/000119`  ·  page cid: `4e6be74fc549159468eafd87424e6d56708926d9b4fc7fca51a272434acb3cbb`
- rendered page: `end-to-end/scaffold-images/p119.png`
- auto-generated for this page:
    - [synthesis] The page shows two refactoring sections. Which heading appears first on the page: "Move Field" or "Extract Class"? Answe… → "Move Field"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
class AccountType...
double overdraftCharge(Account account) {
if (isPremium()) {
double result = 10;
if (account.getDaysOverdrawn() > 7)
result += (account.getDaysOverdrawn() - 7) * 0.85;
return result;
}
else return account.getDaysOverdrawn() * 1.75;
}
I also pass in the source object if I need several features of the class, although if there are too
```
</details>

## Page 122 — "Extract Class"

- address: `ohf://fold-bench/pages/000122`  ·  page cid: `db6b6de7d3656e65bfde93a0f56542dce26f3f870b9eb34f44489389830853a6`
- rendered page: `end-to-end/scaffold-images/p122.png`
- candidate noun phrases: "Move Method"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "Consolidate Conditional Expression" or "Extract … → "Extract Class"
    - [synthesis] The page shows two refactoring sections. Which heading appears first on the page: "Move Method" or "Extract Class"? Answ… → "Move Method"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
I can redirect the clients of the accessors to use the new object later if I want. Using self-
encapsulation allows me to take a smaller step. This is useful if I'm doing a lot of things with the
class. In particular, it simplifies use
Move Method
to move methods to the target class. If they
refer to the accessor, such references don't need to change.
Extract Class
You have one class doing work that should be done by two.
Create a new class and move the relevant fields and methods from the old class into the new
class.
Motivation
```
</details>

## Page 133 — "Introduce Local Extension"

- address: `ohf://fold-bench/pages/000133`  ·  page cid: `7006b7facb04cfef86eefdc451549a6970ed84d173549f2fba5892cf396ed4a7`
- rendered page: `end-to-end/scaffold-images/p133.png`
- candidate noun phrases: "Introduce Local Extension"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "Introduce Local Extension" or "Encapsulate Colle… → "Introduce Local Extension"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
Date newStart = nextDay(previousEnd);
private static Date nextDay(Date arg) {
// foreign method, should be on date
return new Date (arg.getYear(),arg.getMonth(), arg.getDate() +
1);
}
Introduce Local Extension
A server class you are using needs several additional methods, but you can't modify the class.
Create a new class that contains these extra methods. Make this extension class a subclass or a
wrapper of the original.
Motivation
```
</details>

## Page 166 — "Replace Magic Number with Symbolic Constant"

- address: `ohf://fold-bench/pages/000166`  ·  page cid: `3617995141e4852612e572bba0074886ef19be9d85a114b0a1af6051d642e0cc`
- rendered page: `end-to-end/scaffold-images/p166.png`
- candidate noun phrases: "Replace Magic Number"
- auto-generated for this page:
    - [synthesis] The page shows two refactoring sections. Which heading appears first on the page: "Replace Magic Number with Symbolic Co… → "Replace Magic Number with Symbolic Constant"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
change. In practice, this process usually is pretty rapid. If it were complicated, I would give up on
this refactoring.
Once I've eliminated the readers of the field, I can work on the writers of the field. This is as
simple as removing any assignments to the field and then removing the field. Because nobody is
reading it any more, that shouldn't matter.
Replace Magic Number with Symbolic Constant
You have a literal number with a particular meaning.
Create a constant, name it after the meaning, and replace the number with it.
double potentialEnergy(double mass, double height) {
return mass * 9.81 * height;
}
```
</details>

## Page 168 — "Encapsulate Collection"

- address: `ohf://fold-bench/pages/000168`  ·  page cid: `74eb8086ed946555eb301fc14446df67496cc23b7b2b1b537cd652b24018f58c`
- rendered page: `end-to-end/scaffold-images/p168.png`
- **Motivation** (verbatim): Often a class contains a collection of instances. This collection might be an array, list, set, or vector. Such cases often have the usual getter and setter for the collection. However, collections should use a protocol slightly different from that for other kinds of data. The getter should not return the collection object itself, because that allows clients to manipulate the contents of the collection without the owning class's knowing what is going on. It also reveals too much to clients about…
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "Move Field" or "Encapsulate Collection". Which o… → "Encapsulate Collection"
    - [entity] This page describes one refactoring. Its motivation section begins: "Often a class contains a collection of instances.".… → "Encapsulate Collection"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
Encapsulate Collection
A method returns a collection.
Make it return a read-only view and provide add/remove methods.
Motivation
Often a class contains a collection of instances. This collection might be an array, list, set, or
vector. Such cases often have the usual getter and setter for the collection.
However, collections should use a protocol slightly different from that for other kinds of data. The
getter should not return the collection object itself, because that allows clients to manipulate the
contents of the collection without the owning class's knowing what is going on. It also reveals too
much to clients about the object's internal data structures. A getter for a multivalued attribute
should return something that prevents manipulation of the collection and hides unnecessary
```
</details>

## Page 194 — "Consolidate Conditional Expression"

- address: `ohf://fold-bench/pages/000194`  ·  page cid: `54e7929faeac75c362b00c46eefcdafdc2157c9a55373101d212d213d77af323`
- rendered page: `end-to-end/scaffold-images/p194.png`
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "Remove Assignments to Parameters" or "Consolidat… → "Consolidate Conditional Expression"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
Consolidate Conditional Expression
You have a sequence of conditional tests with the same result.
Combine them into a single conditional expression and extract it.
double disabilityAmount() {
if (_seniority < 2) return 0;
if (_monthsDisabled > 12) return 0;
if (_isPartTime) return 0;
// compute the disability amount
double disabilityAmount() {
if (isNotEligableForDisability()) return 0;
// compute the disability amount
```
</details>

## Page 196 — "Consolidate Duplicate Conditional Fragments"

- address: `ohf://fold-bench/pages/000196`  ·  page cid: `0d96d879091791243435a5a8144e0d74b9e4e54d5f904d45ef1d9fc45b8b9f49`
- rendered page: `end-to-end/scaffold-images/p196.png`
- candidate noun phrases: "Consolidate Duplicate Conditional Fragments"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
If the routine I'm looking at tests only the condition and returns a value, I can turn the routine into
a single return statement using the ternary operator. So
if (onVacation() && lengthOfService() > 10) return 1;
else return 0.5;
becomes
return (onVacation() && lengthOfService() > 10) ? 1 : 0.5;
Consolidate Duplicate Conditional Fragments
The same fragment of code is in all branches of a conditional expression.
Move it outside of the expression.
if (isSpecialDeal()) {
total = price * 0.95;
```
</details>

## Page 225 — "Separate Query from Modifier"

- address: `ohf://fold-bench/pages/000225`  ·  page cid: `67cec19d32fbd29d170e2f760e2b5bb7d8eb854608eef2cda85382f4528839e4`
- rendered page: `end-to-end/scaffold-images/p225.png`
- candidate noun phrases: "Separate Query"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "Separate Query from Modifier" or "Encapsulate Do… → "Separate Query from Modifier"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
?rarr;
If the old method is part of the interface and you cannot remove it,
leave it in place and mark it as deprecated.
•
Compile and test.
Because I'm pretty comfortable with adding and removing parameters, I often do a batch in one
go.
Separate Query from Modifier
You have a method that returns a value but also changes the state of an object.
Create two methods, one for the query and one for the modification.
Motivation
```
</details>

## Page 249 — "Encapsulate Downcast"

- address: `ohf://fold-bench/pages/000249`  ·  page cid: `0f94ef2349d23c98e6510fb62d1fb9dbe96cdb82b106e890c9519a7f158db3ed`
- rendered page: `end-to-end/scaffold-images/p249.png`

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
I can use a different approach to hide subclasses with explicit methods. This is useful if you have
just a few subclasses that don't change. So I may have an abstract Person class with subclasses
for Male and Female. I begin by defining a factory method for each subclass on the superclass:
class Person...
static Person createMale(){
return new Male();
}
static Person createFemale() {
return new Female();
}
I can then replace calls of the form
```
</details>

