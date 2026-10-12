# billing-service

Deposits, bills and payments for Service Bay, with **Stripe** as the payment provider. It's the platform's only Java service (Spring Boot 4, Java 25). See [technical-requirement.md](../docs/technical-requirement.md) for why, and [#35](https://github.com/phankieuphu/service-bay-scheduler/issues/35) for the plan.

## Layout

Same hexagonal shape as the Go services:

| Package (`com.servicebay.billing`) | Holds |
|---|---|
| `domain` | Entities, ports (interfaces), services. **No Spring, JPA or Stripe imports.** `domain.money.Money` is the only way to hold an amount. |
| `adapters.http` | Controllers and DTOs |
| `adapters.persistence` | JPA entities and repositories implementing the domain's ports |
| `adapters.kafka` | Outbox relay and consumers |
| `adapters.stripe` | The only package that touches `stripe-java` |
| `application` | Spring wiring |

**Money:** a `long` count of minor units plus an ISO 4217 code, never `double`. Minor units follow Stripe's convention. Zero-decimal currencies (VND, JPY, …) have no cents, so `Money.of(150000, "VND")` is 150 000 ₫. `Money.ofMajor` refuses amounts with more precision than the currency has.

**Schema:** Flyway migrations in `src/main/resources/db/migration`, applied on startup. Hibernate only validates against them (`ddl-auto: validate`).

## Run

```bash
cd billing-service
./gradlew bootRun        # needs Postgres (billing_db) and Kafka; env in .env.example
./gradlew test           # unit tests + Testcontainers integration tests (needs Docker)
./gradlew bootJar        # build/libs/billing-service-<version>.jar
```

The Gradle toolchain pins Java 25 and downloads it if it isn't installed. Gradle itself runs on any JDK 17+.

With the whole stack: `docker compose up -d billing-service` (port **8084**).

| Endpoint | |
|---|---|
| `GET /actuator/health/liveness` | the process is up |
| `GET /actuator/health/readiness` | UP once Postgres is reachable |
| `GET /actuator/prometheus` | metrics, scraped by Prometheus |

`server.shutdown=graceful`: on SIGTERM, in-flight requests get up to `SHUTDOWN_TIMEOUT` (15s) to finish.
