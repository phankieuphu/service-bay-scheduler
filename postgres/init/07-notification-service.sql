-- =============================================================
-- notification_db — notification-service
-- =============================================================
-- Not present in ServiceHub.sql. architecture-design.md §1 has
-- notification-service own an "Outbox/delivery log" — this is it.
-- It originates nothing; every row here exists because some other
-- service's event landed on the bus (§3: BookingConfirmed,
-- MidServiceApprovalRequested, PaymentReceived, ...).
-- =============================================================

\connect notification_db

CREATE TABLE outbox_delivery (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type    varchar(100) NOT NULL,   -- e.g. 'BookingConfirmed'
    recipient_ref BIGINT NOT NULL,         -- customer.id or technician.id, no FK
    channel       varchar(20) NOT NULL CHECK (channel IN ('EMAIL','SMS','PUSH')),
    payload       jsonb NOT NULL,
    status        varchar(20) NOT NULL DEFAULT 'PENDING'
                  CHECK (status IN ('PENDING','SENT','FAILED')),
    sent_at       timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_outbox_status ON outbox_delivery(status, created_at);
