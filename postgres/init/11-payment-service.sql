-- =============================================================
-- payment_db — payment-service
-- =============================================================
-- payment-service is the only service that talks to Stripe.
-- billing-service owns the bill and its payment ledger; this
-- database only tracks the gateway side of a card payment: which
-- Stripe PaymentIntent belongs to which billing payment, and what
-- state Stripe last reported for it.
--
-- bill_id / billing_payment_id are owned by billing-service (no FK,
-- other database). billing_payment_id is UNIQUE because one billing
-- payment maps to exactly one PaymentIntent; it is also the Stripe
-- idempotency key, so a retried create never charges twice.
-- =============================================================

\connect payment_db

CREATE TABLE payment (
    id                       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bill_id                  BIGINT NOT NULL,
    billing_payment_id       BIGINT NOT NULL UNIQUE,
    customer_id              BIGINT NOT NULL,
    amount                   numeric(12,2) NOT NULL CHECK (amount > 0),
    currency                 varchar(3) NOT NULL,
    stripe_payment_intent_id varchar(255) UNIQUE,
    status                   varchar(20) NOT NULL DEFAULT 'PENDING'
                             CHECK (status IN ('PENDING','PROCESSING','SUCCEEDED','CANCELED')),
    failure_reason           varchar(500),
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_bill ON payment(bill_id);

-- Stripe delivers webhooks at least once and may send the same event
-- concurrently. The handler inserts the event id here in the same
-- transaction as the state change; a duplicate insert means "already
-- handled".
CREATE TABLE stripe_event (
    id          varchar(255) PRIMARY KEY,
    type        varchar(100) NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);

-- Transactional outbox, same shape and relay contract as the Go
-- services (see 01-identity-service.sql).
CREATE TABLE outbox_message (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic        varchar(255) NOT NULL,
    message_key  varchar(255) NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX idx_outbox_message_unpublished ON outbox_message (id) WHERE published_at IS NULL;
