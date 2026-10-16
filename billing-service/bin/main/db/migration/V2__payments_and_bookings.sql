-- Payments through Stripe (#83, #86, #88) and the booking snapshot that
-- deposits are priced from.

-- What billing knows about a booking, from scheduler's AppointmentBooked /
-- AppointmentCancelled events (architecture-design §3d), or fetched from
-- scheduler when a deposit request beats the event here. A local read model
-- (§5b): never a cross-database FK.
CREATE TABLE appointment_ref (
    appointment_id BIGINT PRIMARY KEY,
    customer_id    BIGINT NOT NULL,
    vehicle_id     BIGINT NOT NULL,
    dealership_id  BIGINT NOT NULL,
    currency       char(3) NOT NULL CHECK (currency = upper(currency)),
    -- The booking's estimate (sum of the booked services' fees); the deposit
    -- policy is a share of it. The bill itself is priced at completion.
    estimate_minor BIGINT NOT NULL CHECK (estimate_minor >= 0),
    status         varchar(20) NOT NULL CHECK (status IN ('BOOKED','CANCELLED','COMPLETED')),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_appointment_ref_customer ON appointment_ref (customer_id);

-- Shown on the bill (#87): the card's brand and last four digits, never more.
-- failure_message is Stripe's customer-facing decline text for the UI.
ALTER TABLE payment
    ADD COLUMN card_brand      varchar(30),
    ADD COLUMN card_last4      varchar(4),
    ADD COLUMN failure_message varchar(500);
