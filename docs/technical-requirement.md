# Technical Requirement

Non-functional requirements and technical constraints for Service Bay, derived from `business-requirement.md` (domain/journeys) and `architecture-design.md` (service split). This document adds the *how much / how fast / how available* numbers that architecture-design assumed but didn't quantify.

---

## 1. Tech Stack

| Layer | Choice | Notes |
|---|---|---|
| Language | Go | one binary per service (architecture-design §1); goroutines suit the I/O-bound, high-concurrency profile in §3 |
| Relational store | PostgreSQL 16 | one schema/database per service (architecture-design §7); range types + exclusion constraints used for the booking overlap guarantee (architecture-design §4) |
| Cache | Redis | availability-cache (short TTL, hint not source of truth) + identity-service session store (architecture-design §7) |
| Messaging | Kafka | async event bus (architecture-design §3), already in `docker-compose.yml` |
| Connection pooling | PgBouncer (transaction mode) | required at 100k-concurrent scale — see §4 |
| Deployment | Containers on Kubernetes | one Deployment per service, independent HPA per service (matches the microservices split — no reason to scale dealership-service and report-service together) |
| Gateway | API Gateway (architecture-design §7) | authn termination, rate limiting, per-actor BFF composition |
| Observability | OpenTelemetry traces, Prometheus metrics, structured JSON logs | see §4.7 |

This document does not revisit *which* services exist — that's settled in `architecture-design.md` §1. It defines the bar each of those services (and the Postgres/Redis/Kafka they sit on) must clear.

---

## 2. Scale Targets (given)

| Dimension | Target |
|---|---|
| Concurrent users | 1,000 (baseline) → 100,000 (peak), i.e. a 100x swing the system must absorb without redesign |
| Stored records | ~150,000,000 rows across the Vehicle and Material domains |
| Availability | No single point of failure in any tier (app, cache, DB, broker) |

Everything below turns these three lines into concrete engineering targets.

---

## 3. Traffic & Capacity Planning

These are worked assumptions, not measured numbers — flagged again in §7. They exist so "handle 100k users" has a concrete target to design and load-test against, instead of staying an adjective.

**Request rate.** Not every concurrent user issues a request every second — most concurrency is idle browsing/holding a session. Assuming ~0.2 req/s/user of actual load (typical for a booking-and-browse app, not a chat app):

| Concurrency | Est. peak RPS |
|---|---|
| 1,000 | ~200 |
| 100,000 | ~20,000 |

**Read/write split.** Availability search and catalog/service-history browsing dominate; booking writes, mid-service change requests, and payments are comparatively rare. Target an 85:15 read:write ratio:

| | RPS at 100k concurrency |
|---|---|
| Reads (availability search, catalog, history) | ~17,000 |
| Writes (booking, payment, mid-service change) | ~3,000 |

Implication: scheduler-service's availability-search path is the hottest read path in the system and is exactly what the Redis availability-cache (architecture-design §7) exists for. Booking-write throughput (~3,000 RPS peak) is what the Appointment table's overlap constraint (architecture-design §4) has to sustain without becoming a lock-contention bottleneck — partition or shard `appointment` by `hub_id` if a single Postgres instance can't hold that write rate.

**Record volume.** 150M rows is not evenly split — the business's write pattern makes one table dominant:

| Table | Rough share | Why |
|---|---|---|
| `material_consumption` (dealership-service) | ~100–130M rows | one row per material used per service (business-requirement §8); grows with every completed booking, never with every customer |
| `vehicle` (vehicle-service) | ~10–20M rows | one row per registered vehicle; grows with customer signups, orders of magnitude slower |
| `material` (catalog/inventory, dealership-service) | thousands | reference data, not a scale concern |

`material_consumption` is a transaction log, not a live-lookup table — it should be **range-partitioned by month/quarter** on its creation timestamp so that (a) old partitions can be archived/compressed independently, (b) indexes stay small enough for the planner to keep using them at 100M+ rows, and (c) autovacuum operates per-partition instead of stalling on one 100M-row table. `vehicle` stays a single table with a b-tree on `customer_id` — 10–20M rows doesn't need partitioning, just correct indexing.

---

## 4. Availability (HA)

No component in the path of a user request may be a single point of failure.

| Component | HA mechanism |
|---|---|
| Application services | Stateless Go binaries, ≥2 replicas per service per availability zone, behind the gateway's load balancer; a pod dying loses zero state |
| PostgreSQL | Primary + ≥1 streaming replica per service database, automatic failover (Patroni or equivalent); replicas also absorb read traffic (report-service, vehicle-service history reads) |
| Redis | Sentinel (or Cluster mode if the availability-cache dataset outgrows one node) — a cache-node failure degrades scheduler's read latency, it must never turn into a hard outage since Redis here is explicitly "a hint, not a source of truth" (architecture-design §7) |
| Kafka | Replication factor ≥3, `min.insync.replicas=2`; broker loss doesn't stop event delivery or force a service to fall back to synchronous calls |
| API Gateway | Deployed as ≥2 replicas behind an external load balancer; not a single ingress pod |
| Cross-AZ | All of the above spread across ≥2 availability zones, so one zone's outage degrades capacity, not availability |

**Failure isolation.** Because each service owns its own database (architecture-design §5), one service's Postgres instance being down degrades only that service — e.g. dealership-service down blocks new bookings' candidate lookup, but vehicle-service and billing-service keep serving. This is the payoff of the microservice split showing up as an HA property, not just an org-boundary one.

---

## 5. Scalability

### 5.1 System Scalability (Throughput)

- **Horizontal, per-service.** The microservice split (architecture-design §1) means each service scales on its own HPA against its own metric — scheduler-service scales on booking RPS, report-service barely scales at all (it's a nightly batch job's read path). Don't co-scale services that have unrelated load profiles.
- **Stateless services, pooled connections.** Every Go service sits behind PgBouncer in transaction-pooling mode. At 100k concurrent users a naive one-Postgres-connection-per-request model exhausts Postgres's connection limit (typically a few hundred) long before it exhausts CPU; PgBouncer is what lets thousands of app-level connections multiplex onto a small, stable backend connection count.
- **Read replicas** for read-heavy services (vehicle-service's history reads, report-service's rollups) so read load doesn't compete with the write path on the primary.
- **Cache the hot read path.** scheduler's `GET /availability` is the single highest-QPS endpoint in the system per §3 — this is Redis's job, short TTL, cache-aside, and the source of truth stays Postgres per architecture-design §7.
- **Kafka partitioning.** Partition each topic by the entity id that must preserve order (e.g. `ServiceCompleted` partitioned by `booking_id` or `hub_id`) so consumer parallelism scales with partition count without breaking per-entity ordering.
- **Autoscale on the metric that actually predicts load**, not just CPU — RPS or queue depth for I/O-bound Go services, since Go's goroutine model means CPU often stays low while a service is still saturated on downstream DB/Redis latency.

### 5.2 Business / Domain Scalability (Extensibility)

§5.1 is "handle more traffic." This is the other kind of scale the business actually asked for: when the business adds a new capability — a loyalty program, subscription maintenance plans, a new hub format, a new payment method — the system should absorb it by **adding** a piece, not by reopening and rewriting existing services or tables. "Don't need to remove all and build new" is the requirement; the method below is how the design already delivers it, and what to keep doing as the business grows.

**Method: Domain-Driven Design (DDD) bounded contexts**, which is also *why* the service split in `architecture-design.md` §1 looks the way it does — each service maps to one business capability, not to a technical layer. That mapping is what makes the following four mechanisms available:

1. **New capability → new bounded context (new service + schema), existing ones untouched.** E.g. "loyalty points" or "subscription plans" becomes a new `loyalty-service` that consumes existing events (`ServiceCompleted`, `PaymentReceived`) — scheduler-service and billing-service's code and schema don't change. This is the Open-Closed Principle applied at the system level: existing services stay closed for modification, open for new subscribers.

2. **The event backbone is the extension point.** Because every state change is already published as a domain event (architecture-design §3 — `BookingConfirmed`, `ServiceCompleted`, `PaymentReceived`, …), a new service can subscribe to the history it needs without the producer being changed or even being aware the new consumer exists. This decoupling is what turns "add a feature" into "add a consumer" instead of "modify a producer."

3. **Contract-first, additive-only APIs.** Each service's public API is versioned; a new business rule adds a new field or endpoint, it never repurposes an existing one's meaning. Old consumers keep working unmodified on the existing contract.

4. **Strangler-fig for changes too big to be a clean new service** (e.g. the service catalog outgrows dealership-service and needs to become its own bounded context): run the new implementation alongside the old behind the gateway, shift traffic incrementally, retire the old path once proven. Never a stop-the-world rewrite of a working service.

**Where this has limits.** DDD reduces rewrite risk, it doesn't eliminate judgment: a genuinely new *kind* of business thing (a new actor type, a new top-level domain) still needs a deliberate new-bounded-context decision up front, not a field bolted onto the nearest existing table. That discipline is exactly the lesson architecture-design §1a already draws from removing the standalone `worker` service — don't force new data into a context it doesn't belong to just to avoid the ceremony of creating a new one.

---

## 6. Database Design Method

The database side of "adapt, don't rebuild" is a repeatable modeling process, not ad hoc table creation — so a new business capability gets a schema the same way every time, and existing schemas evolve without downtime.

1. **Conceptual model, straight from the business requirement.** Entities and relationships come from `business-requirement.md` §3 (Core Entities) before any table is drawn — Vehicle, ServiceHub, Technician, Skill, Material, Booking, Bill. Every table traces back to a business noun, not an implementation convenience.
2. **Bounded-context mapping (DDD) → schema-per-service.** Each entity is assigned to exactly one owning service schema, per the authority already established in architecture-design §1 ("owns (data)" column). This is what lets the *database* scale with the business the same way the services do in §5.2: a new capability gets a new schema, never a table added to another service's database.
3. **Logical model: normalize to 3NF first**, within each service's own schema, for OLTP correctness — one source of truth per fact, no update anomalies (e.g. `material_consumption` stores `material_id` + quantity, it doesn't duplicate `material.unit_price`).
4. **Physical model: denormalize only where §3's traffic numbers demand it**, using the three cross-service patterns already defined in architecture-design §5 — reference-by-id (validate once, never re-validate), denormalized read-model kept fresh by events, or copy-the-value-at-write-time — chosen by how fresh the data must be, not applied by default.
5. **Partitioning and indexing follow measured/estimated volume, not habit** — e.g. §3's decision to range-partition `material_consumption` by month comes directly from its projected row count and write pattern; `vehicle` at 10–20M rows stays a single indexed table because it doesn't need partitioning yet.
6. **Schema evolution via expand-contract (parallel change), never destructive-first migrations:**
   - *Expand* — add the new nullable column/table, deploy.
   - *Migrate* — backfill existing rows, dual-write from the application.
   - *Contract* — cut reads over to the new shape, then drop the old column/table in a later, separate migration.

   This is the schema-level version of §5.2's "adapt, don't rewrite": a business change (new field, new relationship, a renamed concept) is absorbed as a sequence of additive, reversible migrations, never a drop-and-rebuild.

---

## 7. Data Consistency & Correctness

Inherits architecture-design §4 (booking overlap is enforced locally on `appointment`, cross-service candidate checks are advisory) and §5 (reference-by-id / denormalized read-model / copy-at-write-time patterns for cross-service relationships). Additional requirements at scale:

- **Idempotency.** Every write endpoint that can be retried by a client or gateway (booking creation, payment) must accept an idempotency key and dedupe on it — at 100k concurrent users, client retries on timeout are routine, not exceptional.
- **Outbox delivery is at-least-once.** Every Kafka consumer (skill progression, inventory decrement, bill generation, report rollup — architecture-design §6) must be idempotent on the event id, since the outbox-relay pattern guarantees at-least-once, not exactly-once, delivery.
- **Partition-aware constraints.** The overlap/uniqueness constraint on `appointment` (bay/technician/vehicle × time range) must remain correct once that table is partitioned for write-scale — a naive per-partition exclusion constraint doesn't catch overlaps across a partition boundary, so partitioning key choice (e.g. by `hub_id`, not by time) matters for correctness, not just performance.

---

## 8. Assumptions & Open Questions

This document quantifies the three targets given (1k→100k concurrent users, 150M vehicle+material records, HA) using industry-typical ratios, because the business requirement doesn't yet specify request rates or the vehicle/material row split. Flagging alongside `business-requirement.md` §10:

1. **Concurrency → RPS conversion** (§3) assumes ~0.2 req/s per concurrent user and an 85:15 read:write split — both should be replaced with real numbers (or a stated target) once available, e.g. from expected booking volume per hub per day.
2. **150M row split** (§3) assumes `material_consumption` dominates as a transaction log. If materials are tracked more coarsely (stock-level deltas, not per-use rows), the dominant table — and its partitioning strategy — changes.
3. **Multi-region** is out of scope here — §4 covers multi-AZ only. If the business requires geographic failover (not just zone failover), that's a separate, larger design (cross-region Postgres replication topology, Kafka MirrorMaker, etc.) worth its own document rather than folding into HA §4.
4. **RPO/RTO and backup targets** for Postgres (WAL archiving cadence, point-in-time-recovery window) aren't set here — they depend on the business's tolerance for data loss on a failover, which isn't stated in `business-requirement.md`.
5. **§5.2/§6's extensibility method assumes future capabilities stay expressible as new bounded contexts consuming existing events.** A change that requires rewriting an existing service's core invariant (e.g. redefining what a Booking *is*) is a genuine breaking change no method fully absorbs — the goal is minimizing how often that happens, not claiming it never will.
