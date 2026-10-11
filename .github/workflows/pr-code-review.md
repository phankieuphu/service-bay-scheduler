---
emoji: 🔍
description: AI code review for pull requests — finds correctness, concurrency, data-consistency, and security bugs in the diff and posts them as inline review comments.
intent: Reviewers see the real defects in a PR (bugs, races, broken transactional-outbox or Kafka contracts, security holes) as precise inline comments before they read the diff, with no noise when the change is clean.
on:
  pull_request:
    types: [opened, ready_for_review, reopened]
  skip-bots: [renovate, dependabot]
if: ${{ !github.event.pull_request.draft }}
permissions:
  contents: read
  pull-requests: read
  issues: read
imports:
  - shared/pr-code-review.md
---

This run was triggered automatically by a pull request event, not by a `/review` comment.
