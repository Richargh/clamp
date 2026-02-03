---
name: risk-aware-commit
description: This skill describes how to make a make a commit with the risk-aware-commit notation.
---

# Risk-Aware Commit Notation
STARTER_CHARACTER = 🔒

When commiting, use a commit notation that allows developers to convey how risky the commit is and what the intention is.

## Overview

The Risk-Aware Commit Notation is a system that prefixes commit messages with concise codes to convey:

1. **Risk Level**: The degree of confidence in the change's safety.
2. **Intention**: The purpose or nature of the change.

This notation aids in rapid assessment of commits, facilitating safer and more efficient code reviews and integrations.

## Risk Level

Each commit message starts with a symbol indicating its risk level:

- **.** - Safe, for example a deterministic refactoring by an IDE
- **^** - Validated, for example by an automatic test
- **!** - Risky, for example a manual change that could not be fully validated
- **@** - (Probably) Broken aka no risk-attestation, for example when the developer cannot see the results of the work without checking in

## Intention

Following the risk symbol, a letter denotes the intention of the commit:

- **R:** or **r:** - Refactoring (structural changes, no behavior change)
- **F:** or **f:** - Feature (new behavioral changes)
- **B:** or **b:** - Bug fix (fixing incorrect behavior)
- **T:** or **t:** - Test-only changes
- **D:** or **d:** - Documentation
- **C:** or **c:** - Comments
- **E:** or **e:** - Environment/configuration

The casing signals roughly to "pay more attention to the ones in uppercase". It should be used to show how visible the change is. An implemented feature that is not visible to the user would be a lower-case f. A refactoring for an important internal Api would be an upper-case R.

## Commit Message Format

The general format for commit messages is:

```
<Risk Symbol> <Intention Letter>: <Title>

[Optional Body: Explain the 'why' and 'what' in more detail.]
```

## Examples

- `. R: extract validation logic to separate function with automated refactoring`
- `^ F: implement user-visible login validation`
- `! B: fix null pointer in data processing`
- `^ t: add test for edge case in sorting`
- `. d: add documentation, always safe, unless user-facing`

## Key Principles

ALWAYS add the 2 character prefix to commit messages.
IMPORTANT: Never mix `R` commits with `F` or `B` commits. This enforces the separation of structural and behavioral changes.

