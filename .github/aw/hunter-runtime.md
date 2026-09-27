---
network:
  allowed:
    - defaults
    - go
---

# Tasks repository runtime guidance

This repository-owned import supplements the generated Code Hunters components.
Keep it in every hunter wrapper when refreshing upstream templates. Recompile the
wrappers with gh-aw v0.86.2; compiler and runtime pins must move together.

## Complete within the execution budget

- The agent step has a 20-minute hard timeout. Record the start time immediately
  and check elapsed time before each reproduction, validation, or delivery step.
  Stop investigating by minute 10, stop editing by minute 15, and submit the final
  safe output by minute 18, leaving time for tool completion.
- Bound each shell command with a timeout shorter than the remaining phase budget.
  Stop or cancel unfinished commands at the deadline. Do not launch another
  candidate unless its implementation, verification, and delivery fit the budget.
  The five-PR limit is a ceiling, not a target.
- Before reproducing a candidate, inspect open PR metadata and changed paths, then
  read relevant diffs for overlapping symbols or behavior. Skip overlapping work
  before expensive investigation. Keep the existing refresh gates before editing
  and delivery.
- Prefer deterministic synchronization or direct control-flow evidence over
  repeated timing stress. Bound any stress experiment and abandon it when it
  cannot establish the required evidence within the investigation budget.

## Validate and report honestly

- Use normal Go module downloads and checksum verification. Never set GOSUMDB=off
  or change dependency security settings to bypass a failed download. Report the
  exact dependency or network failure if the configured Go access is insufficient.
- Reserve time for repository-native validation and the final safe-output call.
  Never describe an unrun, interrupted, or failing check as passing, and never
  submit an unvalidated correction merely to beat the deadline.
- Use the configured create_pull_request safe output for validated changes; do not
  replace it with gh pr create or git push. If the run cannot finish, report status
  `incomplete`, the evidence gathered, exact blockers, and remaining validation.
  Use report_incomplete only if it is actually available. Otherwise use the
  available noop tool with an explicit `incomplete` message. This fallback takes
  precedence over the generic component's successful-investigation-only noop rule
  when report_incomplete is unavailable. It means no change
  was delivered, not that the scan or validation succeeded. Do not spend the
  remaining budget searching for an unavailable reporting tool.
