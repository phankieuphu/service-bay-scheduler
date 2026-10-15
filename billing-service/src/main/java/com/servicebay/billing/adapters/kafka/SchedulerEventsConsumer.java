package com.servicebay.billing.adapters.kafka;

import com.servicebay.billing.application.Bookings;
import com.servicebay.billing.application.Bookings.CompletedService;
import com.servicebay.billing.application.Bookings.Completion;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.money.Money;
import java.math.BigDecimal;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;
import tools.jackson.core.JacksonException;
import tools.jackson.databind.JsonNode;
import tools.jackson.databind.json.JsonMapper;

/**
 * scheduler-service's events (architecture-design §3d, §3b). Parsing lives
 * here; what to do with them lives in {@link Bookings}, which is idempotent on
 * (topic, event_id).
 */
@Component
class SchedulerEventsConsumer {

    static final String TOPIC_BOOKED = "scheduler.appointment.booked.v1";
    static final String TOPIC_CANCELLED = "scheduler.appointment.cancelled.v1";
    static final String TOPIC_COMPLETED = "scheduler.appointment.service-completed.v1";

    private static final JsonMapper JSON = JsonMapper.builder().build();

    private final Bookings bookings;
    private final String defaultCurrency;

    SchedulerEventsConsumer(Bookings bookings,
            @org.springframework.beans.factory.annotation.Value("${billing.currency}") String defaultCurrency) {
        this.bookings = bookings;
        this.defaultCurrency = defaultCurrency;
    }

    @KafkaListener(topics = {TOPIC_BOOKED, TOPIC_CANCELLED, TOPIC_COMPLETED})
    void onMessage(ConsumerRecord<String, String> record) {
        JsonNode root = parse(record);
        String eventId = required(root, "event_id", record).asString();
        switch (record.topic()) {
            case TOPIC_BOOKED -> bookings.booked(record.topic(), eventId, appointment(root, record, AppointmentRef.Status.BOOKED));
            case TOPIC_CANCELLED -> bookings.cancelled(record.topic(), eventId,
                    appointment(root, record, AppointmentRef.Status.CANCELLED));
            case TOPIC_COMPLETED -> bookings.completed(record.topic(), eventId, completion(root, record));
            default -> throw new UnreadableEventException("unexpected topic " + record.topic(), null);
        }
    }

    /** AppointmentBooked / AppointmentCancelled: payload under "appointment". */
    private AppointmentRef appointment(JsonNode root, ConsumerRecord<String, String> record, AppointmentRef.Status status) {
        JsonNode a = required(root, "appointment", record);
        // Events from before scheduler stamped a currency carry none: they're in
        // the catalog's currency, which billing is configured with.
        String currency = a.path("currency").asString(defaultCurrency);
        try {
            return new AppointmentRef(required(a, "id", record).asLong(), required(a, "customer_id", record).asLong(),
                    required(a, "vehicle_id", record).asLong(), a.path("dealership_id").asLong(0),
                    Money.ofMajor(new BigDecimal(required(a, "total_amount", record).asString()), currency), status);
        } catch (IllegalArgumentException e) {
            throw new UnreadableEventException("bad appointment in " + record.topic() + "@" + record.offset(), e);
        }
    }

    /** ServiceCompleted: flat (§3b) plus the priced fields (§3e). */
    private Completion completion(JsonNode root, ConsumerRecord<String, String> record) {
        String currency = required(root, "currency", record).asString();
        List<CompletedService> services = new ArrayList<>();
        for (JsonNode s : required(root, "services", record)) {
            services.add(new CompletedService(required(s, "service_id", record).asLong(),
                    required(s, "name", record).asString(), s.path("quantity").asInt(1),
                    required(s, "unit_price_minor", record).asLong()));
        }
        try {
            return new Completion(required(root, "appointment_id", record).asLong(),
                    required(root, "customer_id", record).asLong(), required(root, "vehicle_id", record).asLong(),
                    root.path("dealership_id").asLong(0), Instant.parse(required(root, "completed_at", record).asString()),
                    Money.zero(currency).currency(), services);
        } catch (RuntimeException e) {
            throw new UnreadableEventException("bad ServiceCompleted at " + record.topic() + "@" + record.offset(), e);
        }
    }

    private static JsonNode parse(ConsumerRecord<String, String> record) {
        try {
            return JSON.readTree(record.value());
        } catch (JacksonException | IllegalArgumentException e) {
            throw new UnreadableEventException("undecodable record at " + record.topic() + "@" + record.offset(), e);
        }
    }

    private static JsonNode required(JsonNode node, String field, ConsumerRecord<String, String> record) {
        JsonNode value = node.path(field);
        if (value.isMissingNode() || value.isNull()) {
            throw new UnreadableEventException(record.topic() + "@" + record.offset() + " has no " + field, null);
        }
        return value;
    }
}
