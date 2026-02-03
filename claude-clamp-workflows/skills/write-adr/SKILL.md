---
name: write-adr
description: This skill describes how to document an architecture decision record (ADR).
---
The file name of an Architecture Decision Records (ADRs), is `ADR-###-<name-of-problem-in-kebab-case>.md` where `###` is the number of the ADR, starting at `001`. Location @docs/adrs.


Structure of an ADR document with YAML frontmatter followed by content:
```markdown
---
status: [Proposed, Accepted, Superseeded]
version: [version starting at 0.1.0 for proposed and 1.0.0 for accepted]
date: [Current date in YYYY-MM-DD format]
last_updated: [Current date in YYYY-MM-DD format]
---

# ADR-###: [The problem that this ADR solves]

## 1) Context & Problem Statement
Short description

## 2) Options Considered
A bullet-point list of what options were possible. Mention at least 2 but at most 5.

Followed by a **Quick comparison** table that lists pros, cons, risk and cost/complexity for all options.

## 3)  Decision
Describe the

* **Chosen option:** [which option was picked]
* **Because:** [the reason it was picked]

## 4) Consequences
What the positive and negative consequences of the decision are in bullet points.

## 5) Change Log
A bullet-point list of versions and what was changed. Typically starts with:

* v0.1.0 — Initial draft
```