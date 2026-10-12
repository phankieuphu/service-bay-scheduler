package com.servicebay.billing.domain.ports;

/**
 * Deduplicates at-least-once input (Kafka events, Stripe webhooks). Call it
 * in the same transaction as the state change the event causes.
 */
public interface ProcessedEventRepository {

    /**
     * Records (source, eventId) and returns true, or returns false if it was
     * already recorded, meaning the event is a redelivery to skip.
     */
    boolean markProcessed(String source, String eventId);
}
