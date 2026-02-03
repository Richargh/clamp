---
description: Implement technical plans from thoughts/shared/plans with verification
---

STARTER_CHARACTER = 🛠️

# Implement Plan

Your job is to implement a plan in `@.drafts/<feature>.plan.md`. If no plan was provided, ask for one.

## Getting Started

When given a plan:
* Read the plan completely and check for any existing checkmarks (* [x])
* **Read files fully** — never use limit/offset parameters, you need complete context
* Think deeply about how the pieces fit together
* Start implementing if you understand what needs to be done
* If not ask questions and add the missing information to the plan.
* You're implementing a solution, not just checking boxes. Keep the end goal in mind and maintain forward momentum.

## Clean Start

Before starting the implementation check:

* Check if there are untracked or modified files. If so add an additional STARTER_CHARACTER: ⏯️
* Check you are in valid state: do all lints, tests etc. run? If not suggest how to fix them.

## Resuming Work

If the plan has existing checkmarks:
* Trust that completed work is done
* Pick up from the first unchecked item
* Verify previous work only if something seems off

## Implementation Philosophy

If you encounter a mismatch between what the plan says and the code:
* STOP and think deeply about why the plan can't be followed
* Present the issue clearly:
  ```
  Issue in Step [M.N]:
  Expected: [what the plan says]
  Found: [actual situation]
  Why this matters: [explanation]

  How should I proceed?
  ```

## Verification Approach

After implementing a step or phase:
* Fix any issues before proceeding
* Update your progress in both the plan
* Check off completed items in the plan file itself
* **Pause for human verification**: After completing all automated verification for a phase, pause and inform the human that the phase is ready for manual testing. Use this format:
  ```
  Phase [N] Complete - Ready for Manual Verification

  Automated verification passed:
  - [List automated checks that passed]

  Please perform the manual verification steps listed in the plan:
  - [List manual verification items from the plan]

  Let me know when manual testing is complete so I can proceed to Phase [N+1].
  ```

If instructed to execute multiple phases consecutively, skip the pause until the last phase. Otherwise, assume you are just doing one phase.

Do not check off items in the manual testing steps until confirmed by the user.

## Refactor Philosophy

Add all the refactoring results that you got to the plan like this:

```markdown
* [ ] REFACTOR: Launch `@.claude/agents/code-review.md` agent and fix problems if there are any
  * [ ] Review problem 1
  * [ ] Review problem 2
```

If there are no problems then change the text to:

```markdown
* [x] REFACTOR: Launched code-review agent. It found no problems
```

Afterward proceed to fix the problems or continue with the next step. 

## If You Get Stuck

When something isn't working as expected:
* First, make sure you've read and understood all the relevant code
* Consider if the codebase has evolved since the plan was written
* Present the mismatch clearly and ask for guidance

Use sub-tasks sparingly - mainly for targeted debugging or exploring unfamiliar territory.
