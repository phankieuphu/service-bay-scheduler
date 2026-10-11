---
emoji: 🔍
description: On-demand AI code review — comment "/review" on a pull request to re-run the PR code review.
intent: Anyone with write access can get a fresh AI code review of a PR's current state, e.g. after pushing fixes, by commenting /review.
engine: claude
on:
  slash_command:
    name: review
    events: [pull_request_comment, pull_request_review_comment]
permissions:
  contents: read
  pull-requests: read
  issues: read
imports:
  - shared/pr-code-review.md
---

This run was triggered by a `/review` comment.
