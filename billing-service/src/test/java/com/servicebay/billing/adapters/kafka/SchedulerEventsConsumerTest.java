package com.servicebay.billing.adapters.kafka;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;

import com.servicebay.billing.application.Bookings;
import com.servicebay.billing.application.Bookings.CompletedService;
import com.servicebay.billing.application.Bookings.Completion;
import com.servicebay.billing.domain.booking.AppointmentRef;
import com.servicebay.billing.domain.money.Money;
import java.time.Instant;
import java.util.List;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

class SchedulerEventsConsumerTest {

    private final Bookings bookings = mock(Bookings.class);
    private final SchedulerEventsConsumer consumer = new SchedulerEventsConsumer(bookings, "USD");

    private static ConsumerRecord<String, String> record(String topic, String value) {
        return new ConsumerRecord<>(topic, 0, 3L, "12", value);
    }

    @Test
    void bookedEventBecomesABookingSnapshot() {
        consumer.onMessage(record(SchedulerEventsConsumer.TOPIC_BOOKED, """
                {"event_id":"e1","event_type":"AppointmentBooked","appointment":{"id":12,"dealership_id":1,
                 "customer_id":1001,"vehicle_id":7,"status":"BOOKED","total_amount":"105.50","currency":"USD",
                 "services":[{"service_id":1,"name":"Oil and filter change","fee":"45.00"}]}}"""));

        verify(bookings).booked(SchedulerEventsConsumer.TOPIC_BOOKED, "e1",
                new AppointmentRef(12, 1001, 7, 1, Money.of(10_550, "USD"), AppointmentRef.Status.BOOKED));
    }

    @Test
    void anEventWithoutCurrencyUsesTheCatalogs() {
        consumer.onMessage(record(SchedulerEventsConsumer.TOPIC_CANCELLED, """
                {"event_id":"e2","appointment":{"id":12,"customer_id":1001,"vehicle_id":7,"total_amount":"45.00"}}"""));

        verify(bookings).cancelled(eq(SchedulerEventsConsumer.TOPIC_CANCELLED), eq("e2"),
                eq(new AppointmentRef(12, 1001, 7, 0, Money.of(4_500, "USD"), AppointmentRef.Status.CANCELLED)));
    }

    @Test
    void serviceCompletedCarriesPricedLines() {
        consumer.onMessage(record(SchedulerEventsConsumer.TOPIC_COMPLETED, """
                {"event_id":"e3","event_type":"ServiceCompleted","appointment_id":12,"vehicle_id":7,"customer_id":1001,
                 "dealership_id":1,"completed_at":"2026-10-20T04:00:00.000Z","currency":"usd",
                 "services":[{"service_id":1,"name":"Oil change","quantity":1,"fee":"45.00","unit_price_minor":4500}]}"""));

        ArgumentCaptor<Completion> completion = ArgumentCaptor.forClass(Completion.class);
        verify(bookings).completed(eq(SchedulerEventsConsumer.TOPIC_COMPLETED), eq("e3"), completion.capture());
        assertThat(completion.getValue()).isEqualTo(new Completion(12, 1001, 7, 1, Instant.parse("2026-10-20T04:00:00Z"),
                "USD", List.of(new CompletedService(1, "Oil change", 1, 4_500))));
    }

    @Test
    void unreadableRecordsAreNotRetried() {
        assertThatThrownBy(() -> consumer.onMessage(record(SchedulerEventsConsumer.TOPIC_BOOKED, "not json")))
                .isInstanceOf(UnreadableEventException.class);
        assertThatThrownBy(() -> consumer.onMessage(record(SchedulerEventsConsumer.TOPIC_COMPLETED,
                "{\"event_id\":\"e4\",\"appointment_id\":12}"))).isInstanceOf(UnreadableEventException.class);
        assertThatThrownBy(() -> consumer.onMessage(record(SchedulerEventsConsumer.TOPIC_BOOKED,
                "{\"event_id\":\"e5\",\"appointment\":{\"id\":1,\"customer_id\":1,\"vehicle_id\":1,\"total_amount\":\"1.234\"}}")))
                .isInstanceOf(UnreadableEventException.class);
        verifyNoInteractions(bookings);
    }
}
