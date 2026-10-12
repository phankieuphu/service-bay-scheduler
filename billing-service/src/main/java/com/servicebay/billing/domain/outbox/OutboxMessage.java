package com.servicebay.billing.domain.outbox;

import java.util.Objects;

/**
 * An event to publish, written in the same transaction as the state change
 * it describes; the outbox relay publishes it to Kafka afterwards.
 *
 * @param payload the JSON message value
 */
public record OutboxMessage(Long id, String topic, String key, String payload) {

    public OutboxMessage {
        Objects.requireNonNull(topic, "topic");
        Objects.requireNonNull(key, "key");
        Objects.requireNonNull(payload, "payload");
    }

    public static OutboxMessage of(String topic, String key, String payload) {
        return new OutboxMessage(null, topic, key, payload);
    }
}
