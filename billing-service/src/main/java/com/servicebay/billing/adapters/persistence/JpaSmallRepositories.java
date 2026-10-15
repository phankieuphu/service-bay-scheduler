package com.servicebay.billing.adapters.persistence;

import com.servicebay.billing.adapters.persistence.SpringDataRepositories.OutboxJpa;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.money.Money;
import com.servicebay.billing.domain.outbox.OutboxMessage;
import com.servicebay.billing.domain.ports.AppointmentRefRepository;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.OutboxRepository;
import com.servicebay.billing.domain.ports.ProcessedEventRepository;
import com.servicebay.billing.domain.ports.StripeCustomerRepository;
import java.util.Optional;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Propagation;
import org.springframework.transaction.annotation.Transactional;

/** The adapters too small for a file each. */
final class JpaSmallRepositories {

    private JpaSmallRepositories() {
    }

    @Component
    static class JpaOutboxRepository implements OutboxRepository {

        private final OutboxJpa outbox;

        JpaOutboxRepository(OutboxJpa outbox) {
            this.outbox = outbox;
        }

        // MANDATORY: an outbox row written outside the state change's
        // transaction defeats the point of the outbox.
        @Override
        @Transactional(propagation = Propagation.MANDATORY)
        public void append(OutboxMessage message) {
            OutboxMessageEntity e = new OutboxMessageEntity();
            e.topic = message.topic();
            e.messageKey = message.key();
            e.payload = message.payload();
            outbox.save(e);
        }
    }

    // Plain SQL through JdbcClient, which joins the JPA transaction: neither
    // table needs an entity.
    @Component
    static class JdbcProcessedEventRepository implements ProcessedEventRepository {

        private final JdbcClient jdbc;

        JdbcProcessedEventRepository(JdbcClient jdbc) {
            this.jdbc = jdbc;
        }

        @Override
        @Transactional(propagation = Propagation.MANDATORY)
        public boolean markProcessed(String source, String eventId) {
            return jdbc.sql("insert into processed_event (source, event_id) values (:source, :eventId) on conflict do nothing")
                    .param("source", source)
                    .param("eventId", eventId)
                    .update() == 1;
        }
    }

    @Component
    static class JdbcAppointmentRefRepository implements AppointmentRefRepository {

        private final JdbcClient jdbc;

        JdbcAppointmentRefRepository(JdbcClient jdbc) {
            this.jdbc = jdbc;
        }

        @Override
        @Transactional
        public void upsert(AppointmentRef ref) {
            jdbc.sql("""
                    insert into appointment_ref
                        (appointment_id, customer_id, vehicle_id, dealership_id, currency, estimate_minor, status, updated_at)
                    values (:id, :customerId, :vehicleId, :dealershipId, :currency, :estimate, :status, now())
                    on conflict (appointment_id) do update set
                        customer_id = excluded.customer_id, vehicle_id = excluded.vehicle_id,
                        dealership_id = excluded.dealership_id, currency = excluded.currency,
                        estimate_minor = excluded.estimate_minor, status = excluded.status, updated_at = now()
                    """)
                    .param("id", ref.appointmentId())
                    .param("customerId", ref.customerId())
                    .param("vehicleId", ref.vehicleId())
                    .param("dealershipId", ref.dealershipId())
                    .param("currency", ref.currency())
                    .param("estimate", ref.estimate().amountMinor())
                    .param("status", ref.status().name())
                    .update();
        }

        @Override
        @Transactional(readOnly = true)
        public Optional<AppointmentRef> findById(long appointmentId) {
            return jdbc.sql("select * from appointment_ref where appointment_id = :id")
                    .param("id", appointmentId)
                    .query((rs, n) -> new AppointmentRef(rs.getLong("appointment_id"), rs.getLong("customer_id"),
                            rs.getLong("vehicle_id"), rs.getLong("dealership_id"),
                            Money.of(rs.getLong("estimate_minor"), rs.getString("currency")),
                            AppointmentRef.Status.valueOf(rs.getString("status"))))
                    .optional();
        }
    }

    @Component
    static class JdbcStripeCustomerRepository implements StripeCustomerRepository {

        private final JdbcClient jdbc;

        JdbcStripeCustomerRepository(JdbcClient jdbc) {
            this.jdbc = jdbc;
        }

        @Override
        @Transactional(readOnly = true)
        public Optional<String> findStripeCustomerId(long customerId) {
            return jdbc.sql("select stripe_customer_id from stripe_customer where customer_id = :customerId")
                    .param("customerId", customerId)
                    .query(String.class)
                    .optional();
        }

        // A plain insert, so a second row for the customer is a unique
        // violation (a concurrent create won) rather than a silent overwrite.
        @Override
        @Transactional
        public void save(long customerId, String stripeCustomerId) {
            try {
                jdbc.sql("insert into stripe_customer (customer_id, stripe_customer_id) values (:customerId, :stripeId)")
                        .param("customerId", customerId)
                        .param("stripeId", stripeCustomerId)
                        .update();
            } catch (DataIntegrityViolationException e) {
                throw new ConflictException("customer " + customerId + " already has a Stripe customer", e);
            }
        }
    }
}
