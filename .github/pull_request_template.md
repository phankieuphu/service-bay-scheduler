<!--
Title format: [area] Short imperative summary
e.g. [scheduler-service] Complete appointments and publish ServiceCompleted
Scale the detail to the change: a docs fix needs a line per section, a cross-service change needs more.
-->

## Summary

<!-- What changed and why. Someone with no prior context should be able to follow it. -->

## Linked issues

<!-- Closes #123 / Refs #123, or say why there is no issue. -->

## Changes

<!-- One bullet per significant area touched. -->

-

**Services / areas touched**

- [ ] customer-service
- [ ] vehicle-service
- [ ] identity-service
- [ ] scheduler-service
- [ ] billing-service
- [ ] frontend
- [ ] infra (`docker-compose.yml`, `k8s/`, `postgres/`, Dockerfiles)
- [ ] observability (Prometheus / Grafana / Loki / Alloy)
- [ ] CI / docs

## How was this tested?

<!-- What you ran and what you checked by hand. Remove lines that don't apply. -->

- [ ] Go: `go build ./... && go vet ./... && go test ./...` in each changed service
- [ ] billing-service: `./gradlew test`
- [ ] frontend: `npm run lint && npm run build`, plus screenshots for UI changes
- [ ] End-to-end: `postman/run-flow.sh` against `docker compose up`
- [ ] Manual steps:

## Risk & rollout

<!-- Tick what applies and explain it below. If none apply, write "Low risk because ...". -->

- [ ] Database model / migration change
- [ ] Kafka topic or event payload change (update `docs/architecture-design.md`)
- [ ] Auth / token handling change
- [ ] New or changed env vars / config (`config/config.go`, `.env.example`, `docker-compose.yml`, `k8s/`)
- [ ] Breaking API change for another service or the frontend

<!-- Rollback plan or anything reviewers should watch after merge. -->

## Checklist

- [ ] New tests are in `_test.go` files. Files ending in `-test.go` are never run.
- [ ] Docs updated (README, `docs/`, `CLAUDE.md`) if behavior or contracts changed
- [ ] Regenerated `.lock.yml` with `gh aw compile --strict --validate --actionlint` if `.github/workflows/*.md` changed
