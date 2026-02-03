---
description: Create detailed implementation plans through interactive research and iteration
model: opus
---
STARTER_CHARACTER = 📆

# Create Plan

Your job is to break down the work of a `@.drafts/<feature>.spec.md` into small, known-good increments and document them as a `@.drafts/<feature>.plan.md`. If no spec is provided ask for one. Read the spec FULLY before continuing. 

## Core Principle

### Small, Known-Good Increments
All work must be done in small, known-good increments. Each increment leaves the codebase in a working state where all tests pass.

### Walking Skeleton First
**Always start with a walking skeleton** - a minimal end-to-end implementation that connects all architectural layers with the simplest possible functionality.

A walking skeleton:
- Touches every layer (data → logic → UI)
- Uses hardcoded/stub data initially
- Shows *something* in the UI immediately
- Proves the architecture works before adding complexity
- Prioritize getting results visible in the UI as early as possible. A user should see *something* working after Phase 0, even if it's hardcoded.

## What Makes a "Known-Good Increment"

Each step MUST:
- Leave all tests passing
- Be independently deployable
- Have clear done criteria
- Fit in a single commit
- Be describable in one sentence

**If you can't describe a step in one sentence, break it down further.**

## Step Size Heuristics

**Too big if:**
- Takes more than one session
- Requires multiple commits to complete
- Has multiple "and"s in description
- You're unsure how to test it
- Involves more than 3 files

**Right size if:**
- One clear test case
- One logical change
- Can explain to someone in 30 seconds
- Obvious when done
- Single responsibility


## Test rules

1. **Always use `XyzBuilder` - Never create entities by hand
2. **Types created on demand** - Add data classes/interfaces only when a test requires them
3. **Strict Red-Green-Refactor**:
    - RED: Write failing test first
    - GREEN: Minimal production code to pass
    - REFACTOR: Clean up only after green
4. **Run all tests after every change**

## Document Template

**IMPORTANT: Follow this template EXACTLY. Never paraphrase, summarize, or abbreviate template text. Copy it verbatim and only fill in the bracketed `[placeholders]`.**

### .plan.md

```markdown
# Plan: [Feature Name]

# Context

## Goal
[Goal of the feature, key acceptance criteria]

## Files to Create/Modify

| File | Created When |
|------|--------------|
| `some/where.spec.ts` | Step 1.1 |
| `some/where.ts` | Step 1.1 |

## Design Notes

### [Subchapter title]
[Any valid design notes]

### [Subchapter title]
[Any valid patterns to follow]

### [Subchapter title]
[Constraints to work within]

## Example Code

### Subchapter
[Any valid code]

## Blockers

None | [Description of blocker]

---------------

# Next Action

[Specific next thing to do]

# Phases and Steps

## Phase 1: [Title]

### Step 1.1: [Title]
- [x] RED: Write test. [should do X]
   - Input: `[input values]`
   - Output: `[output value]`
   - Test will [be red/not compile/etc.]: [Reason]
   - Used/Created [Builder]
- [x] GREEN: Create `[file/class]` with method `[xyz]`
- [x] Run all lint and tests
- [x] REFACTOR: Launch `@.claude/agents/code-review.md` agent, add what needs fixing as substeps to the plan and then fix those problems, if there are any
   - [x] Remove duplicate code
   - [x] Increase synergy of names
- [x] Run all tests

### Step 1.2: [Title]
- [ ] RED: Write test. `class/file.method` should X
   - Input: `[input values]`
   - Output: `[output value]`
   - Test will [be red/not compile/etc.]: [Reason]
- [ ] GREEN: Create `[file/class]` with method `[xyz]`
- [ ] Run all lint and tests
- [ ] REFACTOR: Launch `@.claude/agents/code-review.md` agent, add what needs fixing to the plan and then fix those problems, if there are any
- [ ] Run all lint and tests
```