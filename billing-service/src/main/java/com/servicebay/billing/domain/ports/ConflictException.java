package com.servicebay.billing.domain.ports;

/**
 * A write conflicted with existing data: a unique constraint (a second live
 * bill for an appointment, a reused idempotency key) or a stale version.
 */
public class ConflictException extends RuntimeException {

    public ConflictException(String message, Throwable cause) {
        super(message, cause);
    }

    public ConflictException(String message) {
        super(message);
    }
}
