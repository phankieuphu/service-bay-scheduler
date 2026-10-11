# AI Agents

The repository runs four AI agent workflows on GitHub Actions. They review pull requests and label issues and PRs. All of them are built with [GitHub Agentic Workflows](https://github.github.com/gh-aw/) (gh-aw, v0.89.21) and use the default **Copilot** engine, billed to the token owner's GitHub Copilot plan.

None of them can push code, merge, approve, or close anything. Each one can only post the comments and labels listed below.

---

## 1. What runs, and when

| Workflow | Source | Trigger | Writes |
|---|---|---|---|
| PR Description Review | [`pr-description-review.md`](../.github/workflows/pr-description-review.md) | Non-draft PR opened, edited, reopened, or marked ready | One review comment, which replaces its previous one, and one `quality: *` label |
| Issue & PR Labeler | [`issue-pr-labeler.md`](../.github/workflows/issue-pr-labeler.md) | Issue opened, edited, or reopened; non-draft PR opened, reopened, or marked ready | Type, area, and risk labels; `quality: *` on issues |
| PR Code Review | [`pr-code-review.md`](../.github/workflows/pr-code-review.md) | Non-draft PR opened, reopened, or marked ready | Up to 10 inline review comments and one review |
| PR Code Review (`/review`) | [`pr-code-review-command.md`](../.github/workflows/pr-code-review-command.md) | A PR comment or review comment starting with `/review` | Same as PR Code Review |

The two code review workflows share their prompt, tools, network settings, and safe outputs through [`shared/pr-code-review.md`](../.github/workflows/shared/pr-code-review.md). They are split only because gh-aw cannot combine `pull_request` and `slash_command` triggers in one workflow (§6).

All four skip PRs and issues opened by Renovate and Dependabot. PRs from forks are blocked by default.

The code review **does not run on every push**, to keep costs down. Comment `/review` to review again after pushing fixes.

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

The labeler runs for **anyone** who opens an issue (`roles: all`), so outside reports get triaged too. The other workflows only run for people with write access.

---

## 4. PR Code Review

This workflow reads the diff and the surrounding code. For Go changes it can also run `go build`, `go vet`, and `go test` in the affected modules (Go 1.26, with network access limited to the Go module proxy). It looks for, in priority order:

1. **Correctness:** logic errors, nil dereferences, `ports.Err*` sentinels not checked with `errors.Is`, wrong HTTP status mapping.
2. **Transactional outbox:** a state change and its event must be written in the same `txManager.RunInTx` closure. Nothing may publish to Kafka directly from a write path. Repositories must use `database_provider.DBFromContext`.
3. **Concurrency:** a missing `SELECT … FOR UPDATE` row lock, races, goroutine leaks.
4. **Kafka contracts:** a service consuming its own producer topic; payload or envelope changes that break backward compatibility (see [architecture-design.md](architecture-design.md)).
5. **Security:** injection, missing authorization checks, secrets in code or logs, token handling in `identity-service`.
6. **Tests:** new test files must end in `_test.go`. A file ending in `-test.go` is never run.

Every comment must describe a concrete failure scenario and be attached to a changed line. It never comments on style or naming.

The review is submitted as **Request changes** only for a confirmed bug that would cause incorrect behavior, data loss, or a security problem in production. Otherwise it is a **Comment**. A newer review replaces the older one. If a PR touches only docs or generated files, the workflow posts nothing; a `/review` request still gets a short "No issues found".

The Copilot CLI also loads the repository's custom instructions, such as [`AGENTS.md`](../AGENTS.md), so the agent sees the repo's conventions.

---

## 5. How it works

```
.github/workflows/<name>.md      ← you edit this (YAML frontmatter + prompt)
        │  gh aw compile
        ▼
.github/workflows/<name>.lock.yml ← generated GitHub Actions workflow; never edit by hand
```

Every run goes through these jobs:

1. **pre_activation:** checks the actor's role, skips bots, and, for `/review`, checks that the comment starts with the command.
2. **activation:** builds the prompt. The `.md` body is loaded when the run starts, not copied into the lock file, so expression errors in it only show up here (§6). Recompile after any edit anyway; this job also checks that the lock file matches its source.
3. **agent:** runs the GitHub Copilot CLI in a sandbox behind a network firewall, with **read-only** access to `contents`, `issues`, and `pull-requests`. It reads GitHub data with `gh`. The model credential is kept out of the sandbox; only gh-aw's API proxy uses it.
4. **detection:** a second model pass that screens the agent's proposed output for prompt injection, secret leaks, and malicious content.
5. **safe_outputs:** the only job with write permission. It applies the comments, labels, and reviews the agent asked for, enforced against the allowlists and limits in the frontmatter (`safe-outputs:`).

Text in issues, PRs, and comments is treated as data. Each prompt tells the agent to ignore any instructions inside it. Even if an injection got through, the agent could only apply allowed labels or post review comments.

---

## 6. Changing a workflow

```bash
gh extension install github/gh-aw          # once
# edit .github/workflows/<name>.md (or shared/pr-code-review.md)
gh aw compile --strict --validate --actionlint
git add .github/workflows/                 # commit the .md AND the regenerated .lock.yml
```

Rules learned the hard way:

- **Recompile after every change**, including a shared import, and commit the regenerated `.lock.yml` files. The lock file is what Actions actually runs.
- **Use only plain expressions in the prompt body.** `${{ github.event.pull_request.number }}` and `${{ a || b }}` are allowed. A conditional such as `${{ github.event_name == 'issues' && 'issue' || 'pull request' }}` compiles, but every run then fails in the activation job with *"unauthorized GitHub Actions expressions"*.
- **Don't put `pull_request` and `slash_command` triggers in one workflow.** Plain mode refuses to compile. In `strategy: centralized` mode the `/review` check also gates PR events, so the review is silently skipped on every new PR. Use two workflows that share an import, like the code review does.
- **Grant imported permissions in the main workflow.** Permissions in a `shared/*.md` import must also be listed in each workflow that imports it, or compiling fails.
- **New secrets need `--approve`.** Strict mode warns ("safe update mode detected unapproved changes") when a compile adds a secret or action. Check where the secret is used in the `.lock.yml`, then recompile with `--approve` and note it in the PR.
- **Keep `ghes: false`** in [`aw.json`](../.github/workflows/aw.json). `true` makes the compiler use v3 of the artifact actions, which github.com has retired.
- To use a different model, set `model:` in the frontmatter. To use a different engine, change `engine:` in all four main workflows; the engine setting is not shared through the import.

---

## 7. Setup

| Requirement | How |
|---|---|
| `COPILOT_GITHUB_TOKEN` repository secret | A **fine-grained** PAT ([create one, prefilled](https://github.com/settings/personal-access-tokens/new?name=COPILOT_GITHUB_TOKEN&user_copilot_requests=read)): resource owner is **your user account**, and **Account permissions → Copilot Requests: Read**. The owner needs an active paid Copilot plan (Pro or Pro+; see [settings/copilot](https://github.com/settings/copilot)). Copy it, then run `pbpaste \| gh aw secrets set COPILOT_GITHUB_TOKEN`, or set it under Settings → Secrets and variables → Actions. Don't use `--value`, which saves the token in your shell history. |
| Actions enabled | Settings → Actions → General: allow GitHub Actions, including `github/gh-aw-actions/*` and the pinned `actions/*`. |
| Labels (optional) | GitHub creates missing labels in grey on first use. Create them up front with `gh label create "quality: good" --color 0e8a16` and so on to choose colours. |

The token is used only to call the model. Comments and labels are written through the default `GITHUB_TOKEN`. OAuth tokens (`gho_…`, e.g. from `gh auth token`) are rejected at startup.

To use Claude instead, add `engine: claude` to all four main workflows, set an `ANTHROPIC_API_KEY` secret, and recompile with `--approve`. The key must be created **inside a Console workspace**: a key without a workspace fails with *"not scoped to a workspace"*. API use needs prepaid Anthropic credits, separate from a claude.ai subscription.

---

## 8. Troubleshooting

Open the run under **Actions**, or run `gh run view <id> --log-failed`.

| Symptom | Cause | Fix |
|---|---|---|
| `activation` fails at *Interpolate variables and render templates* with "unauthorized GitHub Actions expressions" | A compound or conditional `${{ }}` in the prompt body | Use only plain or `a \|\| b` expressions (§6) |
| `activation` fails at *Validate COPILOT_GITHUB_TOKEN secret* | Secret missing or empty | Set it (§7) |
| `agent` fails at *Execute GitHub Copilot CLI* with "models fetch returned 401" and "model alias resolution failed" | The token was rejected: it isn't fine-grained, lacks **Copilot Requests: Read**, is owned by an org, has expired, or its owner has no Copilot plan | Create a new token (§7), reset the secret, then `gh run rerun <id> --failed`. Test locally: `COPILOT_GITHUB_TOKEN=<pat> copilot -p "hi"` |
| Code review run is green but `agent` was **skipped** | `pre_activation` decided not to run: draft PR, bot author, actor without write access, or a comment that doesn't start with `/review` | Mark the PR ready, or comment `/review` |
| No run after pushing new commits | Expected: these workflows don't trigger on pushes | Comment `/review`, or close and reopen the PR |
| Compile warns "safe update mode detected unapproved changes" | A new secret or action was added | Review it, then compile with `--approve` (§6) |

To check the workflows' status and recent runs, use `gh aw status` and `gh aw logs <workflow>`. To investigate a single run, use `gh aw audit <run-id>`.
