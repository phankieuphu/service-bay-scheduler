package com.servicebay.billing.domain.payment;

/** Mirrors the PaymentIntent lifecycle; webhooks move it. */
public enum PaymentStatus {
    REQUIRES_PAYMENT,
    PROCESSING,
    SUCCEEDED,
    FAILED,
    CANCELED,
    REFUNDED,
    PARTIALLY_REFUNDED;

    /**
     * Whether the money was captured. A (partly) refunded payment was still
     * captured; its refunds are counted separately, so it must not drop out
     * of the paid total as well.
     */
    public boolean isCaptured() {
        return this == SUCCEEDED || this == REFUNDED || this == PARTIALLY_REFUNDED;
    }
}
