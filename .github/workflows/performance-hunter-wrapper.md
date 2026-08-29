---
on:
  schedule: weekly on saturday
  workflow_dispatch:
  steps:
    - name: Select rotating hunter cohort
      id: rotation
      run: |
        epoch_days="$(($(date -u +%s) / 86400))"
        week="$(((epoch_days + 3) / 7))"
        if [ "$((10#$week % 4))" -eq 2 ]; then
          echo "active=true" >> "$GITHUB_OUTPUT"
        else
          echo "active=false" >> "$GITHUB_OUTPUT"
        fi
  skip-if-match:
    query: 'is:pr is:open author:app/github-actions in:body "code-hunters-origin"'
    # Default total-open limit. Change only through explicit operator configuration.
    max: 10

if: github.event_name == 'workflow_dispatch' || needs.pre_activation.outputs.rotation_active == 'true'

jobs:
  pre-activation:
    outputs:
      rotation_active: ${{ steps.rotation.outputs.active }}

concurrency:
  group: code-hunters-performance-hunter
  cancel-in-progress: false

imports:
  - uses: ./performance-hunter.md
    with:
      allowed-files: ["*.go", "go.*"]
      protected-files: fallback-to-issue
      # Default per-run limit. Change only through explicit operator configuration.
      max-pull-requests-per-run: 5
---

<!-- Generated from hunter.json. Do not edit directly. -->

# Run Performance Hunter

Run one bounded Performance Hunter maintenance pass.
