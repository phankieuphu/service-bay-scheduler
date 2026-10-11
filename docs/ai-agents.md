# AI Agents

> **Status: removed (2026-10-11).** The AI workflows below were deleted because every run failed. The Copilot engine rejected `COPILOT_GITHUB_TOKEN` with a 401, so the agent never started. The code is still in git history; see [Restoring](#restoring).

From #117 to #123 the repository ran four [GitHub Agentic Workflows](https://github.github.com/gh-aw/) (gh-aw v0.89.21) on pull requests and issues:

| Workflow | What it did |
|---|---|
| PR Description Review | Scored each PR description on summary, completeness, clarity, linked issues, testing notes, and risk; posted one comment and a `quality: good / needs-work / incomplete` label. |
| Issue & PR Labeler | Added type (`bug`, `enhancement`, …), area (`area: vehicle-service`, `area: kafka`, …), `risk: high` (PRs), and `quality: *` (issues) labels. |
| PR Code Review | Reviewed the diff for correctness, transactional-outbox, concurrency, Kafka-contract, security, and `-test.go` naming defects; posted up to 10 inline comments and one review. |
| PR Code Review (`/review`) | The same review, run by commenting `/review` on a PR. |

In each workflow the agent had read-only access. All of its writes went through gh-aw safe outputs, which were limited to fixed label lists and capped numbers of comments.

---

## Restoring

The last commit that has the workflows is `cba9cd274a04c44aeeaedd65f0f7a4cb4c2778a3` (the merge of #123).

```bash
git checkout cba9cd274a04c44aeeaedd65f0f7a4cb4c2778a3 -- .github .gitattributes
git show cba9cd274a04c44aeeaedd65f0f7a4cb4c2778a3:docs/ai-agents.md > docs/ai-agents.md   # full docs: labels, rubric, setup, troubleshooting
gh extension install github/gh-aw
gh aw compile --strict --validate --actionlint
```

Before restoring, set up a model credential that works:

- **Copilot (default engine).** Needs a fine-grained PAT owned by your **user account**, with **Account permissions → Copilot Requests: Read**, from an account with an active paid Copilot plan. Test it first with `COPILOT_GITHUB_TOKEN=<pat> copilot -p "hi"`. If that fails, the workflows will fail too.
- **Claude.** Add `engine: claude` to each of the four main workflows and set `ANTHROPIC_API_KEY`. The key must be created **inside a Console workspace** (otherwise every call fails with *"not scoped to a workspace"*), and the account needs prepaid API credits, separate from a claude.ai subscription.

---

## Lessons from building them

These apply to any future gh-aw workflow in this repo:

- **The `.lock.yml` is what runs.** Recompile after every change to a workflow `.md` file or shared import, and commit both files.
- **Use only plain `${{ }}` expressions in prompt bodies.** `${{ github.event.pull_request.number }}` and `${{ a || b }}` are fine. A conditional such as `${{ x == 'y' && 'a' || 'b' }}` compiles, but every run then fails at activation with *"unauthorized GitHub Actions expressions"*.
- **`pull_request` and `slash_command` triggers can't share a workflow.** Plain mode refuses to compile. In `strategy: centralized` mode the command check also gates PR events, so the workflow is silently skipped on every new PR. Use two workflows that share an import.
- **List imported permissions in the main workflow too.** Permissions in a `shared/*.md` import must also be granted in each workflow that imports it.
- **New secrets need `--approve`.** Strict mode flags an added secret or action. Check where the lock file uses it before approving.
- **Keep `ghes: false`** in `.github/workflows/aw.json` on github.com. `true` makes the compiler use v3 of the artifact actions, which github.com has retired.
- **Prove the credential before rollout.** A green compile says nothing about whether the model will accept the token. Run the local test above first.
