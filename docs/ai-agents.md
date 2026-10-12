# AI Agents

The repository runs two AI agent workflows on GitHub Actions. One reviews pull request descriptions; the other labels issues and PRs. Both are built with [GitHub Agentic Workflows](https://github.github.com/gh-aw/) (gh-aw, v0.89.21) and use the default **Copilot** engine, billed to the token owner's GitHub Copilot plan.

Neither can push code, merge, approve, or close anything. Each one can only post the comments and labels listed below.

---

## 1. What runs, and when

| Workflow | Source | Trigger | Writes |
|---|---|---|---|
| PR Description Review | [`pr-description-review.md`](../.github/workflows/pr-description-review.md) | Non-draft PR opened, edited, reopened, or marked ready | One review comment, which replaces its previous one, and one `quality: *` label |
| Issue & PR Labeler | [`issue-pr-labeler.md`](../.github/workflows/issue-pr-labeler.md) | Issue opened, edited, or reopened; non-draft PR opened, reopened, or marked ready | Type, area, and risk labels; `quality: *` on issues |

Both skip PRs and issues opened by Renovate and Dependabot.

An AI code review workflow (with a `/review` command) was added in #120 and removed in #131 because its runs kept failing. To restore it: `git checkout cba9cd2 -- .github/workflows/pr-code-review.md .github/workflows/pr-code-review-command.md .github/workflows/shared/pr-code-review.md`, then `gh aw compile`. PRs from forks are blocked by default.

---

## 2. PR Description Review

This workflow reviews the PR **description** (title and body), not the code. It scores six criteria as ✅ met, ⚠️ partial, or ❌ missing. It reads the list of changed files to check that the description covers what the PR touches.

| # | Criterion | Looks for |
|---|---|---|
| 1 | Summary & motivation | what changed and why |
| 2 | Completeness vs. scope | every significant area in the file list is mentioned |
| 3 | Clarity | readable without prior context; structured if non-trivial |
| 4 | Linked issues | `Closes #n` etc., or a reason there is none |
| 5 | Testing notes | how it was verified (`go test`, manual steps, screenshots) |
| 6 | Risk & rollout | breaking changes, migrations, Kafka contract or config changes, rollback |

Expectations scale with the size of the PR: a 5-line docs fix needs far less than a cross-service change.

| Label | Rule |
|---|---|
| `quality: good` | no ❌ and at most one ⚠️ |
| `quality: needs-work` | more than one ⚠️, or exactly one ❌ on criteria 4–6 |
| `quality: incomplete` | ❌ on criterion 1 or 2, or two or more ❌ |

If you edit the description, the workflow runs again and replaces both its comment and its label.

---

## 3. Issue & PR Labeler

It only **adds** labels. It never removes a label someone applied by hand. The one exception is that it updates its own `quality: *` label when an issue is edited.

**Type** (exactly one): `bug`, `enhancement`, `documentation`, `question`, `chore`

**Area** (zero or more):

| Label | Paths / signals |
|---|---|
| `area: customer-service` | `customer-service/**` |
| `area: vehicle-service` | `vehicle-service/**` |
| `area: identity-service` | `identity-service/**`, auth, JWT, refresh tokens |
| `area: frontend` | `frontend/**` |
| `area: kafka` | `*/internal/adapters/kafka/**`, `*/internal/constants/topics.go`, outbox, event envelope, Flink |
| `area: infra` | `docker-compose.yml`, `Dockerfile`, `Makefile`, `k8s/**`, `postgres/**`, Railway/Vercel config |
| `area: observability` | `prometheus/**`, `grafana/**`, `loki/**`, `alloy/**`, `loadtest/**`, `*/internal/adapters/metrics/**` |
| `area: ci` | `.github/**` |

**Risk** (PRs only): `risk: high` when a PR changes database models or migrations, Kafka payloads or topics, identity auth or token handling, or `k8s/**`, or when it spans two or more services.

**Quality** (issues only): uses the same `quality: *` scale as §2. Bugs need reproduction steps or error output, plus expected vs. actual behavior. Features need acceptance criteria or a concrete use case.

The labeler runs for **anyone** who opens an issue (`roles: all`), so outside reports get triaged too. The description review only runs for people with write access.

---

## 4. How it works

```
.github/workflows/<name>.md      ← you edit this (YAML frontmatter + prompt)
        │  gh aw compile
        ▼
.github/workflows/<name>.lock.yml ← generated GitHub Actions workflow; never edit by hand
```

Every run goes through these jobs:

1. **pre_activation:** checks the actor's role and skips bots.
2. **activation:** builds the prompt. The `.md` body is loaded when the run starts, not copied into the lock file, so expression errors in it only show up here (§5). Recompile after any edit anyway; this job also checks that the lock file matches its source.
3. **agent:** runs the GitHub Copilot CLI in a sandbox behind a network firewall, with **read-only** access to `contents`, `issues`, and `pull-requests`. It reads GitHub data with `gh`. The model credential is kept out of the sandbox; only gh-aw's API proxy uses it.
4. **detection:** a second model pass that screens the agent's proposed output for prompt injection, secret leaks, and malicious content.
5. **safe_outputs:** the only job with write permission. It applies the comments and labels the agent asked for, enforced against the allowlists and limits in the frontmatter (`safe-outputs:`).

Text in issues, PRs, and comments is treated as data. Each prompt tells the agent to ignore any instructions inside it. Even if an injection got through, the agent could only apply allowed labels or post one comment.

---

## 5. Changing a workflow

```bash
gh extension install github/gh-aw          # once
# edit .github/workflows/<name>.md
gh aw compile --strict --validate --actionlint
git add .github/workflows/                 # commit the .md AND the regenerated .lock.yml
```

Rules learned the hard way:

- **Recompile after every change** and commit the regenerated `.lock.yml` files. The lock file is what Actions actually runs.
- **Use only plain expressions in the prompt body.** `${{ github.event.pull_request.number }}` and `${{ a || b }}` are allowed. A conditional such as `${{ github.event_name == 'issues' && 'issue' || 'pull request' }}` compiles, but every run then fails in the activation job with *"unauthorized GitHub Actions expressions"*.
- **New secrets need `--approve`.** Strict mode warns ("safe update mode detected unapproved changes") when a compile adds a secret or action. Check where the secret is used in the `.lock.yml`, then recompile with `--approve` and note it in the PR.
- **Keep `ghes: false`** in [`aw.json`](../.github/workflows/aw.json). `true` makes the compiler use v3 of the artifact actions, which github.com has retired.
- To use a different model, set `model:` in the frontmatter. To use a different engine, change `engine:` in both workflows.

---

## 6. Setup

| Requirement | How |
|---|---|
| `COPILOT_GITHUB_TOKEN` repository secret | A **fine-grained** PAT ([create one, prefilled](https://github.com/settings/personal-access-tokens/new?name=COPILOT_GITHUB_TOKEN&user_copilot_requests=read)): resource owner is **your user account**, and **Account permissions → Copilot Requests: Read**. The owner needs an active paid Copilot plan (Pro or Pro+; see [settings/copilot](https://github.com/settings/copilot)). Copy it, then run `pbpaste \| gh aw secrets set COPILOT_GITHUB_TOKEN`, or set it under Settings → Secrets and variables → Actions. Don't use `--value`, which saves the token in your shell history. |
| Actions enabled | Settings → Actions → General: allow GitHub Actions, including `github/gh-aw-actions/*` and the pinned `actions/*`. |
| Labels (optional) | GitHub creates missing labels in grey on first use. Create them up front with `gh label create "quality: good" --color 0e8a16` and so on to choose colours. |

The token is used only to call the model. Comments and labels are written through the default `GITHUB_TOKEN`. OAuth tokens (`gho_…`, e.g. from `gh auth token`) are rejected at startup.

To use Claude instead, add `engine: claude` to both workflows, set an `ANTHROPIC_API_KEY` secret, and recompile with `--approve`. The key must be created **inside a Console workspace**: a key without a workspace fails with *"not scoped to a workspace"*. API use needs prepaid Anthropic credits, separate from a claude.ai subscription.

---

## 7. Troubleshooting

Open the run under **Actions**, or run `gh run view <id> --log-failed`.

| Symptom | Cause | Fix |
|---|---|---|
| `activation` fails at *Interpolate variables and render templates* with "unauthorized GitHub Actions expressions" | A compound or conditional `${{ }}` in the prompt body | Use only plain or `a \|\| b` expressions (§5) |
| `activation` fails at *Validate COPILOT_GITHUB_TOKEN secret* | Secret missing or empty | Set it (§6) |
| `agent` fails at *Execute GitHub Copilot CLI* with "models fetch returned 401" and "model alias resolution failed" | The token was rejected: it isn't fine-grained, lacks **Copilot Requests: Read**, is owned by an org, has expired, or its owner has no Copilot plan | Create a new token (§6), reset the secret, then `gh run rerun <id> --failed`. Test locally: `COPILOT_GITHUB_TOKEN=<pat> copilot -p "hi"` |
| Run is green but `agent` was **skipped** | `pre_activation` decided not to run: draft PR, bot author, or actor without write access | Mark the PR ready for review |
| No run after pushing new commits | Expected: these workflows don't trigger on pushes | Edit the PR description, or close and reopen the PR |
| Compile warns "safe update mode detected unapproved changes" | A new secret or action was added | Review it, then compile with `--approve` (§5) |

To check the workflows' status and recent runs, use `gh aw status` and `gh aw logs <workflow>`. To investigate a single run, use `gh aw audit <run-id>`.
