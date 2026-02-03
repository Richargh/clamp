---
description: Create detailed implementation specifications through interactive research and iteration
model: opus
---
STARTER_CHARACTER = 📐

# Create Spec

Your job is to create a specification of a feature in the form `@.drafts/<feature>.spec.md`.  

Ask questions to make sure you understand the intent of the feature to-be-specified, then research the code base so you can fill out the template.

## After filling out the template

Present the recommended approach and ask for confirmation. Incorporate any feedback and update the spec. Once done mark the `## Selected Approach` in the document, describe why it was taken (if this is not clear ask follow-up questions) then remove recommendations that were not taken. Suggest creating an ADR if the selected approach is a significant decision.

## Document Template

### .spec.md

```markdown
# Spec: [Feature Name]

**Created**: [Date]
**Status**: In Progress | Complete

## Goal

[One sentence describing the outcome]

## Out of Scope

* Out of Scope 1
* Out of Scope 2

## Acceptance Criteria

*[ ] (1) Criterion 1
*[ ] (2) Criterion 2

## Quality Constrains

* [ ] Quality constraint 1
* [ ] Quality constraint 2

## Edge Cases

* [ ] Edge Case 1
* [ ] Edge Case 2

## Risks

* [ ] Risk 1
* [ ] Risk 2


## Current State Analysis

[What exists now, what's missing, key constraints discovered]

## Key Discoveries:
- [Important finding with file:line reference]
- [Pattern to follow]
- [Constraint to work within]

### Relevant files for the implementation

| File | Relevant when |
|------|--------------|
| `some/where.spec.ts` | [Relevant when X] |
| `some/where.ts` | [Relevant when Y] |

## Approach Options

### Option A: [Title]

- [Description]
- Pro 1
- Pro 2
- Con 1
- Con 2

## Recommended Approach

- Option X [Title]
- Reason 1
- Technical Detail 1

---

*Changes to this spec require explicit approval.*
```