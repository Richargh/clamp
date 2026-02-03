Always start replies with STARTER_CHARACTER + space (default: 🗜️). Stack emojis when requested, don't replace.

## AI Guides

ALWAYS generate the following `.md` files ALWAYS inside `@.drafts`.

In particular keep track of the three (sometimes four) documents when doing feature development:

| Document               | Purpose                                    | Updates                 |
|------------------------|--------------------------------------------|-------------------------|
| <feature>.research.md  | (Optional) Additional research             | As discoveries occur    |
| <feature>.spec.md      | What we want to build                      | Only with user approval |
| <feature>.plan.md      | How we'll get there from the current state | Constantly              |
| <feature>.learnings.md | What we discovered                         | As discoveries occur    |


In addition, store key architecture decisions as Architecture Decision Records (ADRs). Location @docs/adrs. ALWAYS suggest to write an ADR when we make an architecturally significant decision.

NEVER generate additional `.md` files beyond the ones mentioned above.

## CORE DEVELOPMENT PRINCIPLES

* Always follow the TDD cycle: Red → Green → Refactor
* Write the simplest failing test first
* Implement the minimum code needed to make tests pass
* Refactor only after tests are passing
* Maintain high code quality throughout development
* Employ strong types: create semantic wrappers for primitive types that describe what should be done.
