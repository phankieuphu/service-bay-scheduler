package com.servicebay.billing.domain.ports;

/** A payment provider call failed. kind decides the HTTP status the caller gets. */
public class PaymentProviderException extends RuntimeException {

    public enum Kind {
        /** No keys configured: 503, so the rest of the stack runs without them. */
        NOT_CONFIGURED,
        /** Network error, provider outage, or an API error we didn't expect: 502. */
        UNAVAILABLE,
        /** The intent's state doesn't allow it (e.g. canceling one that already succeeded): 409. */
        CONFLICT,
        /** Bad webhook signature or payload: 400. */
        INVALID_WEBHOOK,
        /** A test-only operation with a live key: 403. */
        TEST_MODE_ONLY
    }

    private final Kind kind;

    public PaymentProviderException(Kind kind, String message, Throwable cause) {
        super(message, cause);
        this.kind = kind;
    }

    public PaymentProviderException(Kind kind, String message) {
        this(kind, message, null);
    }

    public Kind kind() {
        return kind;
    }
}
