-- billing_db: deposits, bills and payments, with Stripe as the provider.
--
-- Money is always integer minor units plus an ISO 4217 currency
-- (domain.money.Money), never numeric/float. Minor units follow Stripe's
-- convention, so zero-decimal currencies such as VND have no cents.
--
-- appointment_id, customer_id and vehicle_id belong to scheduler-,
-- customer- and vehicle-service: plain columns, no FKs (architecture-design
-- §5a).

-- One bill per appointment, created by the first money event: a deposit
-- (the bill starts OPEN, not yet priced) or ServiceCompleted (it starts
-- PENDING). Prices are copied onto the bill when it's priced (Pattern c,
-- architecture-design §5c), so a bill never changes because a catalog price
-- or a warranty changed later.
CREATE TABLE bill (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    appointment_id      BIGINT NOT NULL,
    customer_id         BIGINT NOT NULL,
    vehicle_id          BIGINT NOT NULL,
    currency            char(3) NOT NULL CHECK (currency = upper(currency)),
    warranty_fee_minor  BIGINT NOT NULL DEFAULT 0 CHECK (warranty_fee_minor >= 0),
    service_fee_minor   BIGINT NOT NULL DEFAULT 0 CHECK (service_fee_minor >= 0),
    material_cost_minor BIGINT NOT NULL DEFAULT 0 CHECK (material_cost_minor >= 0),
    total_minor         BIGINT GENERATED ALWAYS AS
                            (warranty_fee_minor + service_fee_minor + material_cost_minor) STORED,
    -- OPEN: deposit only, not priced. PENDING: priced, balance due.
    -- PARTIALLY_PAID / PAID: by the sum of captured payments. VOID: cancelled.
    status              varchar(20) NOT NULL
                        CHECK (status IN ('OPEN','PENDING','PARTIALLY_PAID','PAID','VOID')),
    -- When the bill was priced (OPEN -> PENDING). NULL while OPEN.
    finalized_at        timestamptz,
    version             BIGINT NOT NULL DEFAULT 0,  -- JPA optimistic lock
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'OPEN') = (finalized_at IS NULL) OR status = 'VOID')
);

-- At most one live bill per appointment. A VOID bill doesn't count, so an
-- appointment can get a fresh bill after its bill was voided.
CREATE UNIQUE INDEX uq_bill_live_appointment ON bill (appointment_id) WHERE status <> 'VOID';
CREATE INDEX idx_bill_customer ON bill (customer_id);

-- What was charged, copied at pricing time (architecture-design §5c).
CREATE TABLE bill_line (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bill_id          BIGINT NOT NULL REFERENCES bill(id) ON DELETE CASCADE,
    kind             varchar(20) NOT NULL CHECK (kind IN ('SERVICE','MATERIAL','WARRANTY_FEE')),
    source_id        BIGINT,           -- service_id / material_id in their owning service; NULL for WARRANTY_FEE
    name             varchar(255) NOT NULL,
    quantity         int NOT NULL CHECK (quantity > 0),
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor >= 0),
    line_total_minor BIGINT GENERATED ALWAYS AS (quantity * unit_price_minor) STORED,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_bill_line_bill ON bill_line (bill_id);

-- A deposit (paid at booking, against the OPEN bill) or the final balance.
-- Stripe is the source of truth for success: status moves on webhooks.
CREATE TABLE payment (
    id                       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bill_id                  BIGINT NOT NULL REFERENCES bill(id) ON DELETE RESTRICT,
    type                     varchar(20) NOT NULL CHECK (type IN ('DEPOSIT','FINAL')),
    amount_minor             BIGINT NOT NULL CHECK (amount_minor > 0),
    currency                 char(3) NOT NULL CHECK (currency = upper(currency)),
    status                   varchar(20) NOT NULL
                             CHECK (status IN ('REQUIRES_PAYMENT','PROCESSING','SUCCEEDED','FAILED',
                                               'CANCELED','REFUNDED','PARTIALLY_REFUNDED')),
    stripe_payment_intent_id varchar(255) UNIQUE,
    -- The client's Idempotency-Key (or one derived from it); also passed to
    -- Stripe, so a retried request can't create a second PaymentIntent.
    idempotency_key          varchar(255) NOT NULL UNIQUE,
    failure_code             varchar(100),
    succeeded_at             timestamptz,
    version                  BIGINT NOT NULL DEFAULT 0,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_bill ON payment (bill_id);

CREATE TABLE refund (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    payment_id       BIGINT NOT NULL REFERENCES payment(id) ON DELETE RESTRICT,
    amount_minor     BIGINT NOT NULL CHECK (amount_minor > 0),
    currency         char(3) NOT NULL CHECK (currency = upper(currency)),
    stripe_refund_id varchar(255) UNIQUE,
    status           varchar(20) NOT NULL
                     CHECK (status IN ('PENDING','SUCCEEDED','FAILED','CANCELED')),
    reason           varchar(255),
    version          BIGINT NOT NULL DEFAULT 0,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_refund_payment ON refund (payment_id);

-- Stripe Customer per Service Bay customer, created lazily.
CREATE TABLE stripe_customer (
    customer_id        BIGINT PRIMARY KEY,
    stripe_customer_id varchar(255) NOT NULL UNIQUE,
    created_at         timestamptz NOT NULL DEFAULT now()
);

-- Deduplicates at-least-once input: Kafka events (source = topic) and
-- Stripe webhooks (source = 'stripe'), keyed by their event id. A handler
-- inserts here in the same transaction as its state change.
CREATE TABLE processed_event (
    source       varchar(255) NOT NULL,
    event_id     varchar(255) NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source, event_id)
);

-- Transactional outbox, the same shape as the Go services'.
CREATE TABLE outbox_message (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic        varchar(255) NOT NULL,
    message_key  varchar(255) NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX idx_outbox_message_unpublished ON outbox_message (id) WHERE published_at IS NULL;
